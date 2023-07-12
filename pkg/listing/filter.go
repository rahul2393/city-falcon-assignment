package listing

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-pg/pg/v10/orm"
	"github.com/go-pg/pg/v10/types"
	"github.com/iancoleman/strcase"
)

// Mapper is a function that maps a value to another value.
type Mapper func(interface{}) (interface{}, error)

// FilterHook is a function that takes a pointer to a parsed filter condition as parameter and returns an error.
type FilterHook func(*Condition) error

// FilterConfig is the configuration for parsing filter expressions and generating WHERE clauses based on them.
type FilterConfig struct {
	// Hooks maps filter expression field names to hook functions.
	//
	// Hooks is consulted when generating a WHERE clause as part of processing a filter expression. If the field name participating in a comparison is found in Hooks,
	// the FilterHook is called and the filterparser.Condition being processed is passed to it. The FilterHook may edit the field name as well as modify any values in the right hand side
	// of the comparison, including adding and removing values.
	// If the FilterHook returns an error, the query is aborted and the error is propagated back.
	Hooks map[string]FilterHook
}

// MapValues calls fn for every element in values. If fn returns an error MapValues returns early.
func MapValues(values []interface{}, fn Mapper) error {
	for i, curr := range values {
		newValue, err := fn(curr)
		if err != nil {
			return fmt.Errorf("value #%d: %v", i+1, err)
		}

		values[i] = newValue
	}

	return nil
}

// RenameFieldAndMapValuesFilterHook is a convenience function that returns a FilterHook performing the following operations on the filterparser.Condition:
//
// 1. Set c.Field to fieldName.
//
// 2. Calls MapValues(c.Values, valueFunc).
func RenameFieldAndMapValuesFilterHook(fieldName string, valueFunc Mapper) FilterHook {
	return func(cond *Condition) error {
		if err := MapValues(cond.Values, valueFunc); err != nil {
			return err
		}

		cond.Field = fieldName
		return nil
	}
}

// ErrNoop error when returned from FilterHook for a field will prevent that field from being applied in the filter of ListResources
var ErrNoop = errors.New("noop")

var likeEscaper = strings.NewReplacer("\\", "\\\\", "%", "\\%", ".", "\\.")

func escapeLike(expr string) string {
	return likeEscaper.Replace(expr)
}

func ApplyFilters(expr string, config FilterConfig, query *orm.Query) error {
	filters, err := Parse(expr)
	if err != nil {
		return err
	}
	table := query.TableModel().Table()
	for i, curr := range filters.Conditions {
		not := ""
		if curr.Not {
			not = " NOT "
		}

		if hook, ok := config.Hooks[curr.Field]; ok {
			if err := hook(&curr); err != nil {
				if err == ErrNoop {
					continue
				}

				return fmt.Errorf("hook: %v", err)
			}
		}

		field := string(table.Alias) + "." // "alias".

		var params []interface{}

		// Check if dealing with a nested field
		if s := strings.Split(curr.Field, "."); len(s) > 1 {
			pgField, ok := table.FieldsMap[strcase.ToSnake(s[0])]
			if !ok {
				return fmt.Errorf("field: %q: not found in model", s[0])
			}

			if pgField.Field.Type.Kind() != reflect.Map {
				return fmt.Errorf("field: %q: not a map", s[0])
			}

			// Support only map[string]string for now
			if t := pgField.Field.Type; t.Key().Kind() != reflect.String || t.Elem().Kind() != reflect.String {
				return fmt.Errorf("field: %q: got map[%s]%s want map[string]string", s[0], t.Key().Kind(), t.Elem().Kind())
			}

			// To escape the name of a nested field it has to be passed to query.Where() in the "params" parameter. The nested field name is the first/leftmost parameter.
			params = append(params, s[1])

			field += string(pgField.Column) + "->>?" // "alias"."column"->>?
		} else {
			pgField, ok := table.FieldsMap[strcase.ToSnake(curr.Field)]
			if !ok {
				return fmt.Errorf("field: %q: not found in model", curr.Field)
			}

			field += string(pgField.Column) // "alias"."column"
		}

		switch curr.Op {
		case OpEqual, OpNotEqual, OpGreater, OpGreaterOrEqual, OpLess, OpLessOrEqual:
			op := curr.Op.String()
			if curr.Op == OpNotEqual {
				op = "<>"
			}
			params = appendToNonEmpty(params, curr.Values...)
			query.Where(not+field+" "+op+" ?", params...)

		case OpIn:
			params = appendToNonEmpty(params, types.In(curr.Values))
			query.Where(field+not+" IN (?)", params...)

		case OpRange:
			params = appendToNonEmpty(params, curr.Values...)
			query.Where(field+not+" BETWEEN ? AND ?", params...)

		case OpContains:
			params = appendToNonEmpty(params, "%"+escapeLike(curr.Values[0].(string))+"%")
			query.Where(field+not+" LIKE ?", params...)

		default:
			return fmt.Errorf("condition #%d: unknown operator", i+1)
		}
	}
	return nil
}

// appendToNonEmpty returns elems appended to slice if slice is not empty else it returns elems.
func appendToNonEmpty(slice []interface{}, elems ...interface{}) []interface{} {
	if len(slice) == 0 {
		return elems
	}
	return append(slice, elems...)
}
