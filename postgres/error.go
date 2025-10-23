package postgres

import "regexp"

var (
	// These errors originate from the std lib database/sql package.
	//
	// Cf., https://cs.opensource.google/go/go/+/master:src/database/sql/sql.go;l=3395;drc=3dbef65bf37f1b7ccd1f884761341a5a15456ffa
	errSQLScan          = regexp.MustCompile(`sql: expected \d+ destination arguments in Scan, not \d+`)
	errSQLUnaddressable = regexp.MustCompile(`sql: Scan error on column index \d+, name "\w+": destination not a pointer`)

	// errSQLSyntax is a very loose aggregation of error codes
	// originating from PostgreSQL itself
	// that are some sort of syntax issue in the statement or datatype mismatch.
	//
	// Cf., https://www.postgresql.org/docs/current/errcodes-appendix.html
	errSQLSyntax = regexp.MustCompile(`SQLSTATE (42601|22P02)`)

	errConstraintViolation = regexp.MustCompile(`SQLSTATE (23502)`)
	errUniqViolation       = regexp.MustCompile(`SQLSTATE (23505)`)
)
