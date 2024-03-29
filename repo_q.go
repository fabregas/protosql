package protosql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/lib/pq"
	"google.golang.org/protobuf/types/known/durationpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type repoQ struct {
	r   *Repo
	ctx context.Context

	query   string
	alias   string
	lock    bool
	filter  *Filter
	sorting interface{}
	pager   Pager
	joins   []join
	groupBy []string
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

func (q *repoQ) GroupBy(fields ...string) *repoQ {
	q.groupBy = append(q.groupBy, fields...)
	return q
}

// s must be *Sorting or sorting proto message
func (q *repoQ) OrderBy(s interface{}) *repoQ {
	q.sorting = s
	return q
}

func (q *repoQ) Paginate(p Pager) *repoQ {
	if p == nil {
		// default pagination for preventing full table fetch with 'bad customer request'
		p = Page(0, 25)
	}
	q.pager = p
	return q
}

func (q *repoQ) Lock() *repoQ {
	q.lock = true
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

	if len(q.groupBy) > 0 {
		wq += fmt.Sprintf(" GROUP BY %s", strings.Join(q.groupBy, ","))
	}

	if q.sorting != nil {
		wq += sortQuery(newSorting(q.sorting))
	}

	if q.pager != nil {
		wq += pageQuery(q.pager)
	}

	baseQuery := q.query
	if baseQuery == "" {
		baseQuery = q.r.selectQuery(q.alias, nil)
	}

	for _, j := range q.joins {
		baseQuery += j.String()
	}

	if q.lock {
		wq += " FOR UPDATE"
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
		case durationIface:
			d, ok := f.val.Addr().Interface().(**durationpb.Duration)
			if !ok {
				return fmt.Errorf("invalid Duration type")
			}
			v = &durationScanner{d}
		default:
			switch f.val.Kind() {
			case reflect.Ptr, reflect.Map:
				v = &jsonScanner{f.val.Addr().Interface()}
			case reflect.Array, reflect.Slice:
				if f.val.Type().Elem().Kind() == reflect.Ptr {
					v = &jsonScanner{f.val.Addr().Interface()}
				} else {
					switch f.val.Interface().(type) {
					case []byte:
						v = f.val.Addr().Interface()
					default:
						v = &arrayScanner{f.val.Addr().Interface()}
					}
				}
			default:
				v = f.val.Addr().Interface()
			}
		}

		dest = append(dest, v)
	}

	return s.Scan(dest...)
}

type timeScanner struct {
	t **timestamppb.Timestamp
}

func (s *timeScanner) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	v, ok := src.(time.Time)
	if !ok {
		return fmt.Errorf("invalid value for timestamp: %v", src)
	}

	*s.t = timestamppb.New(v)
	return nil
}

type durationScanner struct {
	d **durationpb.Duration
}

func (s *durationScanner) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	v, ok := src.(int64)
	if !ok {
		return fmt.Errorf("invalid value for duration: %v", src)
	}

	*s.d = durationpb.New(time.Duration(v * 1000000)) // convert ms to ns
	return nil
}

type jsonScanner struct {
	dest interface{}
}

func (s *jsonScanner) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	var raw []byte
	switch v := src.(type) {
	case string:
		raw = []byte(v)
	case []byte:
		raw = v
	default:
		return fmt.Errorf("invalid value for json: %v", src)
	}

	return json.Unmarshal(raw, s.dest)
}

type arrayScanner struct {
	dest interface{}
}

func (s *arrayScanner) Scan(src interface{}) error {
	itemType := reflect.ValueOf(s.dest).Elem().Type().Elem()

	switch itemType.Kind() {
	case reflect.Int32:
		dest := []int32{}
		if err := pq.Array(&dest).Scan(src); err != nil {
			return err
		}

		//	ss := reflect.MakeSlice(reflect.SliceOf(itemType), 0, len(dest))
		ss := reflect.ValueOf(s.dest).Elem()
		for _, v := range dest {
			vv := reflect.New(itemType).Elem()
			vv.Set(reflect.ValueOf(v).Convert(itemType))
			ss = reflect.Append(ss, vv)
		}

		reflect.ValueOf(s.dest).Elem().Set(ss)

		return nil
	default:
		return pq.Array(s.dest).Scan(src)
	}
}
