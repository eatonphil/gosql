package gosql

import "errors"

var (
	ErrTableDoesNotExist  = errors.New("Table does not exist")
	ErrColumnDoesNotExist = errors.New("Column does not exist")
	ErrInvalidSelectItem  = errors.New("Select item is not valid")
	ErrInvalidDatatype    = errors.New("Invalid datatype")
	ErrMissingValues      = errors.New("Missing values")
	ErrInvalidCell        = errors.New("Cell is invalid")
	ErrInvalidOperands    = errors.New("Operands are invalid")
)
