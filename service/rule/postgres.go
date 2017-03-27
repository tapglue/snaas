package rule

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/tapglue/snaas/platform/flake"
	"github.com/tapglue/snaas/platform/pg"
)

const (
	pgInsertRule = `INSERT INTO
		%s.rules(active, criteria, deleted, ecosystem, id, name, recipients, type, created_at, updated_at)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	pgUpdateRule = `
		UPDATE
			%s.rules
		SET
			active = $2,
			criteria = $3,
			deleted = $4,
			ecosystem = $5,
			name = $6,
			recipients = $7,
			type = $8,
			updated_at = $9
		WHERE
			id = $1`

	pgClauseActive  = `active = ?`
	pgClauseDeleted = `deleted = ?`
	pgClauseIDs     = `id IN (?)`
	pgClauseTypes   = `type IN (?)`

	pgListRules = `
		SELECT
			active, criteria, deleted, ecosystem, id, name, recipients, type, created_at, updated_at
		FROM
			%s.rules
		%s`
	pgOrderCreatedAt = `ORDER BY created_at DESC`

	pgCreateSchema = `CREATE SCHEMA IF NOT EXISTS %s`
	pgCreateTable  = `CREATE TABLE IF NOT EXISTS %s.rules(
		active BOOL DEFAULT false,
		criteria JSONB NOT NULL,
		deleted BOOL DEFAULT false,
		ecosystem INT,
		id BIGINT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		recipients JSONB NOT NULL,
		type INT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`
	pgDropTable = `DROP TABLE IF EXISTS %s.rules`
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

func (s *pgService) Put(ns string, r *Rule) (*Rule, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	if r.ID == 0 {
		return s.insert(ns, r)
	}

	return s.update(ns, r)
}

func (s *pgService) Query(ns string, opts QueryOptions) (List, error) {
	where, params, err := convertOpts(opts)
	if err != nil {
		return nil, err
	}

	rs, err := s.listRules(ns, where, params...)
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return nil, err
			}
		}

		rs, err = s.listRules(ns, where, params...)
	}

	return rs, err
}

func (s *pgService) Setup(ns string) error {
	qs := []string{
		fmt.Sprintf(pgCreateSchema, ns),
		fmt.Sprintf(pgCreateTable, ns),
	}

	for _, q := range qs {
		_, err := s.db.Exec(q)
		if err != nil {
			return fmt.Errorf("setup '%s': %s", q, err)
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

func (s *pgService) insert(ns string, r *Rule) (*Rule, error) {
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}

	ts, err := time.Parse(pg.TimeFormat, r.CreatedAt.UTC().Format(pg.TimeFormat))
	if err != nil {
		return nil, err
	}

	r.CreatedAt = ts
	r.UpdatedAt = ts

	id, err := flake.NextID(flakeNamespace(ns))
	if err != nil {
		return nil, err
	}

	r.ID = id

	criteria, err := json.Marshal(r.Criteria)
	if err != nil {
		return nil, err
	}

	recipients, err := json.Marshal(r.Recipients)
	if err != nil {
		return nil, err
	}

	var (
		params = []interface{}{
			r.Active,
			criteria,
			r.Deleted,
			r.Ecosystem,
			r.ID,
			r.Name,
			recipients,
			r.Type,
			r.CreatedAt,
			r.UpdatedAt,
		}
		query = fmt.Sprintf(pgInsertRule, ns)
	)

	_, err = s.db.Exec(query, params...)
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return nil, err
			}

			_, err = s.db.Exec(query, params...)
		} else {
			return nil, err
		}
	}

	return r, err
}

func (s *pgService) listRules(
	ns, where string,
	params ...interface{},
) (List, error) {
	query := fmt.Sprintf(pgListRules, ns, where)

	rows, err := s.db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rs := List{}

	for rows.Next() {
		var (
			criteria   = []byte{}
			recipients = []byte{}
			r          = &Rule{}
		)

		err := rows.Scan(
			&r.Active,
			&criteria,
			&r.Deleted,
			&r.Ecosystem,
			&r.ID,
			&r.Name,
			&recipients,
			&r.Type,
			&r.CreatedAt,
			&r.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		switch r.Type {
		case TypeConnection:
			r.Criteria = &CriteriaConnection{}
		case TypeEvent:
			r.Criteria = &CriteriaEvent{}
		case TypeObject:
			r.Criteria = &CriteriaObject{}
		case TypeReaction:
			r.Criteria = &CriteriaReaction{}
		default:
			return nil, fmt.Errorf("type not supported")
		}

		if err := json.Unmarshal(criteria, r.Criteria); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(recipients, &r.Recipients); err != nil {
			return nil, err
		}

		r.CreatedAt = r.CreatedAt.UTC()
		r.UpdatedAt = r.UpdatedAt.UTC()

		rs = append(rs, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rs, nil
}

func (s *pgService) update(ns string, r *Rule) (*Rule, error) {
	now, err := time.Parse(pg.TimeFormat, time.Now().UTC().Format(pg.TimeFormat))
	if err != nil {
		return nil, err
	}

	r.UpdatedAt = now

	criteria, err := json.Marshal(r.Criteria)
	if err != nil {
		return nil, err
	}

	recipients, err := json.Marshal(r.Recipients)
	if err != nil {
		return nil, err
	}

	var (
		params = []interface{}{
			r.ID,
			r.Active,
			criteria,
			r.Deleted,
			r.Ecosystem,
			r.Name,
			recipients,
			r.Type,
			r.UpdatedAt,
		}
		query = fmt.Sprintf(pgUpdateRule, ns)
	)

	_, err = s.db.Exec(query, params...)
	if err != nil && pg.IsRelationNotFound(pg.WrapError(err)) {
		if err := s.Setup(ns); err != nil {
			return nil, err
		}

		_, err = s.db.Exec(query, params...)
	}

	return r, err
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

	if opts.Deleted != nil {
		clause, _, err := sqlx.In(pgClauseDeleted, []interface{}{*opts.Deleted})
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, *opts.Deleted)
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

	if len(opts.Types) > 0 {
		ps := []interface{}{}

		for _, t := range opts.Types {
			ps = append(ps, t)
		}

		clause, _, err := sqlx.In(pgClauseTypes, ps)
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
