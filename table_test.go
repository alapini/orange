package orange

import (
	"testing"
	"time"
)

func TestTabulizeName(t *testing.T) {
	sample := []struct {
		name, expect string
	}{
		{"gernest", "gernest"},
		{"Gernest", "gernest"},
		{"OrangeJuice", "orange_juice"},
		{"orangeJuice", "orange_juice"},
		{"OrangeJuiceIsSweet", "orange_juice_is_sweet"},
		{"HTMLOrangeJuice", "html_orange_juice"},
		{"normalPILLINGStuffs", "normal_pilling_stuffs"},
	}
	for _, v := range sample {
		n := tabulizeName(v.name)
		if n != v.expect {
			t.Errorf("expected %s got %s", v.expect, n)
		}
	}
}

type simpleModel struct {
	ID        int64 `sql:"id"`
	BOdy      string
	CreatedAt time.Time
	UpdatedAT time.Time
}

func TestLoadTable(t *testing.T) {
	tb, err := loadTable(&simpleModel{})
	if err != nil {
		t.Fatal(err)
	}
	name := "simple_model"
	if tb.Name() != name {
		t.Errorf("expected %s got %s", name, tb.Name())
	}
}
