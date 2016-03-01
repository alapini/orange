package orange_test

import (
	"fmt"

	"github.com/gernest/orange"

	// Include the driver for your database
	_ "github.com/lib/pq"
)

type golangster struct {
	ID   int64
	Name string
}

func Example() {

	// Open a database connection
	connectionSTring := "user=postgres dbname=orange_test sslmode=disable"
	db, err := orange.Open("postgres", connectionSTring)
	if err != nil {
		panic(err)
	}

	// Register the structs that you want to map to
	err = db.Register(&golangster{})
	if err != nil {
		panic(err)
	}

	// Do database migrations( tables will be created if they dont exist
	err = db.Automigrate()
	if err != nil {
		panic(err)
	}

	// Make sure we are connected to the database we want
	name := db.CurrentDatabase()
	fmt.Println(name) // on my case it is orange_test

	// Insert a new record into the database
	err = db.Create(&golangster{Name: "hello"})
	if err != nil {
		panic(err)
	}

	// count the number of records
	var count int
	err = db.Count("*").Bind(&count)
	if err != nil {
		panic(err)
	}
	fmt.Println(count) // in my case 1

	// Retrieve a a record with name hello
	result := golangster{}
	err = db.Find(&result, &golangster{Name: "hello"})
	if err != nil {
		panic(err)
	}
	fmt.Println(result) // on my case { 1, "hello"}

}
