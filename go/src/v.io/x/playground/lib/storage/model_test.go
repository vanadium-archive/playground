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

	"github.com/rubenv/sql-migrate"

	"v.io/x/lib/dbutil"
	"v.io/x/playground/lib/storage"
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
	if _, _, err := storage.GetBundleByLinkIdOrSlug(id); err != storage.ErrNotFound {
		t.Errorf("Expected GetBundleByLinkIdOrSlug with unknown id to return ErrNotFound, but instead got: %v", err)
	}

	// Add a bundle.
	json := "mock_json_data"
	bLink, _, err := storage.StoreBundleLinkAndData(json)
	if err != nil {
		t.Fatalf("Expected StoreBundleLinkAndData(%v) not to error, but got: %v", json, err)
	}

	// Bundle should exist.
	gotBLink, gotBdata, err := storage.GetBundleByLinkIdOrSlug(bLink.Id)
	if err != nil {
		t.Errorf("Expected GetBundleDataByLinkIdOrSlug(%v) not to error, but got: %v", bLink.Id, err)
	}

	// Bundle should have expected id.
	if got, want := gotBLink.Id, bLink.Id; got != want {
		t.Errorf("Expected %v to equal %v.", got, want)
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

	mockJson := "bizbaz"

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
		t.Errorf("Got invalid link data pair: %v", err)
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

func makeMockNamedBundle(mockSlug, mockJson string) *storage.NewBundle {
	return &storage.NewBundle{
		BundleDesc: storage.BundleDesc{
			Slug: storage.EmptyNullString(mockSlug),
		},
		Json: mockJson,
	}
}

func expectDefaultBundles(got []*storage.BundleLink, want []*storage.NewBundle) error {
	if len(got) != len(want) {
		return fmt.Errorf("Expected %d, got %d bundles.", len(want), len(got))
	}

	for _, listBLink := range got {
		// For each listed BundleLink, get corresponding BundleData.
		gotBLink, gotBData, err := storage.GetBundleByLinkIdOrSlug(listBLink.Id)
		if err != nil {
			return fmt.Errorf("Expected GetBundleDataByLinkIdOrSlug(%v) not to error, but got: %v", listBLink.Id, err)
		}

		// Check that the bundle is non-anonymous and default.
		if gotBLink.Slug == "" {
			return fmt.Errorf("Expected bundle for %v to have non-empty slug", gotBLink.Id)
		}
		if !gotBLink.IsDefault {
			return fmt.Errorf("Expected bundle %v to be marked as default", gotBLink.Slug)
		}

		// Find expected NewBundle with slug matching the retrieved bundle.
		var original *storage.NewBundle
		for _, newBundle := range want {
			if newBundle.Slug == gotBLink.Slug {
				original = newBundle
				break
			}
		}
		if original == nil {
			return fmt.Errorf("Unexpected bundle with slug %v", gotBLink.Slug)
		}

		// Check that the retrieved bundle is valid and matches expected JSON.
		if err := assertValidLinkDataPair(original.Json, gotBLink, gotBData); err != nil {
			return fmt.Errorf("Got invalid link data pair: %v", err)
		}
	}
	return nil
}

func expectNonDefaultBundle(bId, wantJson string) error {
	bLink, bData, err := storage.GetBundleByLinkIdOrSlug(bId)
	if err != nil {
		return fmt.Errorf("GetBundleDataByLinkIdOrSlug(%v) failed: %v", bId, err)
	}
	if bLink.Slug != "" {
		return fmt.Errorf("Expected anonymous bundle for %v, got slug %v", bId, bLink.Slug)
	}
	if bLink.IsDefault {
		return fmt.Errorf("Expected non-default bundle for %v", bId)
	}
	if err := assertValidLinkDataPair(wantJson, bLink, bData); err != nil {
		return fmt.Errorf("Got invalid link data pair: %v", err)
	}
	return nil
}

func TestDefaultBundles(t *testing.T) {
	defer setup(t)()

	mockSlugs := []string{"one", "two", "three", "four"}
	mockJson := []string{"forty-two", "forty-seven", "leet"}

	defBundlesA := []*storage.NewBundle{
		makeMockNamedBundle(mockSlugs[0], mockJson[0]),
		makeMockNamedBundle(mockSlugs[1], mockJson[0]),
	}

	// Storing default bundles should succeed.
	if err := storage.ReplaceDefaultBundles(defBundlesA); err != nil {
		t.Fatalf("A: ReplaceDefaultBundles(%v) failed: %v", defBundlesA, err)
	}

	// Storing a non-default bundle should succeed.
	nondBLink, _, err := storage.StoreBundleLinkAndData(mockJson[2])
	if err != nil {
		t.Fatalf("StoreBundleLinkAndData(%v) failed: %v", mockJson[2], err)
	}

	defBundlesDup := []*storage.NewBundle{
		makeMockNamedBundle(mockSlugs[1], mockJson[1]),
		makeMockNamedBundle(mockSlugs[1], mockJson[1]),
	}

	// Trying to store default bundles with duplicate slugs should fail and be
	// rolled back (not affect subsequent assertions).
	if err := storage.ReplaceDefaultBundles(defBundlesDup); err == nil {
		t.Fatalf("Dup: ReplaceDefaultBundles(%v) with duplicate slugs should have failed", defBundlesDup)
	}

	// Listing default bundles should succeed.
	storedDefBundlesA, err := storage.GetDefaultBundleList()
	if err != nil {
		t.Fatalf("A: GetDefaultBundleList() failed: %v", err)
	}

	// Default bundle list should not contain the non-default bundle.
	if err := expectDefaultBundles(storedDefBundlesA, defBundlesA); err != nil {
		t.Errorf("A: Default bundle mismatch: %v", err)
	}

	// Non-default bundle should be untouched.
	if err := expectNonDefaultBundle(nondBLink.Id, mockJson[2]); err != nil {
		t.Errorf("Non-default bundle mismatch: %v", err)
	}

	defBundlesB := []*storage.NewBundle{
		makeMockNamedBundle(mockSlugs[1], mockJson[2]),
		makeMockNamedBundle(mockSlugs[2], mockJson[1]),
		makeMockNamedBundle(mockSlugs[3], mockJson[0]),
	}

	// Replacing default bundles should succeed.
	if err := storage.ReplaceDefaultBundles(defBundlesB); err != nil {
		t.Fatalf("B: ReplaceDefaultBundles(%v) failed: %v", defBundlesB, err)
	}

	// Listing default bundles should succeed.
	storedDefBundlesB, err := storage.GetDefaultBundleList()
	if err != nil {
		t.Fatalf("B: GetDefaultBundleList() failed: %v", err)
	}

	// Default bundle list should not contain the old default bundles.
	if err := expectDefaultBundles(storedDefBundlesB, defBundlesB); err != nil {
		t.Fatalf("B: Default bundle mismatch: %v", err)
	}

	// Non-default bundle should still be untouched.
	if err := expectNonDefaultBundle(nondBLink.Id, mockJson[2]); err != nil {
		t.Errorf("Non-default bundle mismatch: %v", err)
	}

	// Old default bundles should still be reachable by id.
	if err := expectNonDefaultBundle(storedDefBundlesA[0].Id, mockJson[0]); err != nil {
		t.Errorf("Old default bundle mismatch: %v", err)
	}
	// But not by slug.
	if _, _, err := storage.GetBundleByLinkIdOrSlug(mockSlugs[0]); err != storage.ErrNotFound {
		t.Errorf("Expected GetBundleByLinkIdOrSlug with old slug to return ErrNotFound, but instead got: %v", err)
	}

	// New bundle should be reachable by slug.
	niBLink, niBData, err := storage.GetBundleByLinkIdOrSlug(mockSlugs[1])
	if err != nil {
		t.Fatalf("GetBundleDataByLinkIdOrSlug(%v) failed: %v", mockSlugs[1], err)
	}
	if err := assertValidLinkDataPair(mockJson[2], niBLink, niBData); err != nil {
		t.Errorf("Got invalid link data pair: %v", err)
	}
}
