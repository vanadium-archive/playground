// lsql database connection wrapper providing prepared statement caching with
// transaction support and simple CRUD operations over SqlData types.
//
// DBHandle wraps a SQL database connection. It provides Prepare*() and Get*()
// methods, which allow saving and retrieving prepared SQL statements keyed by
// a label and, optionally, SqlData type.
// E*() methods are convenience methods for common CRUD operations on a SqlData
// entity. The entity type must be registered beforehand using RegisterType()
// and the corresponding table created.
// RunInTransaction() supports executing a sequence of operations inside of a
// transaction with retry on failure support.

package lsql

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
)

var (
	// Error returned by EFetch when no entity is found for the given key.
	ErrNoSuchEntity = errors.New("lsql: no such entity")
	// Error returned by RunInTransaction when retries are exhausted.
	ErrTooManyRetries = errors.New("lsql: too many retries")
	// Error returned by EInsert and EUpdate when no rows are affected.
	// Note: In case of EUpdate, this can happen if the entity had been deleted,
	// but also (depending on database configuration) if the entity was unchanged
	// by the update.
	ErrNoRowsAffected = errors.New("lsql: no rows affected")
	// Error that should be returned from a transaction callback to trigger a
	// rollback and retry. Other errors cause a rollback and abort.
	RetryTransaction = errors.New("lsql: retry transaction")
)

// SQL database handle storing prepared statements.
// Initialize using NewDBHandle.
type DBHandle struct {
	dbc *sql.DB
	tx  *sql.Tx

	mu      *sync.RWMutex
	queries map[string]*sql.Stmt
}

func NewDBHandle(dbc *sql.DB) *DBHandle {
	return &DBHandle{
		dbc:     dbc,
		mu:      &sync.RWMutex{},
		queries: make(map[string]*sql.Stmt),
	}
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
// If the error returned from txf is RetryTransaction, txf is retried as if the
// commit failed. Otherwise, txf is not retried, and RunInTransaction returns
// the error.
// In rare cases, txf may be retried even if the transaction was successfully
// committed (when commit falsely returns an error). txf should be idempotent
// or able to detect this case.
// If maxRetries is exhausted, RunInTransaction returns ErrTooManyRetries.
// Nested transactions are not supported and result in undefined behaviour.
// Inspired by https://cloud.google.com/appengine/docs/go/datastore/reference#RunInTransaction
func (h *DBHandle) RunInTransaction(maxRetries int, txf func(txh *DBHandle) error) error {
	for i := 0; i < maxRetries; i++ {
		err := h.attemptInTransaction(txf)
		if err == nil {
			return nil
		} else if err != RetryTransaction {
			return err
		}
	}
	return ErrTooManyRetries
}

func (h *DBHandle) attemptInTransaction(txf func(txh *DBHandle) error) (rerr error) {
	tx, err := h.dbc.Begin()
	if err != nil {
		return fmt.Errorf("lsql: failed opening transaction: %v", err)
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
	err = txf(&DBHandle{
		dbc:     h.dbc,
		mu:      h.mu,
		queries: h.queries,
		tx:      tx,
	})
	if err != nil {
		return err
	}
	// UPSTREAM BUG WORKAROUND: Commit manually.
	//if err = tx.Commit(); err != nil {
	if _, err = tx.Exec("COMMIT"); err != nil {
		return RetryTransaction
	}
	return nil
}

//////////////////////////////////////////
// Prepared statement caching support

// Prepares the SQL statement/query and saves it under the provided label.
func (h *DBHandle) Prepare(label, query string) error {
	return h.prepareInternal("c:"+label, query)
}

// Retrieves the previously prepared SQL statement/query for the label.
func (h *DBHandle) Get(label string) *sql.Stmt {
	return h.getInternal("c:" + label)
}

// PrepareFor and GetFor are equivalent to Prepare and Get, with the label
// interpreted in the namespace of the SqlData type.

func (h *DBHandle) PrepareFor(proto SqlData, label, query string) error {
	return h.prepareInternal("t:"+proto.TableName()+":"+label, query)
}

func (h *DBHandle) GetFor(proto SqlData, label string) *sql.Stmt {
	return h.getInternal("t:" + proto.TableName() + ":" + label)
}

func (h *DBHandle) prepareInternal(label, query string) error {
	stmt, err := h.dbc.Prepare(query)
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.queries[label] = stmt
	return nil
}

func (h *DBHandle) getInternal(label string) (stmt *sql.Stmt) {
	if h.tx != nil {
		defer func() {
			stmt = h.tx.Stmt(stmt)
		}()
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.queries[label]
}

//////////////////////////////////////////
// SqlData common operation support

// Creates the table for the given SqlData prototype.
func (h *DBHandle) CreateTable(proto SqlData, ifNotExists bool, stmtSuffix string) error {
	_, err := h.dbc.Exec(proto.TableDef().GetCreateTable(ifNotExists) + " " + stmtSuffix)
	return err
}

// Prepares common operation queries/statements for the given SqlData type.
// If readonly is set, prepares only reading queries, not writing statements.
func (h *DBHandle) RegisterType(proto SqlData, readonly bool) error {
	t := proto.TableDef()
	if err := h.PrepareFor(proto, "fetch", t.GetSelectQuery(t.GetWhereKey())); err != nil {
		return err
	}
	if err := h.PrepareFor(proto, "exists", t.GetCountQuery(t.GetWhereKey())); err != nil {
		return err
	}
	if !readonly {
		if err := h.PrepareFor(proto, "insert", t.GetInsertQuery()); err != nil {
			return err
		}
		if err := h.PrepareFor(proto, "update", t.GetUpdateQuery(t.GetWhereKey())); err != nil {
			return err
		}
	}
	return nil
}

// Functions below execute common operations for the SqlData type.
// The type must be registered with RegisterType beforehand.

// Fetches the entity with the given key and stores it into dst.
// If no entity for key is found, returns ErrNoSuchEntity.
func (h *DBHandle) EFetch(key interface{}, dst SqlData) error {
	err := h.GetFor(dst, "fetch").QueryRow(key).Scan(dst.QueryRefs()...)
	if err == sql.ErrNoRows {
		err = ErrNoSuchEntity
	}
	return err
}

// Checks if an entity of the given SqlData prototype with the given key
// exists in the database.
func (h *DBHandle) EExists(key interface{}, proto SqlData) (bool, error) {
	var cnt int
	err := h.GetFor(proto, "exists").QueryRow(key).Scan(&cnt)
	return cnt > 0, err
}

// Inserts the entity stored in src into the database.
func (h *DBHandle) EInsert(src SqlData) error {
	res, err := h.GetFor(src, "insert").Exec(src.QueryRefs()...)
	if err == nil {
		err = checkOneRowAffected(res)
	}
	return err
}

// Updates the database record for the entity stored in src.
func (h *DBHandle) EUpdate(src SqlData) error {
	qrefs := src.QueryRefs()
	// Move primary key reference to the end for WHERE clause.
	qrefs = append(qrefs[1:], qrefs[0])
	res, err := h.GetFor(src, "update").Exec(qrefs...)
	if err == nil {
		err = checkOneRowAffected(res)
	}
	return err
}

func checkOneRowAffected(res sql.Result) error {
	raf, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("lsql: failed checking number of rows affected: %v", err)
	}
	if raf > 1 {
		return fmt.Errorf("lsql: more than one row affected")
	}
	if raf == 0 {
		return ErrNoRowsAffected
	}
	return nil
}
