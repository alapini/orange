package orange

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type postgresql struct{}

// Create returns sql query for creating table t if it does not exist
func (p *postgresql) Create(t Table) (string, error) {
	buf := &bytes.Buffer{}
	_, _ = buf.WriteString("CREATE TABLE IF NOT EXISTS " + t.Name() + " (")
	fields, err := t.Fields()
	if err != nil {
		return "", err
	}
	size := len(fields)
	for k, v := range fields {
		column, err := p.Field(v)
		if err != nil {
			return "", nil
		}
		if k == size-1 {
			_, _ = buf.WriteString(column)
			break
		}
		_, _ = buf.WriteString(column + ",")
	}
	_, _ = buf.WriteString(");")
	return buf.String(), nil
}

// Drop returns sql query for dropping table t.
func (p *postgresql) Drop(t Table) (string, error) {
	query := "DROP TABLE IF EXISTS " + t.Name()
	return query, nil
}

// Field returns sql representation of field f..
func (p *postgresql) Field(f Field) (string, error) {
	buf := &bytes.Buffer{}
	fName := f.ColumnName()
	_, _ = buf.WriteString(fName + " ")
	var details string
	switch f.Type().Kind() {
	case reflect.String:
		details = "text"
	case reflect.Bool:
		details = "boolean"
	case reflect.Int:
		if strings.ToLower(f.Name()) == "id" {
			details = "serial"
			break
		}
		details = "integer"
	case reflect.Int64:
		if strings.ToLower(f.Name()) == "id" {
			details = "bigserial"
			break
		}
		details = "bigint"
	case reflect.Struct:
		if f.Type().AssignableTo(reflect.TypeOf(time.Time{})) {
			details = "timestamp with time zone"
		}
	}
	if details == "" {
		return "", fmt.Errorf(" unknown type for field %s", f.Type().Kind())
	}
	_, _ = buf.WriteString(details)
	return buf.String(), nil
}

func (p *postgresql) Quote(pos int) string {
	return fmt.Sprintf("$%d", pos)
}

// Name returns the name of adopter.
func (p *postgresql) Name() string {
	return "postgres"
}

// Database returns the name of the curent database that the queries are running
// on.
func (p *postgresql) Database(s *SQL) string {
	query := "SELECT current_database();"
	var name string
	r := s.QueryRow(query)
	_ = r.Scan(&name)
	return name
}

func (p *postgresql) HasPrepare() bool {
	return true
}
