// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Wrapper around rubenv/sql-migrate to allow MySQL SSL connections using
// dbutil (uses dbutil sqlconf files and flags with playground-specific
// defaults instead of rubenv/sql-migrate YAML config).
//
// WARNING: MySQL doesn't support rolling back DDL transactions, so any failure
// after migrations have started requires restoring from backup or manually
// repairing database state!

package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/rubenv/sql-migrate"

	"v.io/x/lib/cmdline"
	"v.io/x/lib/dbutil"
)

const mysqlWarning = `
WARNING: MySQL doesn't support rolling back DDL transactions, so any failure
after migrations have started requires restoring from backup or manually
repairing database state!
`

// TODO(ivanpi): Add status command and sanity checks (e.g. "skipped" migrations are incorrectly applied by rubenv/sql-migrate).
// TODO(ivanpi): Guard against version skew corrupting data (e.g. add version check to client).

var cmdMigrate = &cmdline.Command{
	Name:  "migrate",
	Short: "Database schema migrations",
	Long: `
See github.com/rubenv/sql-migrate
` + mysqlWarning,
	Children: []*cmdline.Command{cmdMigrateUp, cmdMigrateDown},
}

var cmdMigrateUp = &cmdline.Command{
	Run:   runWithDBConn(runMigrate(migrate.Up)),
	Name:  "up",
	Short: "Apply new database schema migrations",
	Long: `
See github.com/rubenv/sql-migrate
` + mysqlWarning,
}

var cmdMigrateDown = &cmdline.Command{
	Run:   runWithDBConn(runMigrate(migrate.Down)),
	Name:  "down",
	Short: "Roll back database schema migrations",
	Long: `
See github.com/rubenv/sql-migrate
` + mysqlWarning,
}

const (
	migrationsTable = "migrations"
	sqlDialect      = "mysql"
	pgMigrationsDir = "${V23_ROOT}/release/projects/playground/go/src/v.io/x/playground/migrations"
)

var (
	flagMigrationsDir   string
	flagMigrationsLimit int
)

func init() {
	cmdMigrate.Flags.StringVar(&flagMigrationsDir, "dir", pgMigrationsDir, "Path to directory containing migrations.")
	cmdMigrateUp.Flags.IntVar(&flagMigrationsLimit, "limit", 0, "Maximum number of up migrations to apply. 0 for unlimited.")
	cmdMigrateDown.Flags.IntVar(&flagMigrationsLimit, "limit", 1, "Maximum number of down migrations to apply. 0 for unlimited.")
}

// Returns a DBCommand for applying migrations in the provided direction.
func runMigrate(direction migrate.MigrationDirection) DBCommand {
	return func(db *sql.DB, cmd *cmdline.Command, args []string) error {
		migrate.SetTable(migrationsTable)

		source := migrate.FileMigrationSource{
			Dir: os.ExpandEnv(flagMigrationsDir),
		}

		if *flagDryRun {
			planned, _, err := migrate.PlanMigration(db, sqlDialect, source, direction, flagMigrationsLimit)
			if err != nil {
				return fmt.Errorf("Failed getting migrations to apply: %v", err)
			}
			for i, m := range planned {
				fmt.Fprintf(cmd.Stdout(), "#%d: %q\n", i, m.Migration.Id)
				for _, q := range m.Queries {
					fmt.Fprint(cmd.Stdout(), q)
				}
			}
			return nil
		} else {
			amount, err := migrate.ExecMax(db, sqlDialect, source, direction, flagMigrationsLimit)
			if err != nil {
				return fmt.Errorf("Migration FAILED (applied %d migrations): %v", amount, err)
			}
			fmt.Fprintf(cmd.Stdout(), "Successfully applied %d migrations\n", amount)
			return nil
		}
	}
}

// Command to be wrapped with runWithDBConn().
type DBCommand func(db *sql.DB, cmd *cmdline.Command, args []string) error

// runWithDBConn is a wrapper method that handles opening and closing the
// database connection.
func runWithDBConn(fx DBCommand) cmdline.Runner {
	return func(cmd *cmdline.Command, args []string) (rerr error) {
		if *flagSQLConf == "" {
			return cmd.UsageErrorf("SQL configuration file (-sqlconf) must be provided")
		}

		// Open database connection from config,
		db, err := dbutil.NewSqlDBConnFromFile(*flagSQLConf, "SERIALIZABLE")
		if err != nil {
			return fmt.Errorf("Error opening database connection: %v", err)
		}
		// Best effort close.
		defer func() {
			if cerr := db.Close(); cerr != nil {
				cerr = fmt.Errorf("Failed closing database connection: %v", cerr)
				// Merge errors.
				if rerr == nil {
					rerr = cerr
				} else {
					rerr = fmt.Errorf("%v\n%v", rerr, cerr)
				}
			}
		}()
		// Ping database to check connection.
		if err := db.Ping(); err != nil {
			return fmt.Errorf("Error connecting to database: %v", err)
		}

		// Run wrapped function.
		return fx(db, cmd, args)
	}
}
