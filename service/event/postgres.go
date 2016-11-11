package event

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/tapglue/snaas/platform/flake"
	"github.com/tapglue/snaas/platform/pg"
)

const (
	pgInsertEvent = `INSERT INTO %s.events(json_data) VALUES($1)`
	pgUpdateEvent = `UPDATE %s.events SET json_data = $1
		WHERE (json_data->>'id')::BIGINT = $2::BIGINT`
	pgDeleteEvent = `DELETE FROM %s.events
		WHERE (json_data->>'id')::BIGINT = $1::BIGINT`

	pgCountEvents = `SELECT count(json_data) FROM %s.events
		%s`
	pgListEvents = `SELECT json_data FROM %s.events
		%s`

	pgClauseAfter               = `(json_data->>'created_at') > ?`
	pgClauseBefore              = `(json_data->>'created_at') < ?`
	pgClauseEnabled             = `(json_data->>'enabled')::BOOL = ?::BOOL`
	pgClauseExternalObjectIDs   = `(json_data->'object'->>'id')::TEXT IN (?)`
	pgClauseExternalObjectTypes = `(json_data->'object'->>'type')::TEXT in (?)`
	pgClauseIDs                 = `(json_data->>'id')::BIGINT IN (?)`
	pgClauseObjectIDs           = `(json_data->>'object_id')::BIGINT IN (?)`
	pgClauseOwned               = `(json_data->>'owned')::BOOL = ?::BOOL`
	pgClauseTargetIDs           = `(json_data->'target'->>'id')::TEXT in (?)`
	pgClauseTargetTypes         = `(json_data->'target'->>'type')::TEXT in (?)`
	pgClauseTypes               = `(json_data->>'type')::TEXT in (?)`
	pgClauseUserIDs             = `(json_data->>'user_id')::BIGINT IN (?)`
	pgClauseVisibilities        = `(json_data->>'visibility')::INT IN (?)`

	pgOrderCreatedAt = `ORDER BY (json_data->>'created_at') DESC`

	pgCreateSchema = `CREATE SCHEMA IF NOT EXISTS %s`
	pgCreateTable  = `CREATE TABLE IF NOT EXISTS %s.events (
		json_data JSONB NOT NULL
	)`
	pgDropTable = `DROP TABLE IF EXISTS %s.events`

	pgIndexCreatedAt = `
		CREATE INDEX
			%s
		ON
			%s.events
		USING
			btree ((json_data->>'created_at') DESC)`
	pgIndexID = `
		CREATE INDEX
			%s
		ON
			%s.events
		USING
			btree (((json_data->>'id')::BIGINT))`
	pgIndexLikes = `
		CREATE INDEX
			%s
		ON
			%s.events
		USING
			btree (((json_data->>'obejct_id')::BIGINT), ((json_data->>'created_at')) DESC)
		WHERE
			(json_data->>'enabled')::BOOL = true
			AND (json_data->>'owned')::BOOL = true
			AND (json_data->>'type')::TEXT = 'tg_like'`
	pgIndexObjectID = `
		CREATE INDEX
			%s
		ON
			%s.events
		USING
			btree (((json_data->>'object_id')::BIGINT))`
	pgIndexOwned = `
		CREATE INDEX
			%s
		ON
			%s.events
		USING
			btree (((json_data->>'owned')::BOOL))`
	pgIndexType = `
		CREATE INDEX
			%s
		ON
			%s.events
		USING
			btree (((json_data->>'type')::TEXT))`
	pgIndexUserID = `
		CREATE INDEX
			%s
		ON
			%s.events
		USING
			btree (((json_data->>'user_id')::BIGINT))`
	pgIndexVisibility = `
		CREATE INDEX
			%s
		ON
			%s.events
		USING
			btree (((json_data->>'visibility')::INT))`
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

	return s.countEvents(ns, where, params...)
}

func (s *pgService) Put(ns string, event *Event) (*Event, error) {
	var (
		now   = time.Now().UTC()
		query = pgUpdateEvent

		params []interface{}
	)

	if err := event.Validate(); err != nil {
		return nil, err
	}

	if event.ID != 0 {
		params = []interface{}{
			event.ID,
		}

		es, err := s.Query(ns, QueryOptions{
			IDs: []uint64{
				event.ID,
			},
		})
		if err != nil {
			return nil, err
		}

		if len(es) == 0 {
			return nil, ErrNotFound
		}

		event.CreatedAt = es[0].CreatedAt
	} else {
		id, err := flake.NextID(flakeNamespace(ns))
		if err != nil {
			return nil, err
		}

		if event.CreatedAt.IsZero() {
			event.CreatedAt = now
		} else {
			event.CreatedAt = event.CreatedAt.UTC()
		}

		event.ID = id
		query = pgInsertEvent
	}

	event.UpdatedAt = now

	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	params = append([]interface{}{data}, params...)

	_, err = s.db.Exec(wrapNamespace(query, ns), params...)
	if err != nil && pg.IsRelationNotFound(pg.WrapError(err)) {
		if err := s.Setup(ns); err != nil {
			return nil, err
		}

		_, err = s.db.Exec(wrapNamespace(query, ns), params...)
	}

	return event, err
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
		wrapNamespace(pgCreateSchema, ns),
		wrapNamespace(pgCreateTable, ns),
		pg.GuardIndex(ns, "event_created_at", pgIndexCreatedAt),
		pg.GuardIndex(ns, "event_id", pgIndexID),
		pg.GuardIndex(ns, "event_likes", pgIndexLikes),
		pg.GuardIndex(ns, "event_object_id", pgIndexObjectID),
		pg.GuardIndex(ns, "event_owned", pgIndexOwned),
		pg.GuardIndex(ns, "event_type", pgIndexType),
		pg.GuardIndex(ns, "event_user_id", pgIndexUserID),
		pg.GuardIndex(ns, "event_visibility", pgIndexVisibility),
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

func (s *pgService) countEvents(
	ns string, where string,
	params ...interface{},
) (int, error) {
	var (
		count = 0
		query = fmt.Sprintf(pgCountEvents, ns, where)
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

func (s *pgService) listEvents(
	ns string, where string,
	params ...interface{},
) (List, error) {
	query := fmt.Sprintf(pgListEvents, ns, where)

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
		}

		return nil, err
	}
	defer rows.Close()

	es := List{}

	for rows.Next() {
		var (
			event = &Event{}

			raw []byte
		)

		err := rows.Scan(&raw)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(raw, event)
		if err != nil {
			return nil, err
		}

		es = append(es, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return es, nil
}

func convertOpts(opts QueryOptions) (string, []interface{}, error) {
	var (
		clauses = []string{}
		params  = []interface{}{}
	)

	if !opts.After.IsZero() {
		clauses = append(clauses, pgClauseAfter)
		params = append(params, opts.After.UTC().Format(time.RFC3339Nano))
	}

	if !opts.Before.IsZero() {
		clauses = append(clauses, pgClauseBefore)
		params = append(params, opts.Before.UTC().Format(time.RFC3339Nano))
	}

	if opts.Enabled != nil {
		clause, _, err := sqlx.In(pgClauseEnabled, []interface{}{*opts.Enabled})
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, *opts.Enabled)
	}

	if len(opts.ExternalObjectIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.ExternalObjectIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseExternalObjectIDs, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if len(opts.ExternalObjectTypes) > 0 {
		ps := []interface{}{}

		for _, t := range opts.ExternalObjectTypes {
			ps = append(ps, t)
		}

		clause, _, err := sqlx.In(pgClauseExternalObjectTypes, ps)
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

	if opts.Owned != nil {
		clause, _, err := sqlx.In(pgClauseOwned, []interface{}{*opts.Owned})
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, *opts.Owned)
	}

	if len(opts.TargetIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.TargetIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseTargetIDs, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if len(opts.TargetTypes) > 0 {
		ps := []interface{}{}

		for _, t := range opts.TargetTypes {
			ps = append(ps, t)
		}

		clause, _, err := sqlx.In(pgClauseTargetTypes, ps)
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

	if len(opts.Visibilities) > 0 {
		ps := []interface{}{}

		for _, v := range opts.Visibilities {
			ps = append(ps, v)
		}

		clause, _, err := sqlx.In(pgClauseVisibilities, ps)
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
