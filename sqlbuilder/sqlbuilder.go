package sqlbuilder

import (
	"database/sql"
	"reflect"
)

const (
	space = " "
)

type Field struct {
	Tag  string
	Name string
}

type SQL struct {
	Kind        Kind
	err         error
	PayloadType reflect.Type
	FieldKeys   []string
	Fields      []reflect.StructField
	Table       string
	Conditions  string
}

type Kind string

const (
	KindSelect Kind = "select"
	KindUpdate Kind = "update"
	KindInsert Kind = "insert"
	KindDelete Kind = "delete"
)

func (s *SQL) errorCast() error {
	if s == nil {
		return nil
	}
	if s.err != nil {
		return s.err
	}
	return nil
}

func (s *SQL) setTable(table string) {
	s.Table = table
}

func (s *SQL) From(table string) *SQL {
	if err := s.errorCast(); err != nil {
		return s
	}
	s.setTable(table)
	return s
}

func (s *SQL) Build() (*string, error) {
	if err := s.errorCast(); err != nil {
		return nil, err
	}
	switch s.Kind {
	case KindSelect:
		return s.buildSelectString()
	}
	return nil, ErrUnsupportedKind
}

type Scanable interface {
	Scan(dst ...interface{}) error
}

func (s *SQL) ScanRow(scanner Scanable) (interface{}, error) {
	var dsts []interface{}
	payload := reflect.New(s.PayloadType).Elem()
	for _, field := range s.Fields {
		dsts = append(dsts, payload.FieldByName(field.Name).Addr().Interface())
	}
	err := scanner.Scan(dsts...)
	if err != nil {
		return nil, err
	}
	return payload.Interface(), nil
}

func (s *SQL) ScanRows(rows *sql.Rows) ([]interface{}, error) {
	var objects []interface{}
	for rows.Next() {
		object, err := s.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}
	return objects, nil
}
