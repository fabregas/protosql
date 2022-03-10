package protosql

import (
	"fmt"
	"reflect"
	"strings"
)

type Sort struct {
	FieldName string
	Order     string
}

type Sortings []Sort

func (s Sortings) SortQuery() string {
	var query string
	if len(s) > 0 {
		var sorts []string
		for _, si := range s {
			sorts = append(
				sorts,
				fmt.Sprintf("%s %s", si.FieldName, si.Order),
			)
		}

		query += fmt.Sprintf(" ORDER BY %s", strings.Join(sorts, ", "))
	}

	return query
}

// API Sort struct MUST be declared as following:
//   type XXXsome_name_of_Sort struct {
//      FieldName string
//      Order Order
//   }
//
//  where Order has method `String() string`
//
//
// API sortings MUST be declared as following:
//   type XXXfome_Filter {
//      ...
//      Sortings []*XXXsome_name_of_Sort
//}

type ProtoSortings interface{}

func NewSortings(protoSortings ProtoSortings) Sortings {
	if protoSortings == nil {
		return nil
	}

	v := reflect.ValueOf(protoSortings)
	if v.Kind() != reflect.Slice {
		panic("expected list of Sorts")
	}

	if v.Len() == 0 {
		return nil
	}

	ret := make([]Sort, v.Len())
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Elem()
		ret[i] = Sort{
			FieldName: item.FieldByName("FieldName").String(),
			Order:     item.FieldByName("Order").Interface().(fmt.Stringer).String(),
		}
	}

	return Sortings(ret)
}
