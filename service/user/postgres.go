package user

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/tapglue/snaas/platform/flake"
	"github.com/tapglue/snaas/platform/pg"
)

const limitDefault = 200

const (
	pgInsertUser     = `INSERT INTO %s.users(json_data) VALUES($1)`
	pgUpdateLastRead = `
		UPDATE
			%s.users
		SET
			last_read = $2
		WHERE
			(json_data->>'id')::BIGINT = $1::BIGINT AND
			(json_data->>'enabled')::BOOL = true`
	pgUpdateUser = `
		UPDATE
			%s.users
		SET
			json_data = $1
		WHERE
			(json_data->>'id')::BIGINT = $2::BIGINT`

	pgClauseBefore    = `(json_data->>'id')::BIGINT > ?`
	pgClauseCustomIDs = `(json_data->>'custom_id')::TEXT IN (?)`
	pgClauseDeleted   = `(json_data->>'deleted')::BOOL = ?::BOOL`
	pgClauseEmail     = `(json_data->>'email')::CITEXT IN (?)`
	pgClauseEnabled   = `(json_data->>'enabled')::BOOL = ?::BOOL`
	pgClauseIDs       = `(json_data->>'id')::BIGINT IN (?)`
	pgClauseSocialIDs = `(json_data->'social_ids'->>'%s')::TEXT IN (?)`
	pgClauseUsernames = `(json_data->>'user_name')::CITEXT IN (?)`

	pgClauseSearchEmail     = `(json_data->>'email')::TEXT ILIKE '%%%s%%'`
	pgClauseSearchFirstname = `(json_data->>'first_name')::TEXT ILIKE '%%%s%%'`
	pgClauseSearchLastname  = `(json_data->>'last_name')::TEXT ILIKE '%%%s%%'`
	pgClauseSearchUsername  = `(json_data->>'user_name')::TEXT ILIKE '%%%s%%'`

	pgOrderCreatedAt = `json_data->>'created_at' DESC`
	pgOrderFirstname = `json_data->>'first_name' ASC`
	pgOrderLastname  = `json_data->>'first_naem' ASC`
	pgOrderUsername  = `json_data->>'user_name' ASC`

	pgCountUsers = `SELECT count(json_data) FROM %s.users
		%s`
	pgListUsers = `SELECT json_data, last_read FROM %s.users
		%s`

	pgCreateSchema = `CREATE SCHEMA IF NOT EXISTS %s`
	pgCreateTable  = `CREATE TABLE IF NOT EXISTS %s.users (
		json_data JSONB NOT NULL,
		last_read TIMESTAMP DEFAULT '0001-01-01 00:00:00 UTC' NOT NULL
	)`
	pgDropTable = `DROP TABLE IF EXISTS %s.users`

	pgIndexEmail = `
		CREATE UNIQUE INDEX
			%s
		ON
			%s.users(((json_data->>'email')::CITEXT))
		WHERE
			(json_data->>'enabled')::BOOL = true
			AND (json_data->>'email')::TEXT != ''`
	pgIndexID = `
		CREATE INDEX
			%s
		ON
			%s.users(((json_data->>'id')::BIGINT))
		WHERE
			(json_data->>'enabled')::BOOL = true`
	pgIndexSearch = `
		CREATE INDEX
			%s
		ON
			%s.users
		USING
			gin (
				((json_data->>'email')::TEXT) gin_trgm_ops,
				((json_data->>'first_name')::TEXT) gin_trgm_ops,
				((json_data->>'last_name')::TEXT) gin_trgm_ops,
				((json_data->>'user_name')::TEXT) gin_trgm_ops
			)
		WHERE
			(json_data->>'enabled')::BOOL = true`
	pgIndexUsername = `
		CREATE UNIQUE INDEX
			%s
		ON
			%s.users(((json_data->>'user_name')::CITEXT))
		WHERE
			(json_data->>'enabled')::BOOL = true
			AND (json_data->>'user_name')::TEXT != ''`
)

type pgService struct {
	db *sqlx.DB
}

// PostgresService returns a Postgres based Service implementation.
func PostgresService(db *sqlx.DB) Service {
	return &pgService{db: db}
}

func (s *pgService) Count(ns string, opts QueryOptions) (int, error) {
	where, params, err := convertOpts(opts)
	if err != nil {
		return 0, err
	}

	return s.countUsers(ns, where, params...)
}

func (s *pgService) Put(ns string, user *User) (*User, error) {
	var (
		now   = time.Now().UTC()
		query = pgUpdateUser

		params []interface{}
	)

	if err := user.Validate(); err != nil {
		return nil, err
	}

	if user.ID != 0 {
		params = []interface{}{
			user.ID,
		}

		us, err := s.Query(ns, QueryOptions{
			IDs: []uint64{
				user.ID,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(us) == 0 {
			return nil, ErrNotFound
		}

		user.CreatedAt = us[0].CreatedAt
	} else {
		id, err := flake.NextID(flakeNamespace(ns))
		if err != nil {
			return nil, err
		}

		if user.CreatedAt.IsZero() {
			user.CreatedAt = now
		}
		user.ID = id
		user.LastRead = user.LastRead.UTC()

		query = pgInsertUser
	}

	user.UpdatedAt = now

	data, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	params = append([]interface{}{data}, params...)

	_, err = s.db.Exec(wrapNamespace(query, ns), params...)
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return nil, err
			}

			_, err = s.db.Exec(wrapNamespace(query, ns), params...)
		}

		if pg.IsNotUnique(pg.WrapError(err)) {
			return nil, ErrNotUnique
		}
	}

	return user, err
}

func (s *pgService) PutLastRead(ns string, userID uint64, ts time.Time) error {
	_, err := s.db.Exec(wrapNamespace(pgUpdateLastRead, ns), userID, ts.UTC())
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return err
			}

			_, err = s.db.Exec(pgUpdateLastRead, userID, ts.UTC())
		}
	}
	return err
}

func (s *pgService) Query(ns string, opts QueryOptions) (List, error) {
	where, params, err := convertOpts(opts)
	if err != nil {
		return nil, err
	}

	return s.listUsers(ns, where, params...)
}

func (s *pgService) Search(ns string, opts QueryOptions) (List, error) {
	where, params, err := convertSearchOpts(opts)
	if err != nil {
		return nil, err
	}

	return s.listUsers(ns, where, params...)
}

func (s *pgService) Setup(ns string) error {
	qs := []string{
		wrapNamespace(pgCreateSchema, ns),
		wrapNamespace(pgCreateTable, ns),
		pg.GuardIndex(ns, "user_email", pgIndexEmail),
		pg.GuardIndex(ns, "user_id", pgIndexID),
		pg.GuardIndex(ns, "user_search", pgIndexSearch),
		pg.GuardIndex(ns, "user_username", pgIndexUsername),
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
		wrapNamespace(pgDropTable, ns),
	}

	for _, query := range qs {
		_, err := s.db.Exec(query)
		if err != nil {
			return fmt.Errorf("query (%s): %s", query, err)
		}
	}

	return nil
}

func (s *pgService) countUsers(
	ns, where string,
	params ...interface{},
) (int, error) {
	var (
		count = 0
		query = fmt.Sprintf(pgCountUsers, ns, where)
	)

	err := s.db.Get(&count, query, params...)
	if err != nil && pg.IsRelationNotFound(pg.WrapError(err)) {
		if err := s.Setup(ns); err != nil {
			return 0, err
		}

		err = s.db.Get(&count, query, params...)
	}

	return count, err
}

func (s *pgService) listUsers(
	ns, where string,
	params ...interface{},
) (List, error) {
	query := fmt.Sprintf(pgListUsers, ns, where)

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

	us := List{}

	for rows.Next() {
		var (
			user = &User{}

			lastRead time.Time
			raw      []byte
		)

		if err := rows.Scan(&raw, &lastRead); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(raw, user); err != nil {
			return nil, err
		}

		user.LastRead = lastRead.UTC()

		us = append(us, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return us, nil
}

func convertOpts(opts QueryOptions) (string, []interface{}, error) {
	var (
		clauses = []string{}
		params  = []interface{}{}
	)

	if opts.Before > 0 {
		clauses = append(clauses, pgClauseBefore)
		params = append(params, opts.Before)
	}

	if len(opts.CustomIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.CustomIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseCustomIDs, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if opts.Deleted != nil {
		clause, _, err := sqlx.In(pgClauseDeleted, []interface{}{*opts.Deleted})
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, *opts.Deleted)
	}

	if len(opts.Emails) > 0 {
		ps := []interface{}{}

		for _, email := range opts.Emails {
			ps = append(ps, email)
		}

		clause, _, err := sqlx.In(pgClauseEmail, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if opts.Enabled != nil {
		clause, _, err := sqlx.In(pgClauseEnabled, []interface{}{*opts.Enabled})
		if err != nil {
			return "", nil, err
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
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if opts.SocialIDs != nil {
		for platform, ids := range opts.SocialIDs {
			ps := []interface{}{}

			for _, id := range ids {
				ps = append(ps, id)
			}

			clause, _, err := sqlx.In(fmt.Sprintf(pgClauseSocialIDs, platform), ps)
			if err != nil {
				return "", nil, err
			}

			clauses = append(clauses, clause)
			params = append(params, ps...)
		}
	}

	if len(opts.Usernames) > 0 {
		ps := []interface{}{}

		for _, username := range opts.Usernames {
			ps = append(ps, username)
		}

		clause, _, err := sqlx.In(pgClauseUsernames, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	query := ""

	if len(clauses) > 0 {
		query = sqlx.Rebind(sqlx.DOLLAR, pg.ClausesToWhere(clauses...))
	}

	if opts.Before > 0 {
		query = fmt.Sprintf(
			"%s\nORDER BY %s\n",
			query,
			strings.Join([]string{
				pgOrderUsername,
				pgOrderFirstname,
				pgOrderLastname,
			}, ",\n"),
		)
	}

	if opts.Limit > 0 {
		query = fmt.Sprintf("%s\nLIMIT %d", query, opts.Limit)
	}

	return query, params, nil
}

func convertSearchOpts(opts QueryOptions) (string, []interface{}, error) {
	var (
		clauses = []string{}
		params  = []interface{}{}
	)

	if opts.Before > 0 {
		clauses = append(clauses, pgClauseBefore)
		params = append(params, opts.Before)
	}

	if len(opts.CustomIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.CustomIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseCustomIDs, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if opts.Deleted != nil {
		clause, _, err := sqlx.In(pgClauseDeleted, []interface{}{*opts.Deleted})
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, *opts.Deleted)
	}

	if opts.Enabled != nil {
		clause, _, err := sqlx.In(pgClauseEnabled, []interface{}{*opts.Enabled})
		if err != nil {
			return "", nil, err
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
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if opts.SocialIDs != nil {
		for platform, ids := range opts.SocialIDs {
			ps := []interface{}{}

			for _, id := range ids {
				ps = append(ps, id)
			}

			clause, _, err := sqlx.In(fmt.Sprintf(pgClauseSocialIDs, platform), ps)
			if err != nil {
				return "", nil, err
			}

			clauses = append(clauses, clause)
			params = append(params, ps...)
		}
	}

	var (
		sClauses = []string{}
	)

	for _, t := range opts.Emails {
		t = strings.Replace(t, "'", "''", -1)
		sClauses = append(sClauses, fmt.Sprintf(pgClauseSearchEmail, t))
	}

	for _, t := range opts.Firstnames {
		t = strings.Replace(t, "'", "''", -1)
		sClauses = append(sClauses, fmt.Sprintf(pgClauseSearchFirstname, t))
	}

	for _, t := range opts.Lastnames {
		t = strings.Replace(t, "'", "''", -1)
		sClauses = append(sClauses, fmt.Sprintf(pgClauseSearchLastname, t))
	}

	for _, t := range opts.Usernames {
		t = strings.Replace(t, "'", "''", -1)
		sClauses = append(sClauses, fmt.Sprintf(pgClauseSearchUsername, t))
	}

	if len(sClauses) > 0 {
		sClause := fmt.Sprintf("(%s)", strings.Join(sClauses, "\nOR "))
		clauses = append(clauses, sClause)
	}

	query := ""

	if len(clauses) > 0 {
		query = sqlx.Rebind(sqlx.DOLLAR, pg.ClausesToWhere(clauses...))
	}

	if opts.Before > 0 {
		query = fmt.Sprintf(
			"%s\nORDER BY %s\n",
			query,
			strings.Join([]string{
				pgOrderUsername,
				pgOrderFirstname,
				pgOrderLastname,
			}, ",\n"),
		)
	}

	if opts.Limit > 0 {
		query = fmt.Sprintf("%s\nLIMIT %d", query, opts.Limit)
	}

	return query, params, nil
}

func wrapNamespace(query, namespace string) string {
	return fmt.Sprintf(query, namespace)
}
