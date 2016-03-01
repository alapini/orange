package orange

// Adopter is an interface for database centric sql.
type Adopter interface {

	//Create returns sql for creating the table
	Create(Table) (string, error)

	//Field is returns sql representation of the field in the database. The
	//returned string is used for the creation of the tables.
	Field(Field) (string, error)

	//Drop returns sql query for droping the table
	Drop(Table) (string, error)

	// Quote returns  guoted  string for use in the sql queries. This offers a
	// character for positional arguments.
	//
	//	for mysql ? is used e.g name=?
	//	for postgres $ is used e.g name=$1
	// The argument is the position of the parameter.
	Quote(int) string

	//HasPrepare returns true if the adopter support prepared statements
	HasPrepare() bool

	//Name returns the name of the adopter
	Name() string

	//Database returns the current Database
	Database(*SQL) string
}
