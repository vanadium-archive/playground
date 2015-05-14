// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package storage

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"v.io/x/lib/dbutil"
	"v.io/x/playground/lib"
)

var (
	// Database handle with READ_COMMITTED transaction isolation.
	// Used for non-transactional reads.
	dbRead *sqlx.DB

	// Database handle with SERIALIZABLE transaction isolation.
	// Used for read-write transactions.
	dbSeq *sqlx.DB
)

// connectDb is a helper method to connect a single database with the given
// isolation parameter.
func connectDb(sqlConfig *dbutil.ActiveSqlConfig, isolation string) (_ *sqlx.DB, rerr error) {
	// Open db connection from config,
	conn, err := sqlConfig.NewSqlDBConn(isolation)
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %v", err)
	}
	// Create sqlx DB.
	db := sqlx.NewDb(conn, "mysql")
	// Try to close DB on error.
	defer func() {
		if rerr != nil {
			rerr = lib.MergeErrors(rerr, db.Close(), "; ")
		}
	}()

	// Ping db to check connection.
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	return db, nil
}

// Connect opens 2 connections to the database, one read-only, and one
// serializable.
func Connect(sqlConfig *dbutil.ActiveSqlConfig) (rerr error) {
	// Data writes for the schema are complex enough to require transactions with
	// SERIALIZABLE isolation. However, reads do not require SERIALIZABLE. Since
	// database/sql only allows setting transaction isolation per connection,
	// a separate connection with only READ-COMMITTED isolation is used for reads
	// to reduce lock contention and deadlock frequency.

	dbRead, rerr = connectDb(sqlConfig, "READ-COMMITTED")
	if rerr != nil {
		return rerr
	}
	// dbRead is fully initialized, try to close it on subsequent error.
	defer func() {
		if rerr != nil {
			rerr = lib.MergeErrors(rerr, dbRead.Close(), "; ")
		}
	}()

	dbSeq, rerr = connectDb(sqlConfig, "SERIALIZABLE")
	if rerr != nil {
		return rerr
	}

	return nil
}

// Close closes both databases. Should be called iff Connect() was successful.
func Close() error {
	errRead := dbRead.Close()
	errSeq := dbSeq.Close()
	return lib.MergeErrors(errRead, errSeq, "; ")
}
