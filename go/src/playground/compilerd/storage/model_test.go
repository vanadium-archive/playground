// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Tests for the storage model.
// These tests only test the exported API of the storage model.
//
// NOTE: These tests cannot be run in parallel on the same machine because they
// interact with a fixed database on the machine.

package storage_test

import (
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rubenv/sql-migrate"

	"v.io/x/lib/dbutil"

	"playground/compilerd/storage"
)

var (
	dataSourceName = "playground_test@tcp(localhost:3306)/playground_test?parseTime=true"
)

// setup cleans the database, runs migrations, and connects to the database.
// It returns a teardown function that closes the database connection.
func setup(t *testing.T) func() {
	// Migrate down then up.
	migrations := &migrate.FileMigrationSource{
		Dir: "../../migrations",
	}
	migrate.SetTable("migrations")

	sqlConfig := dbutil.SqlConfig{
		DataSourceName: dataSourceName,
		TLSDisable:     true,
	}
	activeSqlConfig, err := sqlConfig.Activate("")

	db, err := activeSqlConfig.NewSqlDBConn("SERIALIZABLE")
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}

	// Remove any existing tables.
	tableNames := []string{"bundle_link", "bundle_data", "migrations"}
	for _, tableName := range tableNames {
		db.Exec("DROP TABLE " + tableName)
	}

	if _, err = migrate.Exec(db, "mysql", migrations, migrate.Up); err != nil {
		t.Fatalf("Error migrating up: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close() failed: %v", err)
	}

	// Connect to the storage.
	if err := storage.Connect(activeSqlConfig); err != nil {
		t.Fatalf("storage.Connect(%v) failed: %v", activeSqlConfig, err)
	}

	teardown := func() {
		if err := storage.Close(); err != nil {
			t.Fatalf("storage.Close() failed: %v", err)
		}
	}
	return teardown
}

func TestGetBundleDataByLinkId(t *testing.T) {
	defer setup(t)()

	// Get with a unknown id should return ErrNotFound.
	id := "foobar"
	if _, err := storage.GetBundleDataByLinkId(id); err != storage.ErrNotFound {
		t.Errorf("Expected GetBundleDataByLinkId with unknown id to return ErrNotFound, but instead got %v", err)
	}

	// Add a bundle.
	json := []byte("mock_json_data")
	bLink, _, err := storage.StoreBundleLinkAndData(json)
	if err != nil {
		t.Fatalf("Expected StoreBundleLinkAndData(%v) not to error, but got %v", json, err)
	}

	// Bundle should exist.
	gotBdata, err := storage.GetBundleDataByLinkId(bLink.Id)
	if err != nil {
		t.Errorf("Expected GetBundleDataByLinkId(%v) not to error, but got %v", bLink.Id, err)
	}

	// Bundle should have expected json.
	if got, want := gotBdata.Json, string(json); got != want {
		t.Errorf("Expected %v to equal %v.", got, want)
	}
}

func assertValidLinkDataPair(json string, bLink *storage.BundleLink, bData *storage.BundleData) error {
	if string(bLink.Hash) != string(bData.Hash) {
		return fmt.Errorf("Expected %v to equal %v", string(bLink.Hash), string(bData.Hash))
	}

	if bLink.Id == "" {
		return fmt.Errorf("Expected bundle link to have id.")
	}

	if bData.Json != json {
		return fmt.Errorf("Expected %v to equal %v", bData.Json, json)
	}
	return nil
}

func TestStoreBundleLinkAndData(t *testing.T) {
	defer setup(t)()

	mockJson := []byte("bizbaz")

	// Storing the json once should succeed.
	bLink1, bData1, err := storage.StoreBundleLinkAndData(mockJson)
	if err != nil {
		t.Fatalf("StoreBundleLinkAndData(%v) failed: %v", mockJson, err)
	}
	if err := assertValidLinkDataPair(string(mockJson), bLink1, bData1); err != nil {
		t.Fatalf("Got invalid link data pair: %v", err)
	}

	// Storing the bundle again should succeed.
	bLink2, bData2, err := storage.StoreBundleLinkAndData(mockJson)
	if err != nil {
		t.Fatalf("StoreBundleLinkAndData(%v) failed: %v", mockJson, err)
	}
	if err := assertValidLinkDataPair(string(mockJson), bLink2, bData2); err != nil {
		t.Error("Got invalid link data pair: %v", err)
	}

	// Bundle links should have different ids.
	if bLink1.Id == bLink2.Id {
		t.Errorf("Expected bundle links to have different ids, but got %v and %v", bLink1.Id, bLink2.Id)
	}

	// Bundle datas should have equal hashes.
	if want, got := string(bData1.Hash), string(bData2.Hash); want != got {
		t.Errorf("Expected bundle datas to have equal hashes, but got %v and %v", want, got)
	}
}
