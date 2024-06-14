package trails

// Enumerable is the interface implemented by types that can only be represented by enumerable, constant values.
//
// Implementing a new Enumerable or adding a new constant value ought to include updating the database with the same
// types and values.
type Enumerable interface {
	String() string
	Valid() error
}
