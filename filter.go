package protosql

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type TimestampValue interface {
	GetSeconds() int64
}

// Basic filter types
type StringValue interface{ GetValue() string }

type (
	Int64Value interface{ GetValue() int64 }
	Int32Value interface{ GetValue() int32 }
	IntValue   interface{ GetValue() int }
	BoolValue  interface{ GetValue() bool }
)

type sval string

func (s sval) GetValue() string { return string(s) }
func WrapString(s string) sval  { return sval(s) }

type operator int

const (
	eqOp operator = iota
	neqOp
	gtOp
	gteOp
	ltOp
	lteOp
	containOp
	inOp

	jsonArrInOp

	emptyStrOp
	notEmptyStrOp

	arrContainOp
	arrOverlapOp
	arrEmptyOp

	orOp

	rawOp
)

func (o operator) value() (s string) {
	switch o {
	case eqOp, emptyStrOp, arrEmptyOp:
		s = "="
	case neqOp, notEmptyStrOp:
		s = "!="
	case gtOp:
		s = ">"
	case gteOp:
		s = ">="
	case ltOp:
		s = "<"
	case lteOp:
		s = "<="
	case containOp:
		s = "ILIKE"
	case inOp:
		s = "IN"
	case jsonArrInOp:
		s = "?|"
	case arrContainOp:
		s = "@>"
	case arrOverlapOp:
		s = "&&"
	case rawOp:
		s = ""
	}

	return
}

var ignoreFilterErr = errors.New("filter no need")

type filterExpr struct {
	lval string
	op   operator
	rval interface{}
}

func (f filterExpr) formatStr(s string) string {
	if f.op == containOp {
		return fmt.Sprintf("%%%s%%", s)
	}

	return s
}

func (f filterExpr) format(gidx int) (string, []interface{}, error) {
	if f.op == rawOp {
		return f.lval, []interface{}{}, nil
	}

	if f.rval == nil {
		return "", nil, ignoreFilterErr
	}

	switch f.op {
	case orOp:
		stmt, args, err := f.rval.(*Filter).toQuery(gidx, "OR")
		if stmt == "" {
			return "", nil, ignoreFilterErr
		}
		return fmt.Sprintf("(%s)", stmt), args, err
	case emptyStrOp, notEmptyStrOp:
		return fmt.Sprintf("%s %s ''", f.lval, f.op.value()), nil, nil
	case arrEmptyOp:
		return fmt.Sprintf("COALESCE(array_length(%s, 1), 0) = 0", f.lval), nil, nil
	}

	if val := reflect.ValueOf(f.rval); val.Kind() == reflect.Ptr && val.IsNil() {
		return "", nil, ignoreFilterErr
	}

	var retList []interface{}
	switch v := f.rval.(type) {
	case int, int32, int64, bool:
		retList = append(retList, f.rval)
	case string:
		retList = append(retList, f.formatStr(v))
	case StringValue:
		if len(v.GetValue()) == 0 {
			return "", nil, ignoreFilterErr
		}
		retList = append(retList, f.formatStr(v.GetValue()))
	case Int64Value:
		retList = append(retList, v.GetValue())
	case Int32Value:
		retList = append(retList, v.GetValue())
	case IntValue:
		retList = append(retList, v.GetValue())
	case BoolValue:
		retList = append(retList, v.GetValue())
	case []int:
		for _, iv := range v {
			retList = append(retList, iv)
		}
	case []int32:
		for _, iv := range v {
			retList = append(retList, iv)
		}
	case []int64:
		for _, iv := range v {
			retList = append(retList, iv)
		}
	case []string:
		for _, sv := range v {
			retList = append(retList, sv)
		}
	case time.Time:
		retList = append(retList, v.UTC())
	case TimestampValue:
		if v.GetSeconds() == 0 {
			return "", nil, ignoreFilterErr
		}
		retList = append(retList, time.Unix(v.GetSeconds(), 0).UTC())

	case protoreflect.Enum:
		n := int32(v.Number())
		if n == 0 {
			// enum val with 0 must be UNSPECIFIED and should not be filtered
			return "", nil, ignoreFilterErr
		}
		retList = append(retList, n)
	default:
		return "", nil, fmt.Errorf("unexpected type of rval in SQL filter: %T", f.rval)
	}

	placeholders := fmt.Sprintf("$%d", gidx)
	switch f.op {
	case inOp, jsonArrInOp, arrContainOp, arrOverlapOp:
		if len(retList) == 0 {
			return "", nil, ignoreFilterErr
		}

		s := make([]string, len(retList))
		for i := 0; i < len(retList); i++ {
			s[i] = fmt.Sprintf("$%d", gidx+i)
		}

		switch f.op {
		case inOp:
			placeholders = fmt.Sprintf("(%s)", strings.Join(s, ", "))
		case jsonArrInOp:
			placeholders = fmt.Sprintf("array[%s]", strings.Join(s, ", "))
		case arrContainOp, arrOverlapOp:
			arrType := "text"
			switch reflect.TypeOf(retList[0]).Kind() {
			case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16, reflect.Uint32:
				arrType = "integer"
			case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64:
				arrType = "bigint"
			}
			placeholders = fmt.Sprintf("array[%s]::%s[]", strings.Join(s, ", "), arrType)
			f.lval = fmt.Sprintf("%s::%s[]", f.lval, arrType)
		}
	}

	return fmt.Sprintf("%s %s %s", f.lval, f.op.value(), placeholders), retList, nil
}

type Filter struct {
	exprList []filterExpr
}

func (f *Filter) addExpr(e filterExpr) {
	f.exprList = append(f.exprList, e)
}

func NewFilter() *Filter {
	return &Filter{}
}

func (f *Filter) Eq(lval string, rval interface{}) *Filter {
	f.addExpr(filterExpr{lval: lval, op: eqOp, rval: rval})
	return f
}

func (f *Filter) Neq(lval string, rval interface{}) *Filter {
	f.addExpr(filterExpr{lval: lval, op: neqOp, rval: rval})
	return f
}

func (f *Filter) In(lval string, rval interface{}) *Filter {
	f.addExpr(filterExpr{lval: lval, op: inOp, rval: rval})
	return f
}

func (f *Filter) Gt(lval string, rval interface{}) *Filter {
	f.addExpr(filterExpr{lval: lval, op: gtOp, rval: rval})
	return f
}

func (f *Filter) Gte(lval string, rval interface{}) *Filter {
	f.addExpr(filterExpr{lval: lval, op: gteOp, rval: rval})
	return f
}

func (f *Filter) Lt(lval string, rval interface{}) *Filter {
	f.addExpr(filterExpr{lval: lval, op: ltOp, rval: rval})
	return f
}

func (f *Filter) InRange(lval string, from, to interface{}) *Filter {
	f.addExpr(filterExpr{lval: lval, op: gteOp, rval: from})
	f.addExpr(filterExpr{lval: lval, op: ltOp, rval: to})
	return f
}

func (f *Filter) Lte(lval string, rval interface{}) *Filter {
	f.addExpr(filterExpr{lval: lval, op: lteOp, rval: rval})
	return f
}

func (f *Filter) Contain(lval string, rval interface{}) *Filter {
	f.addExpr(filterExpr{lval: lval, op: containOp, rval: rval})
	return f
}

func (f *Filter) JsonArrIn(lval string, rval interface{}) *Filter {
	f.addExpr(filterExpr{lval: lval, op: jsonArrInOp, rval: rval})
	return f
}

func (f *Filter) ArrContain(lval string, rval interface{}) *Filter {
	f.addExpr(filterExpr{lval: lval, op: arrContainOp, rval: rval})
	return f
}

func (f *Filter) ArrOverlap(lval string, rval interface{}) *Filter {
	f.addExpr(filterExpr{lval: lval, op: arrOverlapOp, rval: rval})
	return f
}

func (f *Filter) ArrEmpty(lval string) *Filter {
	f.addExpr(filterExpr{lval: lval, op: arrEmptyOp, rval: ""})
	return f
}

func (f *Filter) EmptyStr(lval string) *Filter {
	f.addExpr(filterExpr{lval: lval, op: emptyStrOp, rval: ""})
	return f
}

func (f *Filter) NotEmptyStr(lval string) *Filter {
	f.addExpr(filterExpr{lval: lval, op: notEmptyStrOp, rval: ""})
	return f
}

func (f *Filter) Or(orFilter *Filter) *Filter {
	f.addExpr(filterExpr{op: orOp, rval: orFilter})
	return f
}

func (f *Filter) Raw(cond string) *Filter {
	f.addExpr(filterExpr{lval: cond, op: rawOp, rval: nil})
	return f
}

func (f *Filter) WhereQuery() (string, []interface{}, error) {
	stmt, args, err := f.toQuery(1, "AND")
	if err != nil || stmt == "" {
		return stmt, args, err
	}

	return fmt.Sprintf(" WHERE %s ", stmt), args, nil
}
func (f *Filter) toQuery(startIdx int, op string) (string, []interface{}, error) {
	if f == nil {
		return "", nil, nil
	}

	var (
		whereList []string
		whereStmt string
		argsList  []interface{}
	)

	i := startIdx
	for _, e := range f.exprList {
		v, l, err := e.format(i)
		if err == ignoreFilterErr {
			continue
		}
		if err != nil {
			return "", nil, err
		}
		i += len(l)
		whereList = append(whereList, v)
		argsList = append(argsList, l...)
	}

	if len(whereList) > 0 {
		whereStmt = strings.Join(whereList, fmt.Sprintf(" %s ", op))
	}

	return whereStmt, argsList, nil
}
