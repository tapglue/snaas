package object

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/tapglue/snaas/platform/flake"
	"github.com/tapglue/snaas/platform/pg"
)

const (
	orderNone ordering = iota
	orderCreatedAt
)

const (
	pgInsertObject = `INSERT INTO %s.objects(json_data) VALUES($1)`
	pgUpdateObject = `UPDATE %s.objects SET json_data = $1
		WHERE (json_data->>'id')::BIGINT = $2::BIGINT`
	pgDeleteObject = `DELETE FROM %s.objects
		WHERE (json_data->>'id')::BIGINT = $1::BIGINT`

	pgCountObjects = `SELECT count(json_data) FROM %s.objects
		%s`
	pgCountObjectsMulti = `
		SELECT
			json_data->>'object_id',
			count(*)
		FROM
			%s.objects
		WHERE
			(json_data->>'deleted')::BOOL = false
			AND (json_data->>'object_id')::BIGINT IN (?)
			AND (json_data->>'owned')::BOOL = true
			AND (json_data->>'type')::TEXT = 'tg_comment'
		GROUP BY
			json_data->>'object_id'`
	pgListObjects = `SELECT json_data FROM %s.objects
		%s`

	pgClauseAfter      = `(json_data->>'created_at') > ?`
	pgClauseBefore     = `(json_data->>'created_at') < ?`
	pgClauseDeleted    = `(json_data->>'deleted')::BOOL = ?::BOOL`
	pgClauseExternalID = `(json_data->>'external_id')::TEXT IN (?)`
	pgClauseID         = `(json_data->>'id')::BIGINT = ?::BIGINT`
	pgClauseObjectID   = `(json_data->>'object_id')::BIGINT IN (?)`
	pgClauseOwnerID    = `(json_data->>'owner_id')::BIGINT IN (?)`
	pgClauseOwned      = `(json_data->>'owned')::BOOL = ?::BOOL`
	pgClauseTags       = `(json_data->'tags')::JSONB @> '[%s]'`
	pgClauseType       = `(json_data->>'type')::TEXT IN (?)`
	pgClauseVisibility = `(json_data->>'visibility')::INT IN (?)`
	pgOrderCreatedAt   = `ORDER BY json_data->>'created_at' DESC`

	pgCreateSchema = `CREATE SCHEMA IF NOT EXISTS %s`
	pgCreateTable  = `CREATE TABLE IF NOT EXISTS %s.objects
		(json_data JSONB NOT NULL)`

	pgCreateIndexCreatedAt = `CREATE INDEX %s ON %s.objects
		USING btree ((json_data->>'created_at') DESC)`
	pgCreateIndexExternalID = `CREATE INDEX %s ON %s.objects
		USING btree (((json_data->>'external_id')::TEXT))`
	pgCreateIndexID = `CREATE INDEX %s ON %s.objects
		USING btree (((json_data->>'id')::BIGINT))`
	pgCreateIndexObjectID = `CREATE INDEX %s ON %s.objects
		USING btree (((json_data->>'object_id')::BIGINT))`
	pgCreateIndexOwnerID = `CREATE INDEX %s ON %s.objects
		USING btree (((json_data->>'owner_id')::BIGINT))`
	pgCreateIndexOwned = `CREATE INDEX %s ON %s.objects
		USING btree (((json_data->>'owned')::BOOL))`
	pgCreateIndexTags = `CREATE INDEX %s ON %s.objects
		USING gin ((json_data->'tags'))`
	pgCreateIndexType = `CREATE INDEX %s ON %s.objects
		USING btree (((json_data->>'type')::TEXT))`
	pgCreateIndexVisibility = `CREATE INDEX %s ON %s.objects
		USING btree (((json_data->>'visibility')::INT))`
	pgCreateIndexPostAll = `
		CREATE INDEX
			%s
		ON
			%s.objects ((json_data->>'created_at') DESC)
		WHERE
			(json_data->>'deleted')::BOOL = false
		    AND (json_data->>'owned')::BOOL = true
		    AND (json_data->>'type')::TEXT IN ('tg_post')
		    AND (json_data->>'visibility')::INT IN (30, 40)`

	pgDropTable = `DROP TABLE IF EXISTS %s.objects`
)

type ordering int

type pgService struct {
	db *sqlx.DB
}

// PostgresService returns a Postgres based Service implementation.
func PostgresService(db *sqlx.DB) Service {
	return &pgService{
		db: db,
	}
}

func (s *pgService) Count(ns string, opts QueryOptions) (int, error) {
	where, params, err := convertOpts(opts, orderNone)
	if err != nil {
		return 0, err
	}

	return s.countObjects(ns, where, params...)
}

func (s *pgService) CountMulti(
	ns string,
	objectIDs ...uint64,
) (m CountsMap, err error) {
	var (
		countsMap = CountsMap{}
		ps        = []interface{}{}
	)

	if len(objectIDs) == 0 {
		return countsMap, nil
	}

	for _, id := range objectIDs {
		ps = append(ps, id)
	}

	query, _, err := sqlx.In(pgCountObjectsMulti, ps)
	if err != nil {
		return nil, err
	}

	query = sqlx.Rebind(sqlx.DOLLAR, query)
	query = fmt.Sprintf(query, ns)

	rows, err := s.db.Query(query, ps...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			objectID uint64
			count    uint64
		)

		err := rows.Scan(&objectID, &count)
		if err != nil {
			return nil, err
		}

		countsMap[objectID] = Counts{
			Comments: count,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return countsMap, nil
}

func (s *pgService) Put(ns string, object *Object) (*Object, error) {
	var (
		now   = time.Now().UTC()
		query = pgUpdateObject

		params []interface{}
	)

	if err := object.Validate(); err != nil {
		return nil, err
	}

	if object.ID != 0 {
		params = []interface{}{
			object.ID,
		}

		os, err := s.Query(ns, QueryOptions{
			ID: &object.ID,
		})
		if err != nil {
			return nil, err
		}

		if len(os) == 0 {
			return nil, ErrNotFound
		}

		object.CreatedAt = os[0].CreatedAt
	} else {
		id, err := flake.NextID(flakeNamespace(ns))
		if err != nil {
			return nil, err
		}

		if object.CreatedAt.IsZero() {
			object.CreatedAt = now
		} else {
			object.CreatedAt = object.CreatedAt.UTC()
		}

		object.ID = id
		query = pgInsertObject
	}

	object.UpdatedAt = now

	data, err := json.Marshal(object)
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
			if _, err := s.db.Exec(wrapNamespace(query, ns), params...); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return object, nil
}

func (s *pgService) Query(ns string, opts QueryOptions) (List, error) {
	where, params, err := convertOpts(opts, orderCreatedAt)
	if err != nil {
		return nil, err
	}

	return s.listObjects(ns, where, params...)
}

func (s *pgService) Setup(ns string) error {
	qs := []string{
		wrapNamespace(pgCreateSchema, ns),
		wrapNamespace(pgCreateTable, ns),
		pg.GuardIndex(ns, "object_created_at", pgCreateIndexCreatedAt),
		pg.GuardIndex(ns, "object_external_id", pgCreateIndexExternalID),
		pg.GuardIndex(ns, "object_id", pgCreateIndexID),
		pg.GuardIndex(ns, "object_object_id", pgCreateIndexObjectID),
		pg.GuardIndex(ns, "object_owned", pgCreateIndexOwned),
		pg.GuardIndex(ns, "object_owned_id", pgCreateIndexOwnerID),
		pg.GuardIndex(ns, "object_tags", pgCreateIndexTags),
		pg.GuardIndex(ns, "object_type", pgCreateIndexType),
		pg.GuardIndex(ns, "object_visibility", pgCreateIndexVisibility),
		pg.GuardIndex(ns, "object_post_all", pgCreateIndexPostAll),
	}

	for _, query := range qs {
		_, err := s.db.Exec(query)
		if err != nil {
			return fmt.Errorf("query (%s): %s", query, err)
		}
	}

	return nil
}

func (s *pgService) Teardown(namespace string) error {
	qs := []string{
		fmt.Sprintf(pgDropTable, namespace),
	}

	for _, query := range qs {
		_, err := s.db.Exec(query)
		if err != nil {
			return fmt.Errorf("query (%s): %s", query, err)
		}
	}

	return nil
}

func (s *pgService) countObjects(
	ns, where string,
	params ...interface{},
) (int, error) {
	var (
		count = 0
		query = fmt.Sprintf(pgCountObjects, ns, where)
	)

	err := s.db.Get(&count, query, params...)
	if err != nil {
		if pg.IsRelationNotFound(pg.WrapError(err)) {
			if err := s.Setup(ns); err != nil {
				return 0, err
			}

			err = s.db.Get(&count, query, params...)
		} else {
			return 0, err
		}
	}

	return count, err
}

func (s *pgService) listObjects(
	ns, where string,
	params ...interface{},
) (List, error) {
	query := fmt.Sprintf(pgListObjects, ns, where)

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

	os := List{}

	for rows.Next() {
		var (
			object = &Object{}

			raw []byte
		)

		err := rows.Scan(&raw)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(raw, object)
		if err != nil {
			return nil, err
		}

		os = append(os, object)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return os, nil
}

func convertOpts(opts QueryOptions, order ordering) (string, []interface{}, error) {
	var (
		clauses = []string{
			pgClauseDeleted,
		}
		params = []interface{}{
			opts.Deleted,
		}
	)

	if !opts.After.IsZero() {
		clauses = append(clauses, pgClauseAfter)
		params = append(params, opts.After.UTC().Format(time.RFC3339Nano))
	}

	if !opts.Before.IsZero() {
		clauses = append(clauses, pgClauseBefore)
		params = append(params, opts.Before.UTC().Format(time.RFC3339Nano))
	}

	if len(opts.ExternalIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.ExternalIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseExternalID, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if opts.ID != nil {
		params = append(params, *opts.ID)
		clauses = append(clauses, pgClauseID)
	}

	if len(opts.OwnerIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.OwnerIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseOwnerID, ps)
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

		clause, _, err := sqlx.In(pgClauseObjectID, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if opts.Owned != nil {
		clause, _, err := sqlx.In(pgClauseOwned, []interface{}{*opts.Owned})
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, *opts.Owned)
	}

	if len(opts.Tags) > 0 {
		ts := []string{}

		for _, t := range opts.Tags {
			ts = append(ts, fmt.Sprintf(`"%s"`, t))
		}

		clause := fmt.Sprintf(pgClauseTags, strings.Join(ts, ","))
		clauses = append(clauses, clause)
	}

	if len(opts.Types) > 0 {
		ps := []interface{}{}

		for _, id := range opts.Types {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseType, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if len(opts.Visibilities) > 0 {
		ps := []interface{}{}

		for _, v := range opts.Visibilities {
			ps = append(ps, v)
		}

		clause, _, err := sqlx.In(pgClauseVisibility, ps)
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

	if !opts.Before.IsZero() && order == orderCreatedAt {
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
