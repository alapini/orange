# Orange [![GoDoc](https://godoc.org/github.com/gernest/orange?status.svg)](https://godoc.org/github.com/gernest/orange)[![Coverage Status](https://coveralls.io/repos/github/gernest/orange/badge.svg?branch=master)](https://coveralls.io/github/gernest/orange?branch=master)
[![Build Status](https://travis-ci.org/gernest/orange.svg?branch=master)](https://travis-ci.org/gernest/orange)

Orange is a lightweight, simple Object relational Mapper for Golang. Built  for
the curious minds which wants to grok how to remove the toil when building
database facing applications.

# Features
* Simple API 
* Multiple database support( currently only postgresql but mysql and sqlite are
work in progress)
* Zero dependency( only the standard libary)

# Motivation
This is my understanding of Object Relational Mapping with Golang. Instead of
writing a blog post, I took the liberty to implement `orange`. It has almost all
the things you might need to interact with databases with Golang.

The source code is geared toward people who want to harness the power of Go.
There is alot of myths around reflections in Go, I have almost used all the
techniques you will need to master reflections.

THIS IS NOT FOR PRODUCTION USE, unless you know what you are doing in which case
your contribution is welcome.


# Why another ORM?


* More options: There are already good ORM's for Go, so adding another option is
a good thing to the community. Some people might find orange suits their needs.


# Why Orange?

* For the knowledge: If you are curious and would like to know the kicks of the
dynamics behind Object Relation Mapping in Go then orange is for you. The code is
DRY, clean, well tested and easy to grok. I know It will grow over time but I
hope this premise will still hold.

* Small overhead: Orange is a honest bookkeeper, there is no need any new
paradigm, new conventions or anything. Just you and your Golang objects.

* Hackable: You can use orange as the base for your bookkeeping shelf, change
 the code to suit your needs without breaking anything.

# Do you need to use Orange?

Probably not, it was meant for educational use.Mainly to have a projects where 
people who are new to Go, or hobbyists looking for a project that they can contribute
to, experiment on and maybe build their next big thing on it.

If you ever want to harness the power of your database, take your time to learn more
about SQL. My time while developing Orange made me realize how powerful SQL is and it
was an eye opener to see the beauty of postgres. Now I know SQL, the golang's 
standard `database/sql` package is all what I need. To remove boilerplate code
I suggest you understand the problem you want to solve and use common sense. 

# Installation

```bash
go get github.com/gernest/orange
```

# Warning
Orange is not matured yet, a lot of features are missing or might be incomplete
, but you can help out by
suggesting features and maybe submitting PR too( I will be happy to  see people
eat more orange for their healthy)

# Usage

The following is a simple example to showcase the power of orange, for
comprehensive API please check [ The Orange Documentation](https://godoc.org/github.com/gernest/orange)

```go
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
	err := db.Register(&golangster{})
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
	db.Count("*").Bind(&count)
	fmt.Println(count) // in my case 1

	// Retrieve a a record with name hello
	result := golangster{}
	db.Find(&result, &golangster{Name: "hello"})
	fmt.Println(result) // on my case { 1, "hello"}

}
```

# TODO list
These  are some of the  things I will hope to add when I get time
* Delete record
* Support mysql
* support sqlite
* polish support for timestamps
* more comprehensive tests
* improve perfomace
* talk about orange


# Contributing

Contributions of all kinds are welcome

# Author
Geofrey  Ernest 
[twitter @gernesti](https://twitter.com/gernesti)

# Licence
MIT see [LICENCE](LICENCE)

