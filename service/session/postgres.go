package session

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/tapglue/api/platform/pg"
)

const (
	pgInsertSession = `INSERT INTO
		%s.sessions(user_id, session_id, created_at, enabled, device_id)
		VALUES($1, $2, $3, $4, $5)`
	pgUpdateSession = `
		UPDATE
			%s.sessions
		SET
			enabled = $3
		WHERE
			user_id = $1 AND
			session_id = $2`

	pgClauseDeviceIDs = `device_id IN (?)`
	pgClauseEnabled   = `enabled = ?`
	pgClauseIDs       = `session_id IN (?)`
	pgClauseUserIDs   = `user_id IN (?)`

	pgOrderCreatedAt = `ORDER BY created_at DESC`

	pgListSessions = `
		SELECT
			user_id, session_id, created_at, enabled, device_id
		FROM
			%s.sessions
		%s`

	pgCreateSchema = `CREATE SCHEMA IF NOT EXISTS %s`
	pgCreateTable  = `CREATE TABLE IF NOT EXISTS %s.sessions (
		user_id BIGINT NOT NULL,
		session_id VARCHAR(40) NOT NULL,
		created_at TIMESTAMP DEFAULT now() NOT NULL,
		enabled BOOL DEFAULT TRUE NOT NULL,
		device_id VARCHAR(255)
	)`
	pgDropTable = `DROP TABLE IF EXISTS %s.sessions`

	pgIndexDeviceIDUserID = `
		CREATE INDEX
			%s
		ON
			%s.sessions (device_id, user_id)
		WHERE
			enabled = true`
	pgIndexID = `
		CREATE INDEX
			%s
		ON
			%s.sessions (session_id)
		WHERE
			enabled = true`
	pgIndexIDUserID = `
		CREATE INDEX
			%s
		ON
			%s.sessions (session_id, user_id)
		WHERE
			enabled = true`
)

type pgService struct {
	db *sqlx.DB
}

// PostgresService returns a Postgres based Service implementation.
func PostgresService(db *sqlx.DB) Service {
	return &pgService{db: db}
}

func (s *pgService) Put(ns string, session *Session) (*Session, error) {
	var (
		params = []interface{}{
			session.UserID,
			session.ID,
			session.Enabled,
		}
		query = fmt.Sprintf(pgUpdateSession, ns)
	)

	if err := session.Validate(); err != nil {
		return nil, err
	}

	if session.CreatedAt.IsZero() {
		ts, err := time.Parse(pg.TimeFormat, time.Now().Format(pg.TimeFormat))
		if err != nil {
			return nil, err
		}
		session.CreatedAt = ts
	}

	session.CreatedAt = session.CreatedAt.UTC()

	if session.ID == "" {
		session.ID = generateID()
		params = []interface{}{
			session.UserID,
			session.ID,
			session.CreatedAt,
			session.Enabled,
			session.DeviceID,
		}
		query = fmt.Sprintf(pgInsertSession, ns)
	}

	_, err := s.db.Exec(query, params...)
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return nil, err
			}

			_, err = s.db.Exec(query, params...)
		}
	}

	return session, err
}

func (s *pgService) Query(ns string, opts QueryOptions) (List, error) {
	clauses, params, err := convertOpts(opts)
	if err != nil {
		return nil, err
	}

	return s.listSessions(ns, clauses, params...)
}

func (s *pgService) Setup(ns string) error {
	qs := []string{
		fmt.Sprintf(pgCreateSchema, ns),
		fmt.Sprintf(pgCreateTable, ns),
		pg.GuardIndex(ns, "session_device_id_user_id", pgIndexDeviceIDUserID),
		pg.GuardIndex(ns, "session_id", pgIndexID),
		pg.GuardIndex(ns, "session_id_user_id", pgIndexIDUserID),
	}

	for _, query := range qs {
		_, err := s.db.Exec(query)
		if err != nil {
			return fmt.Errorf("query (%s): %s", query, err)
		}
	}

	return nil
}

func (s *pgService) Teardown(ns string) error {
	qs := []string{
		fmt.Sprintf(pgDropTable, ns),
	}

	for _, query := range qs {
		_, err := s.db.Exec(query)
		if err != nil {
			return fmt.Errorf("query (%s): %s", query, err)
		}
	}

	return nil
}

func (s *pgService) listSessions(
	ns string,
	clauses []string,
	params ...interface{},
) (List, error) {
	c := strings.Join(clauses, "\nAND ")

	if len(clauses) > 0 {
		c = fmt.Sprintf("WHERE %s", c)
	}

	query := strings.Join([]string{
		fmt.Sprintf(pgListSessions, ns, c),
		pgOrderCreatedAt,
	}, "\n")

	query = sqlx.Rebind(sqlx.DOLLAR, query)

	rows, err := s.db.Query(query, params...)
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return nil, err
			}

			rows, err = s.db.Query(query, params...)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	defer rows.Close()

	ss := List{}

	for rows.Next() {
		s := &Session{}

		err := rows.Scan(
			&s.UserID,
			&s.ID,
			&s.CreatedAt,
			&s.Enabled,
			&s.DeviceID,
		)
		if err != nil {
			return nil, err
		}

		s.CreatedAt = s.CreatedAt.UTC()

		ss = append(ss, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ss, nil
}

func convertOpts(opts QueryOptions) ([]string, []interface{}, error) {
	var (
		clauses = []string{}
		params  = []interface{}{}
	)

	if len(opts.DeviceIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.DeviceIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseDeviceIDs, ps)
		if err != nil {
			return nil, nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if opts.Enabled != nil {
		clause, _, err := sqlx.In(pgClauseEnabled, []interface{}{*opts.Enabled})
		if err != nil {
			return nil, nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, *opts.Enabled)
	}

	if len(opts.IDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.IDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseIDs, ps)
		if err != nil {
			return nil, nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if len(opts.UserIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.UserIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseUserIDs, ps)
		if err != nil {
			return nil, nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	return clauses, params, nil
}
