// Package postgres connects to a PostgreSQL database
// and provides a querying API.
//
// # Connection
//
// Package postgres establishes a connection to a PostgreSQL database via the [Connect] function.
// If instead of a [*DB] you need a raw [*gorm.DB], use [ConnectRaw].
//
// # Querying API
//
// Package postgres provides an API for executing queries
// using the database connection this package provides.
//
// [*DB] is a thin wrapper around [*gorm.DB].
// It's primary purpose is to provide "one way to do things" as much as possible.
// A secondary purpose of this wrapper is to provide additional error handling
// that makes issues arising from queries easier to inspect.
// GORM panics, which this package attempts to gracefully handle.
//
// [*DB] largely achieves its goals by removing options from [*gorm.DB].
// But, the underlying [*gorm.DB] is exposed via the [*DB.DB] method
// as an escape hatch for more advanced use cases.
//
// # Finisher methods
//
// Certain methods on [*DB] execute the query built up by previous methods.
// The query cannot be changed or re-used after a finisher method.
// These finisher methods specifically return an error
// that may have occurred in the query chain or in the execution of the query itself.
//
// In addition to the bulk of GORM's simplest finisher methods,
// package posgres adds [*DB.Exists].
// This provides a single way of creating SELECT EXISTS queries
// that are efficient for checking for the existence of a record.
//
// # TODOs
//
// Here are a few items that may be interesting to tackle:
// - [ ] Union(query *postgres.DB) *postgres.DB
// - [ ] Eliminate Model in favor of Table
package postgres
