// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//=== High-level schema ===
// Each playground bundle is stored once as a BundleData in the bundle_data
// table. The BundleData contains the json string corresponding to the bundle
// files, and is indexed by the hash of the json.
//
// Links to a BundleData are stored in BundleLinks. There can be multiple
// BundleLinks corresponding to a single BundleData. BundleLinks are indexed by
// a unique id, and contain the hash of the BundleData that they correspond to.
//
// The additional layer of indirection provided by BundleLinks allows storing
// identical bundles more efficiently and makes the bundle Id independent of
// its contents, allowing implementation of change history, sharing, expiration
// etc.
//
// Each bundle save request first generates and stores a new BundleLink object,
// and will store a new BundleData only if it does not already exist in the
// database.
//
// Note: If bundles larger than ~1 MiB are to be stored, the max_allowed_packed
// SQL connection parameter must be increased.
//
// TODO(ivanpi): Normalize the Json (e.g. file order).

package storage

import (
	crand "crypto/rand"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"v.io/x/playground/lib/hash"
)

var (
	// Error returned when requested item is not found in the database.
	ErrNotFound = errors.New("Not found")

	// Error returned when retries are exhausted.
	errTooManyRetries = errors.New("Too many retries")

	// Error returned from a transaction callback to trigger a rollback and
	// retry. Other errors cause a rollback and abort.
	errRetryTransaction = errors.New("Retry transaction")
)

//////////////////////////////////////////
// SqlData type definitions

type BundleData struct {
	// Raw SHA256 of the bundle contents
	Hash []byte `db:"hash"` // primary key
	// The bundle contents
	Json string `db:"json"`
}

type BundleLink struct {
	// 64-byte printable ASCII string
	Id string `db:"id"` // primary key
	// Raw SHA256 of the bundle contents
	Hash []byte `db:"hash"` // foreign key => BundleData.Hash
	// Link record creation time
	CreatedAt time.Time `db:"created_at"`
}

///////////////////////////////////////
// DB read-only methods

// TODO(nlacasse): Use prepared statements, otherwise we have an extra
// round-trip to the db, which is slow on cloud sql.

func getBundleLinkById(q sqlx.Queryer, id string) (*BundleLink, error) {
	bLink := BundleLink{}
	if err := sqlx.Get(q, &bLink, "SELECT * FROM bundle_link WHERE id=?", id); err != nil {
		if err == sql.ErrNoRows {
			err = ErrNotFound
		}
		return nil, err
	}
	return &bLink, nil
}

func getBundleDataByHash(q sqlx.Queryer, hash []byte) (*BundleData, error) {
	bData := BundleData{}
	if err := sqlx.Get(q, &bData, "SELECT * FROM bundle_data WHERE hash=?", hash); err != nil {
		if err == sql.ErrNoRows {
			err = ErrNotFound
		}
		return nil, err
	}
	return &bData, nil
}

// GetBundleDataByLinkId returns a BundleData object linked to by a BundleLink
// with a particular id.
// Note: This can fail if the bundle is deleted between fetching BundleLink
// and BundleData. However, it is highly unlikely, costly to mitigate (using
// a serializable transaction), and unimportant (error 500 instead of 404).
func GetBundleDataByLinkId(id string) (*BundleData, error) {
	bLink, err := getBundleLinkById(dbRead, id)
	if err != nil {
		return nil, err
	}
	bData, err := getBundleDataByHash(dbRead, bLink.Hash)
	if err != nil {
		return nil, err
	}
	return bData, nil
}

////////////////////////////////////
// DB write methods

func storeBundleData(ext sqlx.Ext, bData *BundleData) error {
	_, err := sqlx.NamedExec(ext, "INSERT INTO bundle_data (hash, json) VALUES (:hash, :json)", bData)
	return err
}

func storeBundleLink(ext sqlx.Ext, bLink *BundleLink) error {
	_, err := sqlx.NamedExec(ext, "INSERT INTO bundle_link (id, hash) VALUES (:id, :hash)", bLink)
	return err
}

// StoreBundleLinkAndData creates a new bundle data for a given json byte slice
// if one does not already exist. It will create a new bundle link pointing to
// that data. All DB access is done in a transaction, which will retry up to 3
// times. Both the link and the data are returned, or an error if one occured.
func StoreBundleLinkAndData(json []byte) (bLink *BundleLink, bData *BundleData, retErr error) {
	bHashRaw := hash.Raw(json)
	bHash := bHashRaw[:]

	// Attempt transaction up to 3 times.
	runInTransaction(3, func(tx *sqlx.Tx) error {
		// Generate a random id for the bundle link.
		id, err := randomLink(bHash)
		if err != nil {
			return fmt.Errorf("Error creaking link id: %v", err)
		}

		// Check if bundle link with this id already exists in DB.
		if _, err := getBundleLinkById(tx, id); err == nil {
			// Bundle was found. Retry with new id.
			return errRetryTransaction
		} else if err != ErrNotFound {
			return fmt.Errorf("Error getting bundle link: %v", err)
		}

		// Check if bundle data with this hash already exists in DB.
		bData, err = getBundleDataByHash(tx, bHash)
		if err != nil {
			if err != ErrNotFound {
				return fmt.Errorf("Error getting bundle data: %v", err)
			}

			// Bundle does not exist in DB. Store it.
			bData = &BundleData{
				Hash: bHash,
				Json: string(json),
			}
			if err = storeBundleData(tx, bData); err != nil {
				return fmt.Errorf("Error storing bundle data: %v", err)
			}
		}

		// Store the bundle link.
		bLink = &BundleLink{
			Id:   id,
			Hash: bHash,
		}
		if err = storeBundleLink(tx, bLink); err != nil {
			return fmt.Errorf("Error storing bundle link: %v", err)
		}

		return nil
	})

	return
}

//////////////////////////////////////////
// Transaction support

// Runs function txf inside a SQL transaction. txf should only use the database
// handle passed to it, which shares the prepared transaction cache with the
// original handle. If txf returns nil, the transaction is committed.
// Otherwise, it is rolled back.
// txf is retried at most maxRetries times, with a fresh transaction for every
// attempt, until the commit is successful. txf should not have side effects
// that could affect subsequent retries (apart from database operations, which
// are rolled back).
// If the error returned from txf is errRetryTransaction, txf is retried as if
// the commit failed. Otherwise, txf is not retried, and RunInTransaction
// returns the error.
// In rare cases, txf may be retried even if the transaction was successfully
// committed (when commit falsely returns an error). txf should be idempotent
// or able to detect this case.
// If maxRetries is exhausted, runInTransaction returns errTooManyRetries.
// Nested transactions are not supported and result in undefined behaviour.
// Inspired by https://cloud.google.com/appengine/docs/go/datastore/reference#RunInTransaction
func runInTransaction(maxRetries int, txf func(tx *sqlx.Tx) error) error {
	for i := 0; i < maxRetries; i++ {
		err := attemptInTransaction(txf)
		if err == nil {
			return nil
		} else if err != errRetryTransaction {
			return err
		}
	}
	return errTooManyRetries
}

func attemptInTransaction(txf func(tx *sqlx.Tx) error) (rerr error) {
	tx, err := dbSeq.Beginx()
	if err != nil {
		return fmt.Errorf("Failed opening transaction: %v", err)
	}
	defer func() {
		// UPSTREAM BUG WORKAROUND: Rollback anyway to release transaction after
		// manual commit.
		//if rerr != nil {
		if true {
			// Silently ignore rollback error, we cannot do anything. Transaction
			// will timeout eventually.
			// UPSTREAM BUG: Transaction does not timeout, connection gets reused.
			// This case is unlikely, but dangerous.
			// TODO(ivanpi): Remove workaround when bug is resolved.
			_ = tx.Rollback()
		}
	}()
	// Call txf with the transaction handle - a shallow copy of the database
	// handle (sharing the mutex, database connection, queries) with the
	// transaction object added.
	if err := txf(tx); err != nil {
		return err
	}
	// UPSTREAM BUG WORKAROUND: Commit manually.
	//if err = tx.Commit(); err != nil {
	if _, err = tx.Exec("COMMIT"); err != nil {
		return errRetryTransaction
	}
	return nil
}

////////////////////////////////////////////
// Helper methods

// randomLink creates a random link id for a given hash.
func randomLink(bHash []byte) (string, error) {
	h := make([]byte, 16, 16+len(bHash))
	err := binary.Read(crand.Reader, binary.LittleEndian, h)
	if err != nil {
		return "", fmt.Errorf("RNG failed: %v", err)
	}
	return "_" + hash.String(append(h, bHash...))[1:], nil
}
