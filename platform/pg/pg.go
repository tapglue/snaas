package pg

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// MetaNamespace identifies the schema used to bundle tables not belonging to a
// customer/app.
const MetaNamespace = "tg"

// TimeFormat can be used to extract and store time in a reproducible way.
const TimeFormat = "2006-01-02 15:04:05.000000 UTC"

const URLTest = "postgres://%s@127.0.0.1:5432/tapglue_test?sslmode=disable&connect_timeout=5"

const (
	fmtClause = "\nAND "
	fmtWHERE  = "WHERE\n%s"
)

// ErrRelationNotFound is returned as equivalent to the Postgres error.
var ErrRelationNotFound = errors.New("relation not found")

// To ensure idempotence we want to create the index only if it doesn't exist,
// while this feature is about to hit Postgres in 9.5 it is not yet available.
// We fallback to a conditional create taken from:
// http://dba.stackexchange.com/a/35626.
const guardIndex = `DO $$
		BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM pg_indexes WHERE schemaname = '%s' AND indexname = '%s'
		) THEN
		%s;
		END IF;
		END$$;`

// ClausesToWhere transforms a list of SQL clauses into a WHERE statement.
func ClausesToWhere(clauses ...string) string {
	return fmt.Sprintf(fmtWHERE, strings.Join(clauses, fmtClause))
}

// GuardIndex wraps an index creation query with a condition to prevent conflicts.
func GuardIndex(namespace, index, query string) string {
	return fmt.Sprintf(
		guardIndex,
		namespace,
		index,
		fmt.Sprintf(query, index, namespace),
	)
}

// IsRelationNotFound indicates if err is ErrRelationNotFound.
func IsRelationNotFound(err error) bool {
	return err == ErrRelationNotFound
}

// WrapError check the given error if it indicates that the relation wasn't
// present, otherwise returns the original error.
func WrapError(err error) error {
	if err, ok := err.(*pq.Error); ok && err.Code == "42P01" {
		return ErrRelationNotFound
	}

	return err
}
