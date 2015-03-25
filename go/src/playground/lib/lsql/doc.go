// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lsql implements a utility wrapper around the database/sql library.
//
// It simplifies common operations on data entities. An entity type corresponds
// to a SQL table, and an instance of the type corresponds to a single row in
// the table.
// Each entity type must implement the SqlData interface, which specifies the
// table schema and mapping of entity instance fields to SQL table columns.
// Each entity must have exactly one primary key. More complex relations, such
// as foreign key constraints, need to be handled with care (for example,
// executing statements in the correct order).
//
// The library supports creating database tables from the schema, building
// query/statement strings for common operations such as SELECT and INSERT,
// caching prepared queries keyed by labels and entity types, and transactional
// execution. The most common CRUD operations are supported directly by
// automatically prepared queries and helper functions.
package lsql
