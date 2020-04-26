package gosql

import "errors"

var (
	// ErrTableDoesNotExist is returned when the requested table does not exist
	ErrTableDoesNotExist = errors.New("Table does not exist")
	// ErrColumnDoesNotExist is returned when the requested column does not exist
	ErrColumnDoesNotExist = errors.New("Column does not exist")
	// ErrInvalidSelectItem is returned when the selected item is not present in the table
	ErrInvalidSelectItem = errors.New("Select item is not valid")
	// ErrInvalidDatatype is returned when the datatype is invalid
	ErrInvalidDatatype = errors.New("Invalid datatype")
	// ErrMissingValues is returned when the number of columns is not equal to the number of values provided in an
	// Insert Statement
	ErrMissingValues = errors.New("Missing values")
	// ErrInvalidCell is returned when expression type used to evaluate the cells are invalid
	ErrInvalidCell = errors.New("Cell is invalid")
	// ErrInvalidOperands is returned when, while comparison, the LHS and RHS are not of the same type
	ErrInvalidOperands = errors.New("Operands are invalid")
)
