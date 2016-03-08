package orange

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"unicode"
)

var (
	//ErrNoField is returned when the field is not found
	ErrNoField = errors.New("no field found")

	//ErrNoFlag is returned when the flag is not found
	ErrNoFlag   = errors.New("no flag found")
	specialTags = struct {
		fieldName, fieldType, relation string
	}{
		"name", "type", "relation",
	}
)

//Table is an interface for an object that can be mapped to a database table.
//This has no one one one correspondance with the actual database table. There
//is no limitation of the implementation of this interface.
//
// Just be aware that, In case you want to use custom implementation make sure
// the loading function is set correctly to your custom table loading function.
// see *SQL.LoadFunc for more details on custom table loading functoons.
type Table interface {

	//Name returns the name of the table.
	Name() string

	//Fields returns a collection of fields of the table. They are like an abstract
	//representation of database table colums although in some case they might
	//not. This means what they are will depend on the implementation details.
	Fields() ([]Field, error)

	//Size  returns the number of fields present in the table
	Size() int

	//Flags is a collection additional information that is tied to the table. They can be
	//anything within the scope of your wild imagination.
	Flags() ([]Flag, error)
}

//Field is an interface for a table field.
type Field interface {
	Name() string
	Type() reflect.Type
	Flags() ([]Flag, error)

	//ColumnName is the name that this field is represented in the database table
	ColumnName() string
}

//Flag is an interface for tagging objects. This can hold additional information
//about fields or tables.
type Flag interface {
	Name() string
	Key() string
	Value() string
}

type table struct {
	name   string
	fields []*field
	tags   []*tag
}

//LoadFunc is an interface for loading tables from models. Models are structs
//that maps to database tables.
type LoadFunc func(model interface{}) (Table, error)

//loadTable lods the model and returns a Table objec. A model is a golang struct
//whose fields are the database column names.
func loadTable(model interface{}) (Table, error) {
	value := reflect.ValueOf(model)
	switch value.Kind() {
	case reflect.Ptr:
		value = value.Elem()
	default:
		return nil, errors.New("provide a pointer to a model struct")
	}
	if value.Kind() != reflect.Struct {
		return nil, errors.New("modelsshould be structs")
	}
	t := &table{}
	typ := value.Type()
	t.name = tabulizeName(typ.Name())
	for k := range make([]struct{}, typ.NumField()) {
		fieldTyp := typ.Field(k)
		f := &field{}
		f.name = fieldTyp.Name
		f.typ = fieldTyp.Type
		tags := fieldTyp.Tag.Get("sql")

		// do not add ignored fields
		if tags == "-" {
			continue
		}
		f.loadTags(tags)
		t.fields = append(t.fields, f)
	}
	return t, nil
}

// tabulizeName changes name to a good database name. This means
//   CamelCame will be changed to camel_case
//   MIXEDCase will be changed to mixed_case
func tabulizeName(name string) string {
	if name == "" {
		return ""
	}
	if strings.ToLower(name) == "id" {
		return "id"
	}
	isFirstLower := false
	var capIndex []int
	for i, ch := range name {
		if i == 0 {
			isFirstLower = unicode.IsLower(ch)
		}
		if unicode.IsUpper(ch) {
			capIndex = append(capIndex, i)
		}
	}
	buf := &bytes.Buffer{}
	lenCap := len(capIndex)
	if lenCap == 0 {
		return name
	}
	i := 0
	left := 0
	piling := false
END:
	for {
		switch lenCap {
		case 1:
			c := capIndex[0]
			if c == 0 {
				writeSnake(buf, name)
				break END
			}
			writeSnake(buf, name[left:c])
			writeSnake(buf, name[c:])
			break END
		default:
			if i == lenCap-1 {
				c := capIndex[i]
				if piling {
					writeSnake(buf, name[left:c])
				}
				writeSnake(buf, name[c:])
				break END
			}
			c := capIndex[i]
			if i == 0 && isFirstLower {
				writeSnake(buf, name[:c])
			}
			next := capIndex[i+1]
			i++
			if piling && next-c != 1 {
				writeSnake(buf, name[left:c])
				writeSnake(buf, name[c:next])
				left = next
				piling = false
				break
			}
			if next-c == 1 {
				if !piling {
					left = c
				}
				piling = true
				break
			}
			writeSnake(buf, name[left:next])
			left = next
		}
	}
	return buf.String()
}

//writeSnake writes n in b with a snake case
func writeSnake(b *bytes.Buffer, n string) {
	if b.Len() == 0 {
		_, _ = b.WriteString(strings.ToLower(n))
		return
	}
	_, _ = b.WriteString("_")
	_, _ = b.WriteString(strings.ToLower(n))
}

// Name returns the table name
func (t *table) Name() string {
	return t.name
}

// size returns the number of fields present in this table. Note that this does
// not include igored fieds.
func (t *table) Size() int {
	return len(t.fields)
}

// Fields returns the fields of the tablle .
func (t *table) Fields() ([]Field, error) {
	if t.fields != nil {
		var f []Field
		for _, v := range t.fields {
			f = append(f, v)
		}
		return f, nil
	}
	return nil, ErrNoField
}

func (t *table) Flags() ([]Flag, error) {
	if t.tags != nil {
		var f []Flag
		for _, v := range t.tags {
			f = append(f, v)
		}
		return f, nil
	}
	return nil, ErrNoFlag
}

//field implements the Field interface
type field struct {
	name string
	typ  reflect.Type
	tags []*tag
}

//Name returns the name of the field as the actualname defined in the struct,
//the name specified in the tags will always remain in the tags to avoid
//unnecessary name conversions.
func (f *field) Name() string {
	return f.name
}

//Type returns the field value's type
func (f *field) Type() reflect.Type {
	return f.typ
}

//Flags returns the tags held by the field
func (f *field) Flags() ([]Flag, error) {
	if f.tags != nil {
		var t []Flag
		for _, v := range f.tags {
			t = append(t, v)
		}
		return t, nil
	}
	return nil, ErrNoFlag
}

func (f *field) SetValue(v interface{}) error {
	return nil
}

func (f *field) ColumnName() string {
	for _, v := range f.tags {
		if v.name == "field_name" {
			return v.value
		}
	}
	return tabulizeName(f.name)
}

func (f *field) loadTags(sqlTags string) {
	if sqlTags == "" {
		return
	}
	chunks := strings.Split(sqlTags, ",")
	if len(chunks) > 0 {
		for _, v := range chunks {
			f.tags = append(f.tags, &tag{name: "sql", value: v})
		}
	}
}

type tag struct {
	name, key, value string
}

func (t *tag) Name() string {
	return t.name
}

func (t *tag) Key() string {
	return t.key
}

func (t *tag) Value() string {
	return t.value
}
