package counter

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/tapglue/snaas/platform/pg"
)

const (
	pgGetCounter = `
		SELECT
			value
		FROM
			%s.counters
		WHERE
			deleted = false
			AND name = $1
			ANd user_id = $2
		LIMIT
			1`
	pgGetCounterAll = `
		SELECT
			sum(value)
		FROM
			%s.counters
		WHERE
			deleted = false
			AND name = $1`
	pgSetCounter = `
		INSERT INTO %s.counters(name, user_id, value)
		VALUES($1, $2, $3)
		ON CONFLICT (name, user_id) DO
		UPDATE SET
			value = $3`

	pgCreateSchema = `CREATE SCHEMA IF NOT EXISTS %s`
	pgCreateTable  = `
		CREATE TABLE IF NOT EXISTS %s.counters(
			name TEXT NOT NULL,
			user_id BIGINT NOT NULL,
			value BIGINT NOT NULL,
			deleted BOOL DEFAULT false,
			created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'utc'),
			updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'utc'),

			CONSTRAINT counter_id UNIQUE (name, user_id),
			PRIMARY KEY (name, user_id)
		)`
	pgDropTable = `DROP TABLE IF EXISTS %s.counters CASCADE`

	pgIndexCounterID = `
		CREATE UNIQUE INDEX
			%s
		ON
			%s.counters
		USING
			btree(name, user_id)`
	pgIndexCounterName = `
		CREATE INDEX
			%s
		ON
			%s.counters
		USING
			btree(name)`

	// Extensions.
	pgCreateExtensionModdatetime = `CREATE EXTENSION IF NOT EXISTS moddatetime`

	// Trigger to autoamtically set the latest time on updated_at, depends on
	// the moddatetime extension:
	// * https://www.postgresql.org/docs/current/static/contrib-spi.html
	// * https://github.com/postgres/postgres/blob/master/contrib/spi/moddatetime.example
	// * https://dba.stackexchange.com/a/158750
	pgAlterTriggerUpdatedAt = `
		ALTER TRIGGER %s ON %s.counters DEPENDS ON EXTENSION moddatetime`
	pgCreateTriggerUpdatedAt = `
		CREATE TRIGGER
			%s
		BEFORE UPDATE ON
			%s.counters
		FOR EACH ROW EXECUTE PROCEDURE
			moddatetime(updated_at)`
	pgDropTriggerUpdatedAt = `
		DROP TRIGGER IF EXISTS %s ON %s.counters`
)

type pgService struct {
	db *sqlx.DB
}

func PostgresService(db *sqlx.DB) Service {
	return &pgService{db: db}
}

func (s *pgService) Count(ns, name string, userID uint64) (uint64, error) {
	var (
		args  = []interface{}{name, userID}
		query = fmt.Sprintf(pgGetCounter, ns)

		value uint64
	)

	err := s.db.Get(&value, query, args...)
	if err != nil && pg.IsRelationNotFound(pg.WrapError(err)) {
		if err := s.Setup(ns); err != nil {
			return 0, err
		}

		err = s.db.Get(&value, query, args...)
	}

	return value, err
}

func (s *pgService) CountAll(ns, name string) (uint64, error) {
	var (
		args  = []interface{}{name}
		query = fmt.Sprintf(pgGetCounterAll, ns)

		value uint64
	)

	err := s.db.Get(&value, query, args...)
	if err != nil && pg.IsRelationNotFound(pg.WrapError(err)) {
		if err := s.Setup(ns); err != nil {
			return 0, err
		}

		err = s.db.Get(&value, query, args...)
	}

	return value, err
}

func (s *pgService) Set(ns, name string, userID, value uint64) error {
	var (
		args = []interface{}{
			name,
			userID,
			value,
		}
		query = fmt.Sprintf(pgSetCounter, ns)
	)

	_, err := s.db.Exec(query, args...)
	if err != nil && pg.IsRelationNotFound(pg.WrapError(err)) {
		if err := s.Setup(ns); err != nil {
			return err
		}

		_, err = s.db.Exec(query, args...)
	}

	return err
}

func (s *pgService) Setup(ns string) error {
	for _, q := range []string{
		fmt.Sprintf(pgCreateSchema, ns),
		fmt.Sprintf(pgCreateTable, ns),

		// Indexes.
		pg.GuardIndex(ns, "counter_counter_id", pgIndexCounterID),
		pg.GuardIndex(ns, "counter_counter_name", pgIndexCounterName),

		// FIXME: Re-enable when migrated to Postgres 9.6
		// Setup idempotent updated_at trigger.
		// pgCreateExtensionModdatetime,
		// fmt.Sprintf(pgDropTriggerUpdatedAt, "counter_updated_at", ns),
		// fmt.Sprintf(pgCreateTriggerUpdatedAt, "counter_updated_at", ns),
		// fmt.Sprintf(pgAlterTriggerUpdatedAt, "counter_updated_at", ns),
	} {
		_, err := s.db.Exec(q)
		if err != nil {
			return fmt.Errorf("setup '%s': %s", q, err)
		}
	}

	return nil
}

func (s *pgService) Teardown(ns string) error {
	for _, q := range []string{
		fmt.Sprintf(pgDropTable, ns),
	} {
		_, err := s.db.Exec(q)
		if err != nil {
			return fmt.Errorf("teardown '%s': %s", q, err)
		}
	}

	return nil
}
