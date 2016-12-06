package platform

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/tapglue/snaas/platform/flake"
	"github.com/tapglue/snaas/platform/pg"
)

const (
	pgInsertPlatform = `INSERT INTO
		%s.platforms(active, app_id, arn, deleted, ecosystem, id, name, scheme, created_at, updated_at)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	pgUpdatePlatform = `
		UPDATE
			%s.platforms
		SET
			active = $2,
			app_id = $3,
			arn = $4,
			deleted = $5,
			ecosystem = $6,
			name = $7,
			scheme = $8,
			updated_at = $9
		WHERE
			id = $1`

	pgClauseActive     = `active = ?`
	pgClauseAppIDs     = `app_id IN (?)`
	pgClauseARNs       = `arn IN (?)`
	pgClauseDeleted    = `deleted = ?`
	pgClauseEcosystems = `ecosystem IN (?)`
	pgClauseIDs        = `id IN (?)`

	pgListPlatforms = `
		SELECT
			active, app_id, arn, deleted, ecosystem, id, name, scheme, created_at, updated_at
		FROM
			%s.platforms
		%s`

	pgOrderCreatedAt = `ORDER BY created_at DESC`

	pgCreateSchema = `CREATE SCHEMA IF NOT EXISTS %s`
	pgCreateTable  = `CREATE TABLE IF NOT EXISTS %s.platforms(
		active BOOL DEFAULT false,
		app_id BIGINT NOT NULL,
		arn TEXT NOT NULL,
		deleted BOOL DEFAULT false,
		ecosystem INT NOT NULL,
		id BIGINT NOT NULL UNIQUE,
		name CITEXT NOT NULL UNIQUE,
		scheme TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`
	pgDropTable = `DROP TABLE IF EXISTS %s.platforms`
)

type pgService struct {
	db *sqlx.DB
}

// PostgresService returns a Postgres based Service implementation.
func PostgresService(db *sqlx.DB) Service {
	return &pgService{
		db: db,
	}
}

func (s *pgService) Put(ns string, p *Platform) (*Platform, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	if p.ID == 0 {
		return s.insert(ns, p)
	}

	return s.update(ns, p)
}

func (s *pgService) Query(ns string, opts QueryOptions) (List, error) {
	where, params, err := convertOpts(opts)
	if err != nil {
		return nil, err
	}

	ps, err := s.listPlatforms(ns, where, params...)
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return nil, err
			}
		}

		ps, err = s.listPlatforms(ns, where, params...)
	}

	return ps, err
}

func (s *pgService) Setup(ns string) error {
	qs := []string{
		fmt.Sprintf(pgCreateSchema, ns),
		fmt.Sprintf(pgCreateTable, ns),
	}

	for _, q := range qs {
		_, err := s.db.Exec(q)
		if err != nil {
			return fmt.Errorf("setup (%s): %s", q, err)
		}
	}

	return nil
}

func (s *pgService) Teardown(ns string) error {
	qs := []string{
		fmt.Sprintf(pgDropTable, ns),
	}

	for _, q := range qs {
		_, err := s.db.Exec(q)
		if err != nil {
			return fmt.Errorf("teardown '%s': %s", q, err)
		}
	}

	return nil
}

func (s *pgService) insert(ns string, p *Platform) (*Platform, error) {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}

	ts, err := time.Parse(pg.TimeFormat, p.CreatedAt.UTC().Format(pg.TimeFormat))
	if err != nil {
		return nil, err
	}

	p.CreatedAt = ts
	p.UpdatedAt = ts

	id, err := flake.NextID(flakeNamespace(ns))
	if err != nil {
		return nil, err
	}

	p.ID = id

	var (
		params = []interface{}{
			p.Active,
			p.AppID,
			p.ARN,
			p.Deleted,
			p.Ecosystem,
			p.ID,
			p.Name,
			p.Scheme,
			p.CreatedAt,
			p.UpdatedAt,
		}
		query = fmt.Sprintf(pgInsertPlatform, ns)
	)

	_, err = s.db.Exec(query, params...)
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return nil, err
			}

			_, err = s.db.Exec(query, params...)
		}
	}

	return p, err
}

func (s *pgService) listPlatforms(
	ns, where string,
	params ...interface{},
) (List, error) {
	query := fmt.Sprintf(pgListPlatforms, ns, where)

	rows, err := s.db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ps := List{}

	for rows.Next() {
		p := &Platform{}

		err := rows.Scan(
			&p.Active,
			&p.AppID,
			&p.ARN,
			&p.Deleted,
			&p.Ecosystem,
			&p.ID,
			&p.Name,
			&p.Scheme,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		p.CreatedAt = p.CreatedAt.UTC()
		p.UpdatedAt = p.UpdatedAt.UTC()

		ps = append(ps, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ps, nil
}

func (s *pgService) update(ns string, p *Platform) (*Platform, error) {
	now, err := time.Parse(pg.TimeFormat, time.Now().UTC().Format(pg.TimeFormat))
	if err != nil {
		return nil, err
	}

	p.UpdatedAt = now

	var (
		params = []interface{}{
			p.ID,
			p.Active,
			p.AppID,
			p.ARN,
			p.Deleted,
			p.Ecosystem,
			p.Name,
			p.Scheme,
			p.UpdatedAt,
		}
		query = fmt.Sprintf(pgUpdatePlatform, ns)
	)

	_, err = s.db.Exec(query, params...)
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return nil, err
			}

			_, err = s.db.Exec(query, params...)
		}
	}

	return p, err
}

func convertOpts(opts QueryOptions) (string, []interface{}, error) {
	var (
		clauses = []string{}
		params  = []interface{}{}
	)

	if opts.Active != nil {
		clause, _, err := sqlx.In(pgClauseActive, []interface{}{*opts.Active})
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, *opts.Active)
	}

	if len(opts.AppIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.AppIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseAppIDs, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if len(opts.ARNs) > 0 {
		ps := []interface{}{}

		for _, arn := range opts.ARNs {
			ps = append(ps, arn)
		}

		clause, _, err := sqlx.In(pgClauseARNs, ps)
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

	if len(opts.Ecosystems) > 0 {
		ps := []interface{}{}

		for _, e := range opts.Ecosystems {
			ps = append(ps, e)
		}

		clause, _, err := sqlx.In(pgClauseEcosystems, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
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

	where := ""

	if len(clauses) > 0 {
		where = sqlx.Rebind(sqlx.DOLLAR, pg.ClausesToWhere(clauses...))
	}

	return where, params, nil
}
