// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package storage

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"v.io/x/lib/dbutil"
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
func connectDb(sqlConfig *dbutil.ActiveSqlConfig, isolation string) (*sqlx.DB, error) {
	// Open db connection from config,
	conn, err := sqlConfig.NewSqlDBConn(isolation)
	if err != nil {
		return nil, err
	}

	// Create sqlx DB.
	db := sqlx.NewDb(conn, "mysql")

	// Ping db to check connection.
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("Error connecting to database: %v", err)
	}
	return db, nil
}

// Connect opens 2 connections to the database, one read-only, and one
// serializable.
func Connect(sqlConfig *dbutil.ActiveSqlConfig) (err error) {

	// Data writes for the schema are complex enough to require transactions with
	// SERIALIZABLE isolation. However, reads do not require SERIALIZABLE. Since
	// database/sql only allows setting transaction isolation per connection,
	// a separate connection with only READ-COMMITTED isolation is used for reads
	// to reduce lock contention and deadlock frequency.

	dbRead, err = connectDb(sqlConfig, "READ-COMMITTED")
	if err != nil {
		return err
	}

	dbSeq, err = connectDb(sqlConfig, "SERIALIZABLE")
	if err != nil {
		return err
	}

	return nil
}

// Close closes both databases.
func Close() error {
	if err := dbRead.Close(); err != nil {
		return err
	}

	if err := dbSeq.Close(); err != nil {
		return err
	}

	return nil
}
