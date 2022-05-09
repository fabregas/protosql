package protosql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type Repo struct {
	db     *sql.DB
	table  string
	fields []string
	logger Logger
}

func NewRepo(db *sql.DB, tableName string, obj Model, logger Logger) *Repo {
	return &Repo{table: tableName, db: db, fields: objFields(obj), logger: logger}
}

func (r *Repo) Insert(ctx context.Context, obj Model) error {
	ts := timestamppb.Now()
	trySetTime(obj, "CreateTime", ts)
	trySetTime(obj, "UpdateTime", ts)

	q, params := insertQ(r.table, obj)

	r.logger.Debugf("QUERY: %s, ARGS: %+v", q, params)

	_, err := r.getDB(ctx).ExecContext(ctx, q, params...)

	return err
}

func (r *Repo) InsertDuplicateIgnore(ctx context.Context, obj Model) (bool, error) {
	ts := timestamppb.Now()
	trySetTime(obj, "CreateTime", ts)
	trySetTime(obj, "UpdateTime", ts)

	q, params := insertQ(r.table, obj)

	q += " ON CONFLICT(id) DO NOTHING"

	r.logger.Debugf("QUERY: %s, ARGS: %+v", q, params)

	res, err := r.getDB(ctx).ExecContext(ctx, q, params...)
	if err != nil {
		return false, err
	}

	ra, _ := res.RowsAffected()

	return ra > 0, err
}

func (r *Repo) Exec(ctx context.Context, q string, params ...interface{}) error {
	r.logger.Debugf("QUERY: %s, ARGS: %+v", q, params)

	_, err := r.getDB(ctx).ExecContext(ctx, q, params...)
	return err
}

func (r *Repo) UpdateByID(ctx context.Context, obj Model) error {
	trySetTime(obj, "UpdateTime", timestamppb.Now())

	q, params := updateQ(r.table, obj, "id")

	r.logger.Debugf("QUERY: %s, ARGS: %+v", q, params)

	_, err := r.getDB(ctx).ExecContext(ctx, q, params...)

	return err
}

func (r *Repo) Delete(ctx context.Context, f *Filter) error {
	wq, args, err := f.WhereQuery()
	if err != nil {
		return err
	}

	q := fmt.Sprintf("DELETE FROM %s%s", r.table, wq)

	r.logger.Debugf("QUERY: %s, ARGS: %+v", q, args)

	_, err = r.getDB(ctx).ExecContext(ctx, q, args...)

	return err
}

func (r *Repo) FindByID(ctx context.Context, id interface{}) *repoQ {
	q := &repoQ{r: r, ctx: ctx}
	return q.Where(NewFilter().Eq("id", id))
}

func (r *Repo) Select(ctx context.Context) *repoQ {
	return &repoQ{r: r, ctx: ctx}
}

func (r *Repo) SelectCustom(ctx context.Context, query string) *repoQ {
	return &repoQ{r: r, query: query, ctx: ctx}
}

func (r *Repo) SelectQuery() string {
	return r.selectQuery("")
}

func (r *Repo) selectQuery(alias string) string {
	var fields []string
	al := alias
	if al == "" {
		al = r.table
	}
	table := r.table
	if alias != "" {
		table += " AS " + alias
	}

	for _, f := range r.fields {
		fields = append(fields, fmt.Sprintf("%s.%s", al, f))
	}

	return fmt.Sprintf("SELECT %s FROM %s ", strings.Join(fields, ","), table)
}

func insertQ(table string, obj Model) (string, []interface{}) {
	m := parseProtoMsg(obj)
	paramNames, paramValues := toSqlParams(m)

	var placeholders []string
	for i := 0; i < len(paramNames); i++ {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(paramNames, ","),
		strings.Join(placeholders, ","),
	), paramValues
}

func updateQ(table string, obj Model, pkField string) (string, []interface{}) {
	m := parseProtoMsg(obj)
	paramNames, paramValues := toSqlParams(m)

	var (
		placeholders []string
		where        string
	)

	for i, param := range paramNames {
		if param == pkField {
			where = fmt.Sprintf("%s=$%d", pkField, i+1)
			continue
		}
		placeholders = append(placeholders, fmt.Sprintf("%s=$%d", param, i+1))
	}

	return fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(placeholders, ","),
		where,
	), paramValues
}

func objFields(obj Model) []string {
	m := parseProtoMsg(obj)
	var fields []string
	for _, field := range m {
		fields = append(fields, field.name)
	}

	return fields
}

type dbExec interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

func (r *Repo) getDB(ctx context.Context) dbExec {
	v := ctx.Value("_dbtx_")
	if v == nil {
		return r.db
	}

	tx, ok := v.(*sql.Tx)
	if !ok {
		panic("invalid TX type ?!")
	}

	return tx
}

func (r *Repo) Transaction(ctx context.Context, txFunc func(context.Context) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, "_dbtx_", tx)

	err = txFunc(ctx)

	if err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			r.logger.Errorf("rollback tx failed: %s", rerr)
		}

		return err
	}

	return tx.Commit()
}
