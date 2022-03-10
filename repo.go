package protosql

import (
	"database/sql"
	"fmt"
	"strings"
)

type Repo struct {
	db     *sql.DB
	table  string
	fields []string
	logger Logger
}

func NewRepo(db *sql.DB, tableName string, obj Model, logger Logger) Repo {
	return Repo{table: tableName, db: db, fields: objFields(obj), logger: logger}
}

func (r Repo) Insert(obj Model) error {
	q, params := insertQ(r.table, obj)

	_, err := r.db.Exec(q, params...)

	return err
}

func (r Repo) UpdateByID(obj Model) error {
	q, params := updateQ(r.table, obj, "id")

	_, err := r.db.Exec(q, params...)

	return err
}

func (r Repo) GetByID(obj Model, id interface{}) error {
	q := &repoQ{r: &r}
	return q.Where(NewFilter().Eq("id", id)).FetchOne(obj)
}

func (r Repo) Select() *repoQ {
	return &repoQ{r: &r}
}

func (r Repo) SelectCustom(query string) *repoQ {
	return &repoQ{r: &r, query: query}
}

func (r Repo) SelectQuery() string {
	return r.selectQuery("")
}

func (r Repo) selectQuery(alias string) string {
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
