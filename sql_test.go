package orange

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

var testDB = struct {
	ps, mysal, sqlite string
}{}

func init() {
	testDB.ps = os.Getenv("POSTGRES_ORANGE")
	if testDB.ps == "" {
		//testDB.ps = "postgres:://postgres@localhost/orange_test?sslmode=disable"
		testDB.ps = "user=postgres dbname=orange_test sslmode=disable"
	}
}

func TestOpen(t *testing.T) {
	_, err := Open("postgres", testDB.ps)
	if err != nil {
		t.Fatal(err)
	}
}

type golangster struct {
	ID   int64
	Name string
}

func TestSQL_Register(t *testing.T) {
	db, err := Open("postgres", testDB.ps)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Register(&golangster{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSQL_Automigrate(t *testing.T) {
	db, err := Open("postgres", testDB.ps)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Register(&golangster{})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Automigrate()
	if err != nil {
		t.Fatal(err)
	}
	err = db.DropTable(&golangster{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSQL_CurrentDatabase(t *testing.T) {
	db, err := Open("postgres", testDB.ps)
	if err != nil {
		t.Fatal(err)
	}
	name := db.CurrentDatabase()
	dbName := "orange_test"
	if name != dbName {
		t.Errorf("expected %s got %s", dbName, name)
	}
}

func TestValues(t *testing.T) {
	sample := []struct {
		id   int64
		name string
		cols []string
		vals []interface{}
	}{
		{0, "hello", []string{"Name"}, []interface{}{"hello"}},
	}

	model, err := loadTable(&golangster{})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range sample {
		cols, vals, err := Values(model, &golangster{Name: v.name})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(v.cols, cols) {
			t.Errorf("expected %v to equal %v", cols, v.cols)
		}
		if !reflect.DeepEqual(v.vals, vals) {
			t.Errorf("expected %v to equal %v", vals, v.vals)
		}
	}
}

func TestSQL_WHere(t *testing.T) {
	db, err := Open("postgres", testDB.ps)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Register(&golangster{})

	db.Where(&golangster{Name: "hello"})
	query, _, err := db.BuildQuery()
	if err != nil {
		t.Fatal(err)
	}
	exp := "WHERE Name='hello';"
	if strings.TrimSpace(query) != exp {
		t.Errorf("expected %s got %s", exp, query)
	}
}

func TestSQL_Select(t *testing.T) {
	db, err := Open("postgres", testDB.ps)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Register(&golangster{})
	db.Select(&golangster{})
	query, _, err := db.BuildQuery()
	if err != nil {
		t.Fatal(err)
	}
	expect := "SELECT * FROM golangster;"
	if strings.TrimSpace(query) != expect {
		t.Errorf("expected %s got %s", expect, query)
	}

	// This should work for non pointers too
	clone := db.Copy()
	clone.Select(golangster{})
	query, _, err = clone.BuildQuery()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(query) != expect {
		t.Errorf("expected %s got %s", expect, query)
	}

	clone = db.Copy()
	clone.Select("* FROM golangster")
	query, _, err = clone.BuildQuery()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(query) != expect {
		t.Errorf("expected %s got %s", expect, query)
	}

	// combine select with where
	clone = db.Copy().Where(&golangster{Name: "gernest"}).Select(&golangster{})
	query, _, err = clone.BuildQuery()
	if err != nil {
		t.Fatal(err)
	}
	comibeExpect := "SELECT * FROM golangster WHERE Name='gernest';"
	if strings.TrimSpace(query) != comibeExpect {
		t.Errorf("expected %s got %s", comibeExpect, query)
	}
}

func TestSQL_LoadFunc(t *testing.T) {
	buf := &bytes.Buffer{}
	db, err := Open("postgres", testDB.ps)
	if err != nil {
		t.Fatal(err)
	}
	db.LoadFunc(func(m interface{}) (Table, error) {
		tab, err := loadTable(m)
		if err != nil {
			_, _ = buf.WriteString(err.Error())
			return nil, err
		}
		_, _ = buf.WriteString(tab.Name())
		return tab, nil
	})
	_ = db.Register(&golangster{})
	if buf.String() != "golangster" {
		t.Errorf("expect golangster got %s instead", buf)
	}

}

func TestSQL_Create(t *testing.T) {
	db, err := Open("postgres", testDB.ps)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Register(&golangster{})
	query, err := db.creare(&golangster{ID: 2, Name: "gernest"})
	if err != nil {
		t.Fatal(err)
	}
	expect := "INSERT INTO golangster (ID, Name) VALUES (2, 'gernest');"
	if query != expect {
		t.Errorf("expected %s got %s", expect, query)
	}
	_ = db.Automigrate()

	// create an actual entry
	err = db.Create(&golangster{Name: "tanzania"})
	if err != nil {
		t.Error(err)
	}

}

func TestSQL_Update(t *testing.T) {
	db, err := Open("postgres", testDB.ps)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.DropTable(&golangster{}) }()

	_ = db.Register(&golangster{})
	query, err := db.update(&golangster{ID: 2, Name: "gernest the golangster"})
	if err != nil {
		t.Fatal(err)
	}
	expect := "UPDATE golangster SET Name ='gernest the golangster' WHERE  ID=2"
	if query != expect {
		t.Errorf("expected %s got %s", expect, query)
	}

	_ = db.Automigrate()

	// create an actual entry
	err = db.Create(&golangster{Name: "tanzania"})
	if err != nil {
		t.Error(err)
	}
	err = db.Update(&golangster{ID: 1, Name: "gernest the golangster"})
	if err != nil {
		t.Error(err)
	}
}

func TestSQL_Count(t *testing.T) {
	db, err := Open("postgres", testDB.ps)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Register(&golangster{})
	db.Select(&golangster{}).Count("*")
	query, _, err := db.BuildQuery()
	if err != nil {
		t.Fatal(err)
	}
	expect := "SELECT  COUNT (*) FROM golangster;"
	if strings.TrimSpace(query) != expect {
		t.Errorf("expected %s got %s", expect, query)
	}
	_ = db.Automigrate()
	names := []string{"one", "two", "three"}
	for _, v := range names {
		err = db.Create(&golangster{Name: v})
		if err != nil {
			t.Error(err)
		}
	}
	var result int
	err = db.Select(&golangster{}).Count("*").Bind(&result)
	if err != nil {
		t.Error(err)
	}
	if result != len(names) {
		t.Errorf("expected %d got %d", len(names), result)
	}
}

func TestSQL_Find(t *testing.T) {
	db, err := Open("postgres", testDB.ps)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Register(&golangster{})
	rst := &golangster{}
	err = db.Find(rst, &golangster{ID: 1})
	if err != nil {
		t.Error(err)
	}
	if rst.ID != 1 {
		t.Errorf("expected %d got %d", 1, rst.ID)
	}

}
