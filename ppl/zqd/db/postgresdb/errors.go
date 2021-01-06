package postgresdb

import "github.com/go-pg/pg/v10"

// This file is for convience functions with respect to pg errors. Would
// normally use the errors.As facilities but the does not work with certain
// queries in pkg pg.

func IsUniqueViolation(err error) bool {
	pgerr, ok := err.(pg.Error)
	return ok && pgerr.Field('C') == "23505"
}

func IsForeignKeyViolation(err error) bool {
	pgerr, ok := err.(pg.Error)
	return ok && pgerr.Field('C') == "23503"
}
