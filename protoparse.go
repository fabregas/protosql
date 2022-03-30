package protosql

import (
	"reflect"
	"strings"
	"time"
)

type Model interface {
	Reset()
	ProtoMessage()
}

type parsedField struct {
	name string
	val  reflect.Value
}

func parseProtoMsg(m Model) []parsedField {
	v := reflect.ValueOf(m)
	t := reflect.TypeOf(m)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	if !v.IsValid() {
		return nil
	}

	var r []parsedField

	for i := 0; i < t.NumField(); i++ {
		val, ok := t.Field(i).Tag.Lookup("protobuf")
		if !ok {
			val, ok = t.Field(i).Tag.Lookup("db")
			if !ok {
				continue
			}
		}
		r = append(r, parsedField{name: getNameFromTag(val), val: v.Field(i)})
	}

	return r
}

func toSqlParams(params []parsedField) ([]string, []interface{}) {
	var (
		names  []string
		values []interface{}
	)

	for _, p := range params {
		names = append(names, p.name)
		values = append(values, toSqlParam(p.val))
	}

	return names, values
}

type timeIface interface {
	AsTime() time.Time
}

type durationIface interface {
	AsDuration() time.Duration
}

func toSqlParam(v reflect.Value) interface{} {
	switch e := v.Interface().(type) {
	case timeIface:
		return e.AsTime()
	case durationIface:
		return e.AsDuration().Milliseconds()
	}

	switch v.Type().Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Array, reflect.Slice:
		panic("array")
	case reflect.Map:
		panic("map")
	case reflect.Ptr:
		panic("ptr")
	case reflect.String:
		return v.String()
	case reflect.Struct:
		panic("unexpected struct")
	default:
		panic("unexpected")
	}
}

func getNameFromTag(v string) string {
	parts := strings.Split(v, ",")
	for _, p := range parts {
		if strings.HasPrefix(p, "name=") {
			return p[5:]
		}
	}
	return ""
}
