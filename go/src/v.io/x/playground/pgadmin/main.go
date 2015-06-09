// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Admin tool for managing playground database and default bundles.

package main

import (
	"flag"

	"v.io/x/lib/cmdline"
	"v.io/x/lib/dbutil"
)

func main() {
	cmdline.Main(cmdPGAdmin)
}

var cmdPGAdmin = &cmdline.Command{
	Name:  "pgadmin",
	Short: "Playground database management tool",
	Long: `
Tool for managing the playground database and default bundles.
Supports database schema migration and loading default bundles into database.
`,
	Children: []*cmdline.Command{cmdMigrate, cmdBundle},
}

var (
	flagDryRun  = flag.Bool("n", false, "Show necessary database modifications, but do not apply them.")
	flagVerbose = flag.Bool("v", true, "Show more verbose output.")

	// Path to SQL configuration file, as described in v.io/x/lib/dbutil/mysql.go. Required parameter for most commands.
	flagSQLConf = flag.String("sqlconf", "", "Path to SQL configuration file. "+dbutil.SqlConfigFileDescription)
)

func logVerbose() bool {
	return *flagDryRun || *flagVerbose
}
