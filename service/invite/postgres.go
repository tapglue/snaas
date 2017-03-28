package invite

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/tapglue/snaas/platform/flake"
	"github.com/tapglue/snaas/platform/pg"
)

const (
	pgInsertInvite = `INSERT INTO
		%s.invites(deleted, id, key, user_id, value, created_at, updated_at)
		VALUES($1, $2, $3, $4, $5, $6, $7)`
	pgUpdateInvite = `
		UPDATE
			%s.invites
		SET
			deleted = $2,
			updated_at = $3
		WHERE
			id = $1`

	pgClauseBefore  = `created_at < ?`
	pgClauseDeleted = `deleted = ?`
	pgClauseIDs     = `id IN (?)`
	pgClauseKeys    = `key IN (?)`
	pgClauseUserIDs = `user_id IN (?)`
	pgClauseValues  = `value IN (?)`

	pgListInvites = `
		SELECT
			deleted, id, key, user_id, value, created_at, updated_at
		FROM
			%s.invites
		%s`

	pgOrderCreatedAt = `ORDER BY created_at DESC`

	pgCreateScheme = `CREATE SCHEMA IF NOT EXISTS %s`
	pgCreateTable  = `CREATE TABLE IF NOT EXISTS %s.invites(
		deleted BOOL DEFAULT false,
		id BIGINT NOT NULL UNIQUE,
		key TEXT NOT NULL,
		user_id BIGINT NOT NULL,
		value TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`
	pgDropTable = `DROP TABLE IF EXISTS %s.invites`
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

func (s *pgService) Put(ns string, i *Invite) (*Invite, error) {
	if i.ID == 0 {
		return s.insert(ns, i)
	}

	return s.update(ns, i)
}

func (s *pgService) Query(ns string, opts QueryOptions) (List, error) {
	where, params, err := convertOpts(opts)
	if err != nil {
		return nil, err
	}

	return s.listInvites(ns, where, params...)
}

func (s *pgService) Setup(ns string) error {
	qs := []string{
		fmt.Sprintf(pgCreateScheme, ns),
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

func (s *pgService) insert(ns string, i *Invite) (*Invite, error) {
	if i.CreatedAt.IsZero() {
		i.CreatedAt = time.Now().UTC()
	}

	ts, err := time.Parse(pg.TimeFormat, i.CreatedAt.UTC().Format(pg.TimeFormat))
	if err != nil {
		return nil, err
	}

	i.CreatedAt = ts
	i.UpdatedAt = ts

	id, err := flake.NextID(flake.Namespace(ns, entity))
	if err != nil {
		return nil, err
	}

	i.ID = id

	var (
		params = []interface{}{
			i.Deleted,
			i.ID,
			i.Key,
			i.UserID,
			i.Value,
			i.CreatedAt,
			i.UpdatedAt,
		}
		query = fmt.Sprintf(pgInsertInvite, ns)
	)

	_, err = s.db.Exec(query, params...)
	if err != nil && pg.IsRelationNotFound(pg.WrapError(err)) {
		if err := s.Setup(ns); err != nil {
			return nil, err
		}

		_, err = s.db.Exec(query, params...)
	}

	return i, err
}

func (s *pgService) listInvites(
	ns, where string,
	params ...interface{},
) (List, error) {
	query := fmt.Sprintf(pgListInvites, ns, where)

	rows, err := s.db.Query(query, params...)
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return nil, err
			}

			return s.listInvites(ns, where, params...)
		}

		return nil, err
	}
	defer rows.Close()

	is := List{}

	for rows.Next() {
		invite := &Invite{}

		err := rows.Scan(
			&invite.Deleted,
			&invite.ID,
			&invite.Key,
			&invite.UserID,
			&invite.Value,
			&invite.CreatedAt,
			&invite.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		invite.CreatedAt = invite.CreatedAt.UTC()
		invite.UpdatedAt = invite.UpdatedAt.UTC()

		is = append(is, invite)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return is, nil
}

func (s *pgService) update(ns string, i *Invite) (*Invite, error) {
	now, err := time.Parse(pg.TimeFormat, time.Now().UTC().Format(pg.TimeFormat))
	if err != nil {
		return nil, err
	}

	i.UpdatedAt = now

	var (
		params = []interface{}{
			i.ID,
			i.Deleted,
			i.UpdatedAt,
		}
		query = fmt.Sprintf(pgUpdateInvite, ns)
	)

	_, err = s.db.Exec(query, params...)
	if err != nil && pg.IsRelationNotFound(pg.WrapError(err)) {
		if err := s.Setup(ns); err != nil {
			return nil, err
		}

		_, err = s.db.Exec(query, params...)
	}

	return i, err
}

func convertOpts(opts QueryOptions) (string, []interface{}, error) {
	var (
		clauses = []string{}
		params  = []interface{}{}
	)

	if !opts.Before.IsZero() {
		clauses = append(clauses, pgClauseBefore)
		params = append(params, opts.Before.UTC().Format(pg.TimeFormat))
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

	if len(opts.Keys) > 0 {
		ps := []interface{}{}

		for _, k := range opts.Keys {
			ps = append(ps, k)
		}

		clause, _, err := sqlx.In(pgClauseKeys, ps)
		if err != nil {
			return "", nil, err
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
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if len(opts.Values) > 0 {
		ps := []interface{}{}

		for _, v := range opts.Values {
			ps = append(ps, v)
		}

		clause, _, err := sqlx.In(pgClauseValues, ps)
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

	if !opts.Before.IsZero() {
		where = fmt.Sprintf("%s\n%s", where, pgOrderCreatedAt)
	}

	if opts.Limit > 0 {
		where = fmt.Sprintf("%s\nLIMIT %d", where, opts.Limit)
	}

	return where, params, nil
}
