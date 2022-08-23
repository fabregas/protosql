package protosql

//
// API Sort struct MUST be declared as following:
//   type XXXsome_name_of_Sort struct {
//      Order fieldName1
//      Order fieldName2
//		...
//   }
//
//  where Order must have String() method that returns ASC or DESC
//

import (
	"fmt"
	"strings"
)

type Sorting struct {
	FieldName string
	Order     string
}

func Asc(field string) *Sorting {
	return &Sorting{FieldName: field, Order: "ASC"}
}

func Desc(field string) *Sorting {
	return &Sorting{FieldName: field, Order: "DESC"}
}

// returns first defined sorting from proto message
func newSorting(s interface{}) *Sorting {
	if s == nil {
		return nil
	}

	if ss, ok := s.(*Sorting); ok {
		return ss
	}

	protoSorting, ok := s.(Model)
	if !ok {
		panic("unsupported sorting type")
	}

	parsed := parseProtoMsg(protoSorting)

	for _, f := range parsed {
		s, ok := f.val.Interface().(fmt.Stringer)
		if !ok {
			panic(fmt.Sprintf("sorting field %s is not stringer", f.name))
		}

		order := s.String()
		switch strings.ToUpper(order) {
		case "ASC", "DESC":
		default:
			// unknown sort order, just skip it
			continue
		}

		return &Sorting{FieldName: f.name, Order: order}
	}

	return nil
}

func sortQuery(s *Sorting) string {
	if s == nil {
		return ""
	}

	return fmt.Sprintf(" ORDER BY %s %s", s.FieldName, s.Order)
}
