//go:build tinygo.wasm

package db

import (
	"github.com/suborbital/reactr/api/tinygo/runnable/db/query"

	"github.com/suborbital/reactr/api/tinygo/runnable/internal/ffi"
)

// Insert executes the pre-loaded database query with the name <name>,
// and passes the arguments defined by <args>.
//
// The return value is the inserted auto-increment ID from the query result, if any,
// formatted as JSON with the key `lastInsertID`.
func Insert(name string, args ...query.Argument) ([]byte, error) {
	return ffi.DbExec(query.QueryInsert, name, args)
}

// Update executes the pre-loaded database query with the name <name>,
// and passes the arguments defined by <args>.
//
// The return value is number of rows affected by the query,
// formatted as JSON with the key `rowsAffected`.
func Update(name string, args ...query.Argument) ([]byte, error) {
	return ffi.DbExec(query.QueryUpdate, name, args)
}

// Delete executes the pre-loaded database query with the name <name>,
// and passes the arguments defined by <args>.
//
// the return value is number of rows affected by the query,
// formatted as JSON with the key `rowsAffected`.
func Delete(name string, args ...query.Argument) ([]byte, error) {
	return ffi.DbExec(query.QueryDelete, name, args)
}

// Select executes the pre-loaded database query with the name <name>,
// and passes the arguments defined by <args>.
//
// The return value is the query result formatted as JSON, with each column
// name as a top-level key.
func Select(name string, args ...query.Argument) ([]byte, error) {
	return ffi.DbExec(query.QuerySelect, name, args)
}
