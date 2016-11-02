package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/tapglue/api/platform/pg"
)

const (
	pgInsertApp = `INSERT INTO %s.applications (account_id, json_data) VALUES($1, $2) RETURNING id`
	pgUpdateApp = `UPDATE %s.applications SET json_data = $3 WHERE account_id = $1 AND id = $2 RETURNING id`

	pgClauseBefore        = `(json_data->>'created_at')::TIMESTAMP < ?`
	pgClauseBackendTokens = `(json_data->>'backend_token')::TEXT IN (?)`
	pgClauseEnabled       = `(json_data->>'enabled')::BOOL = ?::BOOL`
	pgClauseIDs           = `id IN (?)`
	pgClauseInProduction  = `(json_data->>'in_production')::BOOL = ?::BOOL`
	pgClauseOrgIDs        = `account_id IN (?)`
	pgClausePublicIds     = `(json_data->>'id')::TEXT IN (?)`
	pgClauseTokens        = `(json_data->>'token')::TEXT IN (?)`

	pgListApps = `SELECT id, account_id, json_data FROM %s.applications
		%s`

	pgOrderCreatedAt = `ORDER BY (json_data->>'created_at')::TIMESTAMP DESC`

	pgCreateSchema = `CREATE SCHEMA IF NOT EXISTS %s`
	pgCreateTable  = `CREATE TABLE IF NOT EXISTS %s.applications (
	  id SERIAL PRIMARY KEY NOT NULL,
	  account_id INT NOT NULL,
	  json_data JSONB NOT NULL,
	  enabled INT DEFAULT 1 NOT NULL
	)`
	pgDropTable = `DROP TABLE IF EXISTS %s.applications`

	pgIndexAccuntID = `
		CREATE INDEX
			%s
		ON
			%s.applications(account_id, id)`
	pgIndexBackendToken = `
		CREATE INDEX
			%s
		ON
			%s.applications(((json_data->>'backend_token')::TEXT))
		WHERE
			(json_data->>'enabled')::BOOL = true`
	pgIndexPublicID = `
		CREATE INDEX
			%s
		ON
			%s.applications(((json_data->>'id')::TEXT))
		WHERE
			(json_data->>'enabled')::BOOL = true`
	pgIndexToken = `
		CREATE INDEX
			%s
		ON
			%s.applications(((json_data->>'token')::TEXT))
		WHERE
			(json_data->>'enabled')::BOOL = true`
)

type pgService struct {
	db *sqlx.DB
}

// PostgresService returns a Postgres based Service implementation.
func PostgresService(db *sqlx.DB) Service {
	return &pgService{db: db}
}

func (s *pgService) Put(ns string, input *App) (*App, error) {
	var (
		now    = time.Now().UTC()
		query  = pgUpdateApp
		params = []interface{}{input.OrgID}
	)

	if err := input.Validate(); err != nil {
		return nil, err
	}

	if input.ID != 0 {
		params = append(params, input.ID)

		as, err := s.Query(ns, QueryOptions{
			IDs: []uint64{
				input.ID,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(as) != 1 {
			return nil, ErrNotFound
		}

		input.CreatedAt = as[0].CreatedAt
	} else {
		query = pgInsertApp

		if input.CreatedAt.IsZero() {
			input.CreatedAt = now
		}

		input.CreatedAt = input.CreatedAt.UTC()
	}

	input.UpdatedAt = now

	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	query = wrapNamespace(query, ns)
	params = append(params, data)

	var id uint64

	err = s.db.QueryRow(query, params...).Scan(&id)
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}

		err = s.db.QueryRow(query, params...).Scan(&id)
	}

	input.ID = id

	return input, err
}

func (s *pgService) Query(ns string, opts QueryOptions) (List, error) {
	where, params, err := convertOpts(opts)
	if err != nil {
		return nil, err
	}

	return s.listApps(ns, where, params...)
}

func (s *pgService) Setup(ns string) error {
	qs := []string{
		wrapNamespace(pgCreateSchema, ns),
		wrapNamespace(pgCreateTable, ns),
		pg.GuardIndex(ns, "app_account_id", pgIndexAccuntID),
		pg.GuardIndex(ns, "app_backend_token", pgIndexBackendToken),
		pg.GuardIndex(ns, "app_public_id", pgIndexPublicID),
		pg.GuardIndex(ns, "app_token", pgIndexToken),
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

func (s *pgService) listApps(
	ns, where string,
	params ...interface{},
) (List, error) {
	query := fmt.Sprintf(pgListApps, ns, where)

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

	as := List{}

	for rows.Next() {
		var (
			app = &App{}

			id, orgID uint64
			raw       []byte
		)

		err := rows.Scan(&id, &orgID, &raw)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(raw, app)
		if err != nil {
			return nil, err
		}

		app.ID = id
		app.OrgID = orgID

		as = append(as, app)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return as, nil
}

func convertOpts(opts QueryOptions) (string, []interface{}, error) {
	var (
		clauses = []string{}
		params  = []interface{}{}
	)

	if !opts.Before.IsZero() {
		clauses = append(clauses, pgClauseBefore)
		params = append(params, opts.Before.UTC().Format(time.RFC3339Nano))
	}

	if len(opts.BackendTokens) > 0 {
		ps := []interface{}{}

		for _, t := range opts.BackendTokens {
			ps = append(ps, t)
		}

		clause, _, err := sqlx.In(pgClauseBackendTokens, ps)
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

	if opts.InProduction != nil {
		clause, _, err := sqlx.In(pgClauseInProduction, []interface{}{*opts.InProduction})
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, *opts.Enabled)
	}

	if len(opts.OrgIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.OrgIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseOrgIDs, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if len(opts.PublicIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.PublicIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClausePublicIds, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if len(opts.Tokens) > 0 {
		ps := []interface{}{}

		for _, t := range opts.Tokens {
			ps = append(ps, t)
		}

		clause, _, err := sqlx.In(pgClauseTokens, ps)
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

	if !opts.Before.IsZero() {
		query = fmt.Sprintf("%s\n%s", query, pgOrderCreatedAt)
	}

	if opts.Limit > 0 {
		query = fmt.Sprintf("%s\nLIMIT %d", query, opts.Limit)
	}

	return query, params, nil
}

func wrapNamespace(query, namespace string) string {
	return fmt.Sprintf(query, namespace)
}
