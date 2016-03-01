package orange

import (
	"testing"
	"time"
)

type postgresTest struct {
	ID        int
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func TestPostgres_Create(t *testing.T) {
	p := &postgresql{}
	tab, err := loadTable(&postgresTest{})
	if err != nil {
		t.Fatal(err)
	}
	create, err := p.Create(tab)
	if err != nil {
		t.Fatal(err)
	}
	expect := "CREATE TABLE IF NOT EXISTS postgres_test (ID serial,Body text,CreatedAt timestamp with time zone,UpdatedAt timestamp with time zone);"
	if create != expect {
		t.Errorf("expected %s got %s", expect, create)
	}
}

func TestPostgres_Drop(t *testing.T) {
	p := &postgresql{}
	tab, err := loadTable(&postgresTest{})
	if err != nil {
		t.Fatal(err)
	}
	drop, err := p.Drop(tab)
	if err != nil {
		t.Fatal(err)
	}
	expect := "DROP TABLE IF EXISTS postgres_test"
	if drop != expect {
		t.Errorf("expected %s got %s", expect, drop)
	}
}
