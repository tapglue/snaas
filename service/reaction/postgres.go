package reaction

import (
	"fmt"
	"time"

	"github.com/tapglue/snaas/platform/flake"

	"github.com/jmoiron/sqlx"

	"github.com/tapglue/snaas/platform/pg"
)

const (
	pgInsertReaction = `
		INSERT INTO %s.reactions(
			deleted, id, object_id, owner_id, type, created_at, updated_at
		) VALUES(
			$1, $2, $3, $4, $5, $6, $7
		)`
	pgUpdateReaction = `
		UPDATE
			%s.reactions
		SET
			deleted = $2,
			updated_at = $3
		WHERE
			id = $1`

	pgCountReactions = `SELECT count(*) FROM %s.reactions %s`
	pgListReactions  = `
		SELECT
			deleted, id, object_id, owner_id, type, created_at, updated_at
		FROM
			%s.reactions
		%s`

	pgClauseBefore    = `updated_at < ?`
	pgClauseDeleted   = `deleted = ?`
	pgClauseIDs       = `id IN (?)`
	pgClauseObjectIDs = `object_id IN (?)`
	pgClauseOwnerIDs  = `owner_id IN (?)`
	pgClauseTypes     = `type IN (?)`

	pgOrderUpdatedAt = `ORDER BY updated_at DESC`

	pgCreateSchema = `CREATE SCHEMA IF NOT EXISTS %s`
	pgCreateTable  = `CREATE TABLE IF NOT EXISTS %s.reactions(
		deleted BOOL DEFAULT false,
		id BIGINT PRIMARY KEY,
		object_id BIGINT NOT NULL,
		owner_id BIGINT NOT NULL,
		type INT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`
	pgDropTable = `DROP TABLE IF EXISTS %s.reactions`

	pgIndexObjectByType = `
		CREATE INDEX
			%s
		ON
			%s.reactions
		USING
			btree(object_id, updated_at DESC)
		WHERE
			deleted = false
			AND type = %d`
	pgIndexOwner = `
		CREATE INDEX
			%s
		ON
			%s.reactions
		USING
			btree(object_id, owner_id, type)`
)

type pgService struct {
	db *sqlx.DB
}

// PostgresService returns a Postgres based Service inplementation.
func PostgresService(db *sqlx.DB) Service {
	return &pgService{
		db: db,
	}
}

func (s *pgService) Count(ns string, opts QueryOptions) (uint, error) {
	where, params, err := convertOpts(opts)
	if err != nil {
		return 0, err
	}

	return s.countEvents(ns, where, params...)
}

func (s *pgService) Put(ns string, r *Reaction) (*Reaction, error) {
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

	return s.listEvents(ns, where, params...)
}

func (s *pgService) Setup(ns string) error {
	qs := []string{
		fmt.Sprintf(pgCreateSchema, ns),
		fmt.Sprintf(pgCreateTable, ns),

		// Indexes
		pg.GuardIndex(ns, "reaction_object_like", pgIndexObjectByType, TypeLike),
		pg.GuardIndex(ns, "reaction_object_love", pgIndexObjectByType, TypeLove),
		pg.GuardIndex(ns, "reaction_object_haha", pgIndexObjectByType, TypeHaha),
		pg.GuardIndex(ns, "reaction_object_wow", pgIndexObjectByType, TypeWow),
		pg.GuardIndex(ns, "reaction_object_sad", pgIndexObjectByType, TypeSad),
		pg.GuardIndex(ns, "reaction_object_angry", pgIndexObjectByType, TypeAngry),
		pg.GuardIndex(ns, "reaction_owner_type", pgIndexOwner),
	}

	for _, q := range qs {
		if _, err := s.db.Exec(q); err != nil {
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
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("teardown '%s': %s", q, err)
		}
	}

	return nil
}

func (s *pgService) countEvents(
	ns, where string,
	params ...interface{},
) (uint, error) {
	var (
		query = fmt.Sprintf(pgCountReactions, ns, where)

		count uint
	)

	err := s.db.Get(&count, query, params...)
	if err != nil && pg.IsRelationNotFound(pg.WrapError(err)) {
		if err := s.Setup(ns); err != nil {
			return count, err
		}

		err = s.db.Get(&count, query, params...)
	}

	return count, err
}

func (s *pgService) insert(ns string, r *Reaction) (*Reaction, error) {
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

	var (
		params = []interface{}{
			r.Deleted,
			r.ID,
			r.ObjectID,
			r.OwnerID,
			r.Type,
			r.CreatedAt,
			r.UpdatedAt,
		}
		query = fmt.Sprintf(pgInsertReaction, ns)
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

func (s *pgService) listEvents(
	ns, where string,
	params ...interface{},
) (List, error) {
	query := fmt.Sprintf(pgListReactions, ns, where)

	rows, err := s.db.Query(query, params...)
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return nil, err
			}

			return s.listEvents(ns, where, params...)
		}
	}
	defer rows.Close()

	rs := List{}

	for rows.Next() {
		var (
			reaction = &Reaction{}
		)

		err := rows.Scan(
			&reaction.Deleted,
			&reaction.ID,
			&reaction.ObjectID,
			&reaction.OwnerID,
			&reaction.Type,
			&reaction.CreatedAt,
			&reaction.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		reaction.CreatedAt = reaction.CreatedAt.UTC()
		reaction.UpdatedAt = reaction.UpdatedAt.UTC()

		rs = append(rs, reaction)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rs, nil
}

func (s *pgService) update(ns string, r *Reaction) (*Reaction, error) {
	now, err := time.Parse(pg.TimeFormat, time.Now().UTC().Format(pg.TimeFormat))
	if err != nil {
		return nil, err
	}

	r.UpdatedAt = now

	var (
		params = []interface{}{
			r.ID,
			r.Deleted,
			r.UpdatedAt,
		}
		query = fmt.Sprintf(pgUpdateReaction, ns)
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

	if len(opts.ObjectIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.ObjectIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseObjectIDs, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if len(opts.OwnerIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.OwnerIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseOwnerIDs, ps)
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

	if !opts.Before.IsZero() {
		where = fmt.Sprintf("%s\n%s", where, pgOrderUpdatedAt)
	}

	if opts.Limit > 0 {
		where = fmt.Sprintf("%s\nLIMIT %d", where, opts.Limit)
	}

	return where, params, nil
}
