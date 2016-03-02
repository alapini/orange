package orange

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"reflect"
	"strings"
	"sync"
)

// a sql command with the argumenst
type clause struct {
	condition string
	args      []interface{}
}

//SQL provides methods for interacting with Relational databases
type SQL struct {
	models  map[string]Table
	mu      sync.RWMutex
	adopter Adopter
	loader  LoadFunc
	clause  struct {
		where, limit, offset, order, count, dbSelect *clause
	}
	db      *sql.DB
	verbose bool
	isDone  bool // true when the current query has already been executed.
}

func newSQL(dbAdopter Adopter, dbConnection string) (*SQL, error) {
	db, err := sql.Open(dbAdopter.Name(), dbConnection)
	if err != nil {
		return nil, err
	}
	return &SQL{
		models:  make(map[string]Table),
		adopter: dbAdopter,
		loader:  loadTable,
		db:      db,
	}, nil
}

//Open opens a new database connection for the given adopter
//
// There is only postgres support right now, mysql will come out soon.
//			database	| adopter name
//			----------------------------
//			postgresql	| postgres
func Open(dbAdopter, dbConnection string) (*SQL, error) {
	switch dbAdopter {
	case "postgres":
		return newSQL(&postgresql{}, dbConnection)
	}
	return nil, errors.New("unsupported  databse ")
}

//DB returns the underlying Database connection.
func (s *SQL) DB() *sql.DB {
	return s.db
}

//LoadFunc sets f as the table loading function. To get Table from models the
//function f will be used. It is up to the user to make sense out of the Table
//implementation when this method is used.
func (s *SQL) LoadFunc(f LoadFunc) *SQL {
	s.loader = f
	return s
}

//Register registers model. All models should be registered before calling any
//method from this struct. It is safe to call this method in multiple
//goroutines.
func (s *SQL) Register(models ...interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(models) > 0 {
		for _, v := range models {
			t, err := s.loader(v)
			if err != nil {
				return err
			}
			typ := reflect.TypeOf(v)
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			s.models[typ.Name()] = t
		}
	}
	return nil
}

//DropTable drops the Database table for the model.
func (s *SQL) DropTable(model interface{}) error {
	typ := reflect.TypeOf(model)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	t := s.getModel(typ.Name())
	if t != nil {
		query, err := s.adopter.Drop(t)
		if err != nil {
			return err
		}
		_, err = s.Exec(query)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("the table is not registered yet ")
}

//getModel returns the table registered by name, returns nil if the table was
//not yet registered.
func (s *SQL) getModel(name string) Table {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.models[name]
}

//Automigrate creates the database tables if they don't exist
func (s *SQL) Automigrate() error {
	for _, m := range s.models {
		query, err := s.adopter.Create(m)
		if err != nil {
			return err
		}
		_, err = s.Exec(query)
		if err != nil {
			return err
		}
	}
	return nil
}

//Copy returns a new copy of s. It is used for effective method chaining to
//avoid messing up the scope.
func (s *SQL) Copy() *SQL {
	return &SQL{
		db:      s.db,
		models:  s.models,
		adopter: s.adopter,
	}
}

//CopyQuery returns a copy of *SQL when the composed query has already been
//executed.
func (s *SQL) CopyQuery() *SQL {
	if s.isDone {
		return s.Copy()
	}
	return s
}

//Where adds a where query, value can be a query string, a model or a map
func (s *SQL) Where(value interface{}, args ...interface{}) *SQL {
	dup := s.CopyQuery()
	refVal := reflect.ValueOf(value)
	if refVal.Kind() == reflect.Ptr {
		refVal = refVal.Elem()
	}

	switch refVal.Kind() {
	case reflect.String:
		c := &clause{condition: value.(string)}
		if len(args) > 0 {
			c.args = args
		}
		dup.clause.where = c
		return dup
	case reflect.Struct:
		t, err := loadTable(value)
		if err != nil {
			//TODO handle?
			return dup
		}
		cols, vals, err := Values(t, value)
		if err != nil {
			return dup
		}
		var keyVal string
		for k, v := range cols {
			keyVal = keyVal + fmt.Sprintf(" %s=%s", v, s.quote(vals[k]))
		}
		dup.clause.where = &clause{condition: keyVal}
		return dup
	}
	return dup
}

//Values returns the fields that are present in the table t which have values
//set in model v.
// THis tries to breakdown the mapping of table collum names with their
// corresponding values.
//
// For instance if you have a model defined as
//	type foo  struct{
//		ID int
//	}
//
// After loading a table representation of foo, you can get the column names
// that have been assigned values like this
//	cols,vals,err:=Values(fooTable,&foo{ID: 1})
//	// cols will be []string{"ID"}
//	// vals will be []interface{}{1}
func Values(t Table, v interface{}) (cols []string, vals []interface{}, err error) {
	f, err := t.Fields()
	if err != nil {
		return
	}
	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	for _, field := range f {
		fv := value.FieldByName(field.Name())
		if fv.IsValid() {
			zero := reflect.Zero(fv.Type())
			if reflect.DeepEqual(zero.Interface(), fv.Interface()) {
				continue
			}
			colName := field.Name()
			tags, _ := field.Flags()
			if tags != nil {
				for _, tag := range tags {
					if tag.Name() == "field_name" {
						colName = tag.Value()
						break
					}
				}
			}
			cols = append(cols, colName)
			vals = append(vals, fv.Interface())
		}
	}
	return
}

//Limit sets up LIMIT clause, condition is the value for limit. Calling this with
//condition, will set a barrier to the actual number of rows that are returned
//after executing the query.
//
// If you set limit to 10, only the first 10 records will be returned and if the
// query result returns less than the 10 rows thn they will be used instead.
func (s *SQL) Limit(condition int) *SQL {
	dup := s.CopyQuery()
	query := fmt.Sprintf(" LIMIT %d", condition)
	dup.clause.limit = &clause{condition: query}
	return dup
}

// Count adds COUNT statement, colum is the column name that you want to
// count. It is up to the caller to provide a single value to bind to( in which
// the tal count will be written to.
//
//	var total int64
// 	db.Select(&user{}).Count"id").Bind(&total)
func (s *SQL) Count(column string) *SQL {
	dup := s.CopyQuery()
	query := fmt.Sprintf(" COUNT (%s) ", column)
	dup.clause.count = &clause{condition: query}
	return dup
}

// Offset adds OFFSET clause with the offset value set to condition.This allows
// you to pick just a part of the result of executing the whole query, all the
// rows before condition will be skipped.
//
// For instance if condition is set to 5, then the results will contain rows
// from number 6
func (s *SQL) Offset(condition int) *SQL {
	dup := s.CopyQuery()
	query := fmt.Sprintf(" LIMIT %d", condition)
	dup.clause.offset = &clause{condition: query}
	return dup
}

//Select adds SELECT clause. No query is executed by this method, only the call
//for *SQL.Bind will excute the built query( with exceptions of the wrappers for
//database/sql package)
//
// query can be a model or a string. Only when query is a string will the args
// be used.
func (s *SQL) Select(query interface{}, args ...interface{}) *SQL {
	dup := s.CopyQuery()
	val := reflect.ValueOf(query)
	switch val.Kind() {
	case reflect.String:
		c := &clause{condition: query.(string)}
		if len(args) > 0 {
			c.args = args
		}
		dup.clause.dbSelect = c
		return dup
	case reflect.Struct:
		t := s.getModel(val.Type().Name())
		if t == nil {
			//TODO return an error
			return dup
		}
		q := "* FROM " + t.Name()
		c := &clause{condition: q}
		dup.clause.dbSelect = c
		return dup
	case reflect.Ptr:
		val = val.Elem()
		if val.Kind() == reflect.Struct {
			t := s.getModel(val.Type().Name())
			if t == nil {
				//TODO return an error
				return dup
			}
			q := "* FROM " + t.Name()
			c := &clause{condition: q}
			dup.clause.dbSelect = c
			return dup
		}
	}
	return dup
}

//BuildQuery returns the sql query that will be executed
func (s *SQL) BuildQuery() (string, []interface{}, error) {
	buf := &bytes.Buffer{}
	var args []interface{}
	if s.clause.dbSelect != nil {
		_, _ = buf.WriteString("SELECT ")
		selectCond := s.clause.dbSelect.condition
		if s.clause.count != nil {
			_, _ = buf.WriteString(s.clause.count.condition)
			n := strings.Index(selectCond, "FROM")
			if n > 0 {
				selectCond = selectCond[n:]
			}
		}
		_, _ = buf.WriteString(selectCond)
		if s.clause.dbSelect.args != nil {
			args = append(args, s.clause.dbSelect.args)
		}
	}
	if s.clause.where != nil {
		_, _ = buf.WriteString(" WHERE" + s.clause.where.condition)
		if s.clause.dbSelect != nil && s.clause.dbSelect.args != nil {
			args = append(args, s.clause.where.args)
		}
	}
	if s.clause.offset != nil {
		_, _ = buf.WriteString("OFFSET " + s.clause.offset.condition)
		if s.clause.dbSelect.args != nil {
			args = append(args, s.clause.offset.args)
		}
	}
	if s.clause.limit != nil {
		_, _ = buf.WriteString("LIMIT" + s.clause.limit.condition)
		if s.clause.dbSelect.args != nil {
			args = append(args, s.clause.limit.args)
		}
	}
	_, _ = buf.WriteString(";")
	if s.verbose {
		fmt.Println(buf.String())
	}
	return buf.String(), cleanArgs(args...), nil
}

//Find executes the composed query and retunrs a single value if model is not a
//slice, or a slice of models when the model is slice.
func (s *SQL) Find(model interface{}, where ...interface{}) error {
	dup := s.CopyQuery()
	dup.Select(model)
	switch len(where) {
	case 0:
		break
	case 1:
		dup.Where(where[0])
	default:
		dup.Where(where[0], where[1:]...)
	}
	return dup.Bind(model)
}

// cleanArgs escapes all string values in args. It is a sane way to escape user
// supplied inputs.
//
// NEVER EVER TRUST INPUT FROM THE USER
func cleanArgs(args ...interface{}) (rst []interface{}) {
	if len(args) > 0 {
		for _, v := range args {
			if v != nil {
				if typ, ok := v.(string); ok {
					rst = append(rst, html.EscapeString(typ))
				}
				rst = append(rst, v)
			}

		}
		return args
	}
	return
}

type valScanner interface {
	Scan(dest ...interface{}) error
}

func (s *SQL) scanStruct(scanner valScanner, model interface{}) error {
	val := reflect.ValueOf(model)
	if val.Kind() != reflect.Ptr {
		return errors.New("can not assign to model")
	}
	t, err := s.loader(model)
	if err != nil {
		return err
	}
	var result []interface{}
	fields, err := t.Fields()
	if err != nil {
		return err
	}

	for _, v := range fields {
		result = append(result, reflect.New(v.Type()))
	}
	err = scanner.Scan(result...)
	if err != nil {
		return err
	}

	// we use the actual value now not the address
	val = val.Elem()
	for k, v := range fields {
		f := val.FieldByName(v.Name())
		fVal := reflect.ValueOf(result[k])
		f.Set(fVal.Elem())
	}
	return nil
}

//Query retriews matching rows . This wraps the sql.Query and no further
//no further processing is done.
func (s *SQL) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

//QueryRow QueryRow returnes a single matched row. This wraps sql.QueryRow no
//further processing is done.
func (s *SQL) QueryRow(query string, args ...interface{}) *sql.Row {
	return s.db.QueryRow(query, args...)
}

// Exec executes the query.
func (s *SQL) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args...)
}

//CurrentDatabase returns the name of the database in which the queries are
//executed.
func (s *SQL) CurrentDatabase() string {
	return s.adopter.Database(s)
}

//Bind executes the query and scans results into value. If there is any error it
//will be returned.
//
// values is a pointer to the golang type into which the resulting query results
// will be assigned. For structs, make sure the strucrs have been registered
// with the Register method.
//
// If you want to assign values from the resulting query you can pass them ass a
// comma separated list of argumens.
//		eg db.Bind(&col1,&col2,&col3)
//		will assign results from executing the query(only first row) to
//		col1,col2,and col3 respectively
//
// value can be a slice of struct , in which case the stuct should be a model
// which has previously been registered with Register method.
//
// When value is a slice of truct the restult of multiple rows will be assigned
// to the struct and appeded to the slice. So if the result of the query has 10
// rows, the legth of the slice will be 10 and each slice item will be a struct
// containing the row results.
//
// TODO(gernest) Add support for a slice of map[string]interface{}
func (s *SQL) Bind(value interface{}, args ...interface{}) error {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Ptr {
		return errors.New("non pointer argument")
	}
	defer func() { s.isDone = true }()
	var scanArgs []interface{}

	// We get the actual value that v points to.`:w
	actualVal := v.Elem()
	switch actualVal.Kind() {
	case reflect.Struct:
		t, err := s.loader(value)
		if err != nil {
			return err
		}
		fields, err := t.Fields()
		for _, v := range fields {
			scanArgs = append(scanArgs, reflect.New(v.Type()).Interface())
		}
		query, qArgs, err := s.BuildQuery()
		if err != nil {
			return err
		}
		var row *sql.Row
		switch len(qArgs) {
		case 0:
			row = s.QueryRow(query)
		default:
			row = s.QueryRow(query, qArgs...)
		}
		err = row.Scan(scanArgs...)
		if err != nil {
			return err
		}
		for k, v := range scanArgs {
			sField, ok := actualVal.Type().FieldByName(fields[k].Name())
			if ok {
				aField := actualVal.FieldByName(sField.Name)
				aField.Set(reflect.ValueOf(v).Elem())
			}
		}
	default:
		scanArgs = append(scanArgs, value)
		if len(args) > 0 {
			scanArgs = append(scanArgs, args...)
		}
		query, qArgs, err := s.BuildQuery()
		if err != nil {
			return err
		}
		var row *sql.Row
		switch len(qArgs) {
		case 0:
			row = s.QueryRow(query)
		default:
			row = s.QueryRow(query, qArgs...)
		}
		return row.Scan(scanArgs...)
	}
	return nil
}

//Create creates a new record into the database
func (s *SQL) Create(model interface{}) error {
	query, err := s.creare(model)
	if err != nil {
		return err
	}
	_, err = s.Exec(query)
	return err
}

func (s *SQL) creare(model interface{}) (string, error) {
	t, err := s.loader(model)
	if err != nil {
		return "", err
	}
	cols, vals, err := Values(t, model)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	_, _ = buf.WriteString("INSERT INTO " + t.Name())
	_, _ = buf.WriteString(" (")
	for k, v := range cols {
		if k == 0 {
			_, _ = buf.WriteString(v)
			continue
		}
		_, _ = buf.WriteString(", " + v)
	}
	_, _ = buf.WriteString(")")

	_, _ = buf.WriteString(" VALUES (")

	for k, v := range vals {
		if k == 0 {
			_, _ = buf.WriteString(s.quote(v))
			continue
		}
		_, _ = buf.WriteString(fmt.Sprintf(", %v", s.quote(v)))
	}
	_, _ = buf.WriteString(");")
	return buf.String(), nil
}

//Update updates a model values into the database
func (s *SQL) Update(model interface{}) error {
	query, err := s.update(model)
	if err != nil {
		return err
	}
	_, err = s.Exec(query)
	return err
}

func (s *SQL) update(model interface{}) (string, error) {
	t, err := s.loader(model)
	if err != nil {
		return "", err
	}
	cols, vals, err := Values(t, model)
	if err != nil {
		return "", err
	}
	var where string
	var up string
	for k, v := range cols {
		if strings.ToLower(v) == "id" {
			where = fmt.Sprintf(" %s=%v", v, s.quote(vals[k]))
			continue
		}
		if up == "" {
			up = fmt.Sprintf("%s =%v", v, s.quote(vals[k]))
			continue
		}
		up = up + fmt.Sprintf(",%s =%v", v, s.quote(vals[k]))
	}
	return fmt.Sprintf("UPDATE %s SET %s WHERE %s", t.Name(), up, where), nil
}

// quote add single quote to val if val is a string.
func (s *SQL) quote(val interface{}) string {
	typ := reflect.TypeOf(val)
	if typ.Kind() == reflect.String {
		return fmt.Sprintf("'%v'", val)
	}
	return fmt.Sprint(val)
}
