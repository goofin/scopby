package models

import "github.com/zeebo/errs"

//go:generate dbx.v1 golang -p models -d sqlite3 -t templates models.dbx .

var (
	Err = errs.Class("models")
)

func init() {
	WrapErr = func(err *Error) error { return Err.Wrap(err) }
}
