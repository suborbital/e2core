package capabilities

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var (
	ErrDatabaseTypeInvalid = errors.New("database type invalid")
	ErrQueryNotFound       = errors.New("query not found")
	ErrQueryNotPrepared    = errors.New("query not prepared")
	ErrQueryTypeMismatch   = errors.New("query type incorrect")
	ErrQueryTypeInvalid    = errors.New("query type invalid")
	ErrQueryVarsMismatch   = errors.New("number of variables incorrect")
)

type DatabaseCapability interface {
	ExecQuery(queryType int32, name string, vars []interface{}) ([]byte, error)
	Prepare(q *Query) error
}

type DatabaseConfig struct {
	Enabled          bool    `json:"enabled" yaml:"enabled"`
	DBType           string  `json:"dbType" yaml:"dbType"`
	ConnectionString string  `json:"connectionString" yaml:"connectionString"`
	Queries          []Query `json:"queries" yaml:"queries"`
}

const (
	DBTypeMySQL    = "mysql"
	DBTypePostgres = "pgx"
)

type QueryType int32

const (
	QueryTypeInsert QueryType = QueryType(0)
	QueryTypeSelect QueryType = QueryType(1)
	QueryTypeUpdate QueryType = QueryType(2)
	QueryTypeDelete QueryType = QueryType(3)
)

type Query struct {
	Type     QueryType `json:"type" yaml:"type"`
	Name     string    `json:"name" yaml:"name"`
	VarCount int       `json:"varCount" yaml:"varCount"`
	Query    string    `json:"query" yaml:"query"`

	stmt *sqlx.Stmt `json:"-" yaml:"-"`
}

// SqlDatabase is an SQL implementation of DatabaseCapability
type SqlDatabase struct {
	config *DatabaseConfig
	db     *sqlx.DB

	queries map[string]*Query
}

type queryResult struct {
	LastInsertID int64 `json:"lastInsertID"`
	RowsAffected int64 `json:"rowsAffected"`
}

// NewSqlDatabase creates a new SQL database
func NewSqlDatabase(config *DatabaseConfig) (DatabaseCapability, error) {
	if !config.Enabled || config.ConnectionString == "" {
		return &SqlDatabase{config: config}, nil
	}

	if config.DBType != DBTypeMySQL && config.DBType != DBTypePostgres {
		return nil, ErrDatabaseTypeInvalid
	}

	augmentedConnString := AugmentedValFromEnv(config.ConnectionString)

	db, err := sqlx.Connect(config.DBType, augmentedConnString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Connect")
	}

	s := &SqlDatabase{
		config:  config,
		db:      db,
		queries: map[string]*Query{},
	}

	for i := range config.Queries {
		q := config.Queries[i]

		if err := s.Prepare(&q); err != nil {
			return nil, errors.Wrapf(err, "failed to Prepare query %s", q.Name)
		}
	}

	return s, nil
}

func (s *SqlDatabase) Prepare(q *Query) error {
	stmt, err := s.db.Preparex(q.Query)
	if err != nil {
		return errors.Wrap(err, "failed to Prepare")
	}

	q.stmt = stmt

	s.queries[q.Name] = q

	return nil
}

func (s *SqlDatabase) ExecQuery(queryType int32, name string, vars []interface{}) ([]byte, error) {
	// the returned data varies depending on the query type

	switch QueryType(queryType) {
	case QueryTypeInsert:
		return s.execInsertQuery(name, vars)
	case QueryTypeSelect:
		return s.execSelectQuery(name, vars)
	case QueryTypeUpdate:
		return s.execUpdateQuery(name, vars)
	case QueryTypeDelete:
		return s.execDeleteQuery(name, vars)
	}

	return nil, ErrQueryTypeInvalid
}

// execInsertQuery executes a prepared Insert query
func (s *SqlDatabase) execInsertQuery(name string, vars []interface{}) ([]byte, error) {
	if !s.config.Enabled {
		return nil, ErrCapabilityNotEnabled
	}

	query, exists := s.queries[name]
	if !exists {
		return nil, ErrQueryNotFound
	}

	if query.Type != QueryTypeInsert {
		return nil, ErrQueryTypeMismatch
	}

	if query.stmt == nil {
		return nil, ErrQueryNotPrepared
	}

	if query.VarCount != len(vars) {
		return nil, errors.Wrapf(ErrQueryVarsMismatch, "expected %d variables, got %d", query.VarCount, len(vars))
	}

	result, err := query.stmt.Exec(vars...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Exec")
	}

	// no need to check error, if insertID is 0, that's fine
	insertID, _ := result.LastInsertId()

	insertResult := queryResult{
		LastInsertID: insertID,
	}

	resultJSON, err := json.Marshal(insertResult)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal result")
	}

	return resultJSON, nil
}

// execSelectQuery executes a prepared Select query
func (s *SqlDatabase) execSelectQuery(name string, vars []interface{}) ([]byte, error) {
	if !s.config.Enabled {
		return nil, ErrCapabilityNotEnabled
	}

	query, exists := s.queries[name]
	if !exists {
		return nil, ErrQueryNotFound
	}

	if query.Type != QueryTypeSelect {
		return nil, ErrQueryTypeMismatch
	}

	if query.stmt == nil {
		return nil, ErrQueryNotPrepared
	}

	if query.VarCount != len(vars) {
		return nil, errors.Wrapf(ErrQueryVarsMismatch, "expected %d variables, got %d", query.VarCount, len(vars))
	}

	rows, err := query.stmt.Query(vars...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to stmt.Query")
	}

	defer rows.Close()
	result, err := rowsToMap(rows)
	if err != nil {
		return nil, errors.Wrap(err, "failed to rowsToMap")
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal query result")
	}

	return resultJSON, nil
}

// execUpdateQuery executes a prepared Update query
func (s *SqlDatabase) execUpdateQuery(name string, vars []interface{}) ([]byte, error) {
	if !s.config.Enabled {
		return nil, ErrCapabilityNotEnabled
	}

	query, exists := s.queries[name]
	if !exists {
		return nil, ErrQueryNotFound
	}

	if query.Type != QueryTypeUpdate {
		return nil, ErrQueryTypeMismatch
	}

	if query.stmt == nil {
		return nil, ErrQueryNotPrepared
	}

	if query.VarCount != len(vars) {
		return nil, errors.Wrapf(ErrQueryVarsMismatch, "expected %d variables, got %d", query.VarCount, len(vars))
	}

	result, err := query.stmt.Exec(vars...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Exec")
	}

	// no need to check error, if rowsAffected is 0, that's fine
	rowsAffected, _ := result.RowsAffected()

	updateResult := queryResult{
		RowsAffected: rowsAffected,
	}

	resultJSON, err := json.Marshal(updateResult)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal result")
	}

	return resultJSON, nil
}

// execDeleteQuery executes a prepared Delete query
func (s *SqlDatabase) execDeleteQuery(name string, vars []interface{}) ([]byte, error) {
	if !s.config.Enabled {
		return nil, ErrCapabilityNotEnabled
	}

	query, exists := s.queries[name]
	if !exists {
		return nil, ErrQueryNotFound
	}

	if query.Type != QueryTypeDelete {
		return nil, ErrQueryTypeMismatch
	}

	if query.stmt == nil {
		return nil, ErrQueryNotPrepared
	}

	if query.VarCount != len(vars) {
		return nil, errors.Wrapf(ErrQueryVarsMismatch, "expected %d variables, got %d", query.VarCount, len(vars))
	}

	result, err := query.stmt.Exec(vars...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Exec")
	}

	// no need to check error, if rowsAffected is 0, that's fine
	rowsAffected, _ := result.RowsAffected()

	updateResult := queryResult{
		RowsAffected: rowsAffected,
	}

	resultJSON, err := json.Marshal(updateResult)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal result")
	}

	return resultJSON, nil
}

func rowsToMap(rows *sql.Rows) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Columns from query result")
	}

	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ColumnTypes from query result")
	}

	results := []map[string]interface{}{}

	for {
		if moreRows := rows.Next(); !moreRows {
			if rows.Err() != nil {
				return nil, errors.Wrap(err, "failed to rows.Next")
			}

			break
		}

		dest := make([]interface{}, len(cols))
		for i := range dest {
			val := typeFromDBType(types[i].DatabaseTypeName())
			dest[i] = &val
		}

		if err := rows.Scan(dest...); err != nil {
			return nil, errors.Wrap(err, "failed to Scan row")
		}

		result := map[string]interface{}{}

		for i, c := range cols {
			result[c] = dest[i]
		}

		results = append(results, result)
	}

	return results, nil
}

// converts a database type to a *non-pointer* Go type
func typeFromDBType(dbType string) interface{} {
	switch strings.ToLower(dbType) {
	case "varchar", "text", "nvarchar", "char", "uuid":
		return ""
	case "decimal", "float", "long":
		return 0.0
	case "int", "bigint", "number":
		return int64(0)
	case "timestamp", "datetime", "time", "date":
		return time.Time{}
	case "boolean", "bool":
		return true
	default:
		return []byte{}
	}
}
