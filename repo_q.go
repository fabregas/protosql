package protosql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type repoQ struct {
	r   *Repo
	ctx context.Context

	query    string
	alias    string
	filter   *Filter
	sortings Sortings
	pager    Pager
	joins    []join
}

type join struct {
	jtype string
	table string
	bindQ string
}

func (j join) String() string {
	return fmt.Sprintf("%s JOIN %s ON %s ", j.jtype, j.table, j.bindQ)
}

func (q *repoQ) As(alias string) *repoQ {
	q.alias = alias
	return q
}

func (q *repoQ) Where(f *Filter) *repoQ {
	q.filter = f
	return q
}

func (q *repoQ) OrderBy(s Sortings) *repoQ {
	q.sortings = s
	return q
}

func (q *repoQ) Paginate(p Pager) *repoQ {
	q.pager = p
	return q
}

func (q *repoQ) LeftJoin(table, bindQ string) *repoQ {
	q.joins = append(q.joins, join{"LEFT", table, bindQ})
	return q
}

func (q *repoQ) FetchOne(o Model) error {
	rows, err := q.exec()
	if err != nil {
		return err
	}

	defer rows.Close()
	if !rows.Next() {
		return ErrNotFound
	}

	if err := scanObj(rows, o); err != nil {
		return err
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func (q *repoQ) Fetch(o interface{}) error {
	rows, err := q.exec()
	if err != nil {
		return err
	}

	return scanObjects(rows, o)
}

func (q *repoQ) exec() (*sql.Rows, error) {
	wq, args, err := q.filter.WhereQuery()
	if err != nil {
		return nil, err
	}

	if q.sortings != nil {
		wq += q.sortings.SortQuery()
	}

	if q.pager != nil {
		wq += PageQuery(q.pager)
	}

	baseQuery := q.query
	if baseQuery == "" {
		baseQuery = q.r.selectQuery(q.alias)
	}

	for _, j := range q.joins {
		baseQuery += j.String()
	}

	q.r.logger.Debugf("QUERY: %s, ARGS: %+v", baseQuery+wq, args)

	rows, err := q.r.getDB(q.ctx).QueryContext(q.ctx, baseQuery+wq, args...)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func scanObjects(rows *sql.Rows, o interface{}) error {
	defer rows.Close()

	if reflect.TypeOf(o).Kind() != reflect.Ptr {
		return fmt.Errorf("ptr to slice should be passed for Scan()")
	}

	lst := reflect.ValueOf(o).Elem()
	if lst.Type().Kind() != reflect.Slice {
		return fmt.Errorf("invalid object type for scanner")
	}

	oType := lst.Type().Elem()
	if oType.Kind() != reflect.Ptr {
		return fmt.Errorf("slice element should be a pointer for scan")
	}

	for rows.Next() {
		obj := reflect.New(oType.Elem())
		oi, ok := obj.Interface().(Model)

		if !ok {
			return fmt.Errorf("invalid message type")
		}

		if err := scanObj(rows, oi); err != nil {
			return err
		}

		lst.Set(reflect.Append(lst, obj))
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanObj(s scanner, obj Model) error {
	m := parseProtoMsg(obj)

	var dest []interface{}
	for _, f := range m {
		var v interface{}
		switch f.val.Interface().(type) {
		case timeIface:
			t, ok := f.val.Addr().Interface().(**timestamppb.Timestamp)
			if !ok {
				return fmt.Errorf("invalid Timestamp type")
			}
			v = &timeScanner{t}
		default:
			v = f.val.Addr().Interface()
		}

		dest = append(dest, v)
	}

	return s.Scan(dest...)
}

type timeScanner struct {
	t **timestamppb.Timestamp
}

func (s *timeScanner) Scan(src interface{}) error {
	v, ok := src.(time.Time)
	if !ok {
		return fmt.Errorf("invalid value for timestamp: %v", src)
	}

	*s.t = timestamppb.New(v)
	return nil
}
