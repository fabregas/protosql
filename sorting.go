package protosql

//
// API Sort struct MUST be declared as following:
//   type XXXsome_name_of_Sort struct {
//      FieldName string
//      Order Order
//   }
//

import (
	"fmt"
)

type Sorting interface {
	GetFieldName() string
	GetOrder() string
}

func sortQuery(s Sorting) string {
	return fmt.Sprintf(" ORDER BY %s %s", s.GetFieldName(), s.GetOrder())
}
