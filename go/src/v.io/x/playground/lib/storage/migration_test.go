// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Tests that the migrations succeed up and down.
//
// NOTE: These tests cannot be run in parallel on the same machine because they
// interact with a fixed database on the machine.

package storage

import (
	"fmt"
	"testing"

	"github.com/rubenv/sql-migrate"

	"v.io/x/lib/dbutil"
)

var (
	dataSourceName = "playground_test@tcp(localhost:3306)/playground_test?parseTime=true"
)

// Tests that migrations can be applied to a database and rolled back multiple
// times.
func TestMigrationsUpAndDown(t *testing.T) {
	// TODO(nlacasse): This setup is very similar to the setup() func in
	// model_test.go. Consider combining them.
	migrationSource := &migrate.FileMigrationSource{
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

	// Run all migrations up and down three times.
	for i := 0; i < 3; i++ {
		up, err := migrate.Exec(db, "mysql", migrationSource, migrate.Up)
		if err != nil {
			t.Fatalf("Error migrating up: %v", err)
		}
		fmt.Printf("Applied %v migrations up.\n", up)

		down, err := migrate.Exec(db, "mysql", migrationSource, migrate.Down)
		if err != nil {
			t.Fatalf("Error migrating down: %v", err)
		}
		fmt.Printf("Applied %v migrations down.\n", down)
	}

	// Run each migration up, down, up individually.
	migrations, err := migrationSource.FindMigrations()
	if err != nil {
		t.Fatalf("migrationSource.FindMigrations() failed: %v", err)
	}
	for i, _ := range migrations {
		memMigrationSource := &migrate.MemoryMigrationSource{
			Migrations: append([]*migrate.Migration(nil), migrations[:i+1]...),
		}

		// Migrate up.
		if _, err := migrate.ExecMax(db, "mysql", memMigrationSource, migrate.Up, 1); err != nil {
			t.Fatalf("Error migrating migration %v up: %v", i, err)
		}
		fmt.Printf("Applied migration %v up.\n", i)

		// Migrate down.
		if _, err := migrate.ExecMax(db, "mysql", memMigrationSource, migrate.Down, 1); err != nil {
			t.Fatalf("Error migrating migration %v down: %v", i, err)
		}
		fmt.Printf("Applied migration %v down.\n", i)

		// Migrate up.
		if _, err := migrate.ExecMax(db, "mysql", memMigrationSource, migrate.Up, 1); err != nil {
			t.Fatalf("Error migrating migration %v up: %v", i, err)
		}
		fmt.Printf("Applied migration %v up.\n", i)
	}
}
