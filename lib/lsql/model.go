// lsql support for representing an entity type as a SQL table.
//
// Each entity type (for example, a struct) corresponding to a single SQL
// table should implement the SqlData interface, specifying the table name,
// definition of all table columns and constraints (as a SqlTable), and
// mapping of entity instance data fields to columns.
//
// The SqlTable::Get*() methods generate SQL query/statement strings for
// creating the table and other common SQL operations over the specified
// type. Generated strings and string fragments are in prepared statement
// format (with question mark placeholders for data values). Default values
// and projection are not supported.

package lsql

import (
	"strings"
)

// An entity type representable as a SQL table row.
// Should be implemented by each type to be persisted in a SQL table via lsql.
type SqlData interface {
	// SQL table name (must be a valid SQL identifier).
	// Return value must depend only on entity type, not entity instance.
	TableName() string
	// SQL table definition (refer to NewSqlTable).
	// Return value must depend only on entity type, not entity instance.
	TableDef() *SqlTable
	// Array of references to entity instance data fields mapping to SQL columns,
	// in the same order as specified in the table definition. Every column must
	// have a corresponding reference of the appropriate type.
	QueryRefs() []interface{}
}

//////////////////////////////////////////
// Table schema description

// Read-only specification of a SQL column.
type SqlColumn struct {
	// Column name (must be a valid SQL identifier).
	Name string
	// Column SQL type, as specified in a CREATE TABLE statement.
	Type string
	// Whether the column is allowed to be NULL.
	Null bool
}

// Initialize using NewSqlTable.
type SqlTable struct {
	tableName   string
	keyName     string
	columns     []SqlColumn
	constraints []string
}

// proto: Instance of the corresponding data entity.
// keyName: Name of the primary key column.
// columns: SqlColumn specifications for each column.
// constraints: Table constraints in SQL syntax (excluding PRIMARY KEY).
func NewSqlTable(proto SqlData, keyName string, columns []SqlColumn, constraints []string) *SqlTable {
	return &SqlTable{
		tableName:   proto.TableName(),
		keyName:     keyName,
		columns:     columns,
		constraints: constraints,
	}
}

func (t *SqlTable) TableName() string {
	return t.tableName
}

func (t *SqlTable) KeyName() string {
	return t.keyName
}

// List of column names.
func (t *SqlTable) ColumnNames() []string {
	colNames := make([]string, 0, len(t.columns))
	for _, c := range t.columns {
		colNames = append(colNames, c.Name)
	}
	return colNames
}

func (c *SqlColumn) getColumnSpec() string {
	nullSpec := "NULL"
	if !c.Null {
		nullSpec = "NOT NULL"
	}
	return c.Name + " " + c.Type + " " + nullSpec
}

//////////////////////////////////////////
// Statement/query string generation

// SQL statement to create the table.
func (t *SqlTable) GetCreateTable(ifNotExists bool) string {
	createCmd := "CREATE TABLE "
	if ifNotExists {
		createCmd += "IF NOT EXISTS "
	}
	clauses := make([]string, 0, len(t.columns)+1+len(t.constraints))
	for _, c := range t.columns {
		clauses = append(clauses, c.getColumnSpec())
	}
	clauses = append(clauses, "PRIMARY KEY ("+t.KeyName()+")")
	clauses = append(clauses, t.constraints...)
	return createCmd + t.TableName() + " ( " + strings.Join(clauses, ", ") + " )"
}

// Get* methods below return SQL query/statement strings with placeholders for
// common CRUD operations over the corresponding type.

// WHERE constraint fragment for matching primary key.
func (t *SqlTable) GetWhereKey() string {
	return "(" + t.KeyName() + "=?)"
}

// SELECT query for entities matching WHERE clause. No projection.
func (t *SqlTable) GetSelectQuery(whereClause string) string {
	query := "SELECT " + strings.Join(t.ColumnNames(), ",") + " FROM " + t.TableName()
	return appendWhere(query, whereClause)
}

// COUNT query for entities matching WHERE clause.
func (t *SqlTable) GetCountQuery(whereClause string) string {
	query := "SELECT COUNT(" + t.KeyName() + ") FROM " + t.TableName()
	return appendWhere(query, whereClause)
}

// INSERT query for entity. Default values not supported, all columns mapped.
func (t *SqlTable) GetInsertQuery() string {
	colNames := t.ColumnNames()
	query := "INSERT INTO " + t.TableName() +
		" (" + strings.Join(colNames, ",") + ")" +
		" VALUES (?" + strings.Repeat(",?", len(colNames)-1) + ")"
	return query
}

func appendWhere(query, whereClause string) string {
	return query + " WHERE (" + whereClause + ")"
}
