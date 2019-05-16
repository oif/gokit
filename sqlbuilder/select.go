package sqlbuilder

import (
	"reflect"
	"strings"
)

func Select(object interface{}, ignoreFields ...string) *SQL {
	sql := new(SQL)
	sql.Kind = KindSelect
	typeOfObject := reflect.TypeOf(object)
	if typeOfObject == nil || typeOfObject.Kind() == reflect.Ptr {
		sql.err = ErrInvalidObject
		return sql
	}
	sql.PayloadType = typeOfObject
	quickSearch := make(map[string]bool)
	for _, k := range ignoreFields {
		quickSearch[k] = true
	}
	for i := 0; i < typeOfObject.NumField(); i++ {
		field := typeOfObject.Field(i)
		tag := field.Tag.Get("sb")
		if _, ignore := quickSearch[tag]; ignore {
			continue
		}
		if tag == "" {
			continue
		}
		sql.Fields = append(sql.Fields, field)
		sql.FieldKeys = append(sql.FieldKeys, tag)
	}
	return sql
}

func (s *SQL) buildSelectString() (*string, error) {
	if err := s.errorCast(); err != nil {
		return nil, err
	}
	if s.Table == "" {
		return nil, ErrMissingTable
	}
	if len(s.Fields) == 0 {
		return nil, ErrMissingFields
	}
	selectParts := []string{
		"SELECT",
		strings.Join(s.FieldKeys, ", "),
		"FROM",
		s.Table,
	}
	if s.Conditions != "" {
		selectParts = append(selectParts, "WHERE", s.Conditions)
	}
	selectString := strings.Join(selectParts, space)
	return &selectString, nil
}
