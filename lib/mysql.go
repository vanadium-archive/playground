// Functions for opening and configuring a MySQL-like database connection.

package lib

import (
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/go-sql-driver/mysql"
)

// SQL statement suffix to be appended when creating tables.
const CreateTableSuffix = "CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"

// Opens a connection to the SQL database using the provided sqlConfig
// connection string (see (1); query parameters are not supported) and
// transaction isolation (see (2)).
// 1 -> https://github.com/go-sql-driver/mysql/#dsn-data-source-name
// 2 -> https://dev.mysql.com/doc/refman/5.5/en/server-system-variables.html#sysvar_tx_isolation
func NewDBConn(sqlConfig, txIsolation string) (*sql.DB, error) {
	return openDBConn(configureDBConn(sqlConfig, txIsolation))
}

func configureDBConn(sqlConfig, txIsolation string) string {
	// TODO(ivanpi): Parse parameters from sqlConfig?
	params := url.Values{}
	// Setting charset is unneccessary when collation is set, according to
	// https://github.com/go-sql-driver/mysql/#charset
	params.Set("collation", "utf8mb4_general_ci")
	params.Set("parseTime", "true")
	params.Set("loc", "UTC")
	params.Set("time_zone", "'+00:00'")
	// TODO(ivanpi): Configure TLS certificates for Cloud SQL connection.
	params.Set("tx_isolation", "'"+txIsolation+"'")
	return sqlConfig + "?" + params.Encode()
}

func openDBConn(sqlConfig string) (*sql.DB, error) {
	db, err := sql.Open("mysql", sqlConfig)
	if err != nil {
		return nil, fmt.Errorf("failed opening database connection at %s : %v", sqlConfig, err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed connecting to database at %s : %v", sqlConfig, err)
	}
	return db, nil
}
