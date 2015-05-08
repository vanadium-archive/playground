// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Admin tool for managing playground database and default bundles.

package main

import (
	"flag"

	"v.io/x/lib/cmdline2"
	"v.io/x/lib/dbutil"
)

func main() {
	cmdline2.Main(cmdPGAdmin)
}

var cmdPGAdmin = &cmdline2.Command{
	Name:  "pgadmin",
	Short: "Playground database management tool",
	Long: `
Tool for managing the playground database and default bundles.
Supports database schema migration.
TODO(ivanpi): bundle bootstrap
`,
	Children: []*cmdline2.Command{cmdMigrate},
}

var (
	flagDryRun = flag.Bool("n", false, "Show what commands will run, but do not execute them.")

	// Path to SQL configuration file, as described in v.io/x/lib/dbutil/mysql.go. Required parameter for most commands.
	flagSQLConf = flag.String("sqlconf", "", "Path to SQL configuration file. "+dbutil.SqlConfigFileDescription)
)
