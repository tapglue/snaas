package connection

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/tapglue/snaas/platform/pg"
)

const (
	orderNone ordering = iota
	orderCreatedAt
	orderUpdatedAt
)

const (
	pgDeleteConnection = `DELETE
		FROM %s.connections
		WHERE (json_data->>'user_from_id')::BIGINT = $1::BIGINT
		AND (json_data->>'user_to_id')::BIGINT = $2::BIGINT
		AND (json_data->>'type')::TEXT = $3::TEXT`
	pgInsertConnection = `INSERT INTO %s.connections(json_data) VALUES($1)`
	pgUpdateConnection = `UPDATE %s.connections
		SET json_data = $4
		WHERE (json_data->>'user_from_id')::BIGINT = $1::BIGINT
		AND (json_data->>'user_to_id')::BIGINT = $2::BIGINT
		AND (json_data->>'type')::TEXT = $3::TEXT`

	pgCountConnections = `SELECT count(json_data) FROM %s.connections
		%s`
	pgListConnections = `SELECT json_data FROM %s.connections
		%s`

	pgClauseAfter   = `json_data->>'updated_at' > ?`
	pgClauseBefore  = `json_data->>'updated_at' < ?`
	pgClauseEnabled = `(json_data->>'enabled')::BOOL = ?::BOOL`
	pgClauseFromIDs = `(json_data->>'user_from_id')::BIGINT IN (?)`
	pgClauseStates  = `(json_data->>'state')::TEXT IN (?)`
	pgClauseToIDs   = `(json_data->>'user_to_id')::BIGINT IN (?)`
	pgClauseTypes   = `(json_data->>'type')::TEXT IN (?)`

	pgOrderCreatedAt = `ORDER BY (json_data->>'created_at')::TIMESTAMP DESC`
	pgOrderUpdatedAt = `ORDER BY json_data->>'updated_at' DESC`

	pgIndexFollowConfirmed = `
		CREATE INDEX
			%s
		ON
			%s.connections(((json_data->>'user_to_id')::BIGINT))
		WHERE
			(json_data->>'enabled')::BOOL = true
			AND (json_data->>'state')::TEXT = 'confirmed'
			AND (json_data->>'type')::TEXT = 'follow'`
	pgIndexFollowingConfirmed = `
		CREATE INDEX
			%s
		ON
			%s.connections(((json_data->>'user_from_id')::BIGINT))
		WHERE
			(json_data->>'enabled')::BOOL = true
			AND (json_data->>'state')::TEXT = 'confirmed'
			AND (json_data->>'type')::TEXT = 'follow'`
	pgIndexFriendConfirmedOrigin = `
		CREATE INDEX
			%s
		ON
			%s.connections(((json_data->>'user_from_id')::BIGINT))
		WHERE
			(json_data->>'enabled')::BOOL = true
			AND (json_data->>'state')::TEXT = 'confirmed'
			AND (json_data->>'type')::TEXT = 'friend'`
	pgIndexFriendConfirmedTarget = `
		CREATE INDEX
			%s
		ON
			%s.connections(((json_data->>'user_to_id')::BIGINT))
		WHERE
			(json_data->>'enabled')::BOOL = true
			AND (json_data->>'state')::TEXT = 'confirmed'
			AND (json_data->>'type')::TEXT = 'friend'`
	pgIndexRelationFollow = `
		CREATE INDEX
			%s
		ON
			%s.connections(((json_data->>'user_from_id')::BIGINT), ((json_data->>'user_to_id')::BIGINT))
		WHERE
			(json_data->>'type')::TEXT = 'follow'`
	pgIndexRelationFriend = `
		CREATE INDEX
			%s
		ON
			%s.connections(((json_data->>'user_from_id')::BIGINT), ((json_data->>'user_to_id')::BIGINT))
		WHERE
			(json_data->>'type')::TEXT = 'friend'`

	pgCreateSchema = `CREATE SCHEMA IF NOT EXISTS %s`
	pgCreateTable  = `CREATE TABLE IF NOT EXISTS %s.connections
		(json_data JSONB NOT NULL)`
	pgDropTable = `DROP TABLE IF EXISTS %s.connections`
)

type ordering int

type pgService struct {
	db *sqlx.DB
}

// PostgresService returns a Postgres based Service implementation.
func PostgresService(db *sqlx.DB) Service {
	return &pgService{db: db}
}

func (s *pgService) Count(ns string, opts QueryOptions) (int, error) {
	where, params, err := convertOpts(opts, orderNone)
	if err != nil {
		return 0, err
	}

	return s.countConnections(ns, where, params...)
}

func (s *pgService) Put(ns string, con *Connection) (*Connection, error) {
	if err := con.Validate(); err != nil {
		return nil, err
	}

	var (
		now    = time.Now().UTC()
		params = []interface{}{con.FromID, con.ToID, string(con.Type)}

		query string
	)

	cs, err := s.Query(ns, QueryOptions{
		FromIDs: []uint64{
			con.FromID,
		},
		ToIDs: []uint64{
			con.ToID,
		},
		Types: []Type{
			con.Type,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(cs) > 0 {
		query = wrapNamespace(pgUpdateConnection, ns)

		con.CreatedAt = cs[0].CreatedAt
		con.UpdatedAt = now
	} else {
		params = []interface{}{}
		query = wrapNamespace(pgInsertConnection, ns)

		if con.CreatedAt.IsZero() {
			con.CreatedAt = now
		}

		if con.UpdatedAt.IsZero() {
			con.UpdatedAt = now
		}

		con.CreatedAt = con.CreatedAt.UTC()
		con.UpdatedAt = con.UpdatedAt.UTC()
	}

	data, err := json.Marshal(con)
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(query, append(params, data)...)
	if err != nil {
		return nil, err
	}

	return con, nil
}

func (s *pgService) Query(ns string, opts QueryOptions) (List, error) {
	where, params, err := convertOpts(opts, orderUpdatedAt)
	if err != nil {
		return nil, err
	}

	return s.listConnections(ns, where, params...)
}

func (s *pgService) Setup(ns string) error {
	qs := []string{
		wrapNamespace(pgCreateSchema, ns),
		wrapNamespace(pgCreateTable, ns),
		pg.GuardIndex(ns, "connection_follow_confirmed", pgIndexFollowConfirmed),
		pg.GuardIndex(ns, "connection_following_confirmed", pgIndexFollowingConfirmed),
		pg.GuardIndex(ns, "connection_friend_confirmed_origin", pgIndexFriendConfirmedOrigin),
		pg.GuardIndex(ns, "connection_friend_confirmed_target", pgIndexFriendConfirmedTarget),
		pg.GuardIndex(ns, "connection_relation_follow", pgIndexRelationFollow),
		pg.GuardIndex(ns, "connection_relation_friend", pgIndexRelationFriend),
	}

	for _, query := range qs {
		_, err := s.db.Exec(query)
		if err != nil {
			return fmt.Errorf("query (%s: %s", query, err)
		}
	}

	return nil
}

func (s *pgService) Teardown(ns string) error {
	_, err := s.db.Exec(wrapNamespace(pgDropTable, ns))
	return err
}

func (s *pgService) countConnections(
	ns, where string,
	params ...interface{},
) (int, error) {
	var (
		count = 0
		query = fmt.Sprintf(pgCountConnections, ns, where)
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

func (s *pgService) listConnections(
	ns, where string,
	params ...interface{},
) (List, error) {
	query := fmt.Sprintf(pgListConnections, ns, where)

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

	cs := List{}

	for rows.Next() {
		var (
			con = &Connection{}

			raw []byte
		)

		err := rows.Scan(&raw)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(raw, con)
		if err != nil {
			return nil, err
		}

		cs = append(cs, con)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return cs, nil
}

func convertOpts(opts QueryOptions, order ordering) (string, []interface{}, error) {
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

	if len(opts.FromIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.FromIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseFromIDs, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if len(opts.States) > 0 {
		ps := []interface{}{}

		for _, state := range opts.States {
			ps = append(ps, string(state))
		}

		clause, _, err := sqlx.In(pgClauseStates, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if len(opts.ToIDs) > 0 {
		ps := []interface{}{}

		for _, id := range opts.ToIDs {
			ps = append(ps, id)
		}

		clause, _, err := sqlx.In(pgClauseToIDs, ps)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		params = append(params, ps...)
	}

	if len(opts.Types) > 0 {
		ps := []interface{}{}

		for _, t := range opts.Types {
			ps = append(ps, string(t))
		}

		clause, _, err := sqlx.In(pgClauseTypes, ps)
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

	if order == orderUpdatedAt {
		query = fmt.Sprintf("%s\n%s", query, pgOrderUpdatedAt)
	}

	if opts.Limit > 0 {
		query = fmt.Sprintf("%s\nLIMIT %d", query, opts.Limit)
	}

	return query, params, nil
}

func wrapNamespace(query, namespace string) string {
	return fmt.Sprintf(query, namespace)
}
