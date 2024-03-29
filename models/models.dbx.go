// AUTOGENERATED BY gopkg.in/spacemonkeygo/dbx.v1
// DO NOT EDIT.

package models

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/goofin/scopby/logger"
	"go.uber.org/zap"

	"github.com/mattn/go-sqlite3"
	"math/rand"
)

// Prevent conditional imports from causing build failures
var _ = strconv.Itoa
var _ = strings.LastIndex
var _ = fmt.Sprint
var _ sync.Mutex

var (
	WrapErr = func(err *Error) error { return err }

	errTooManyRows       = errors.New("too many rows")
	errUnsupportedDriver = errors.New("unsupported driver")
	errEmptyUpdate       = errors.New("empty update")
)

func logError(format string, args ...interface{}) {
	logger.Logger.Error("database",
		zap.String("message", fmt.Sprintf(format, args...)))
}

type ErrorCode int

const (
	ErrorCode_Unknown ErrorCode = iota
	ErrorCode_UnsupportedDriver
	ErrorCode_NoRows
	ErrorCode_TxDone
	ErrorCode_TooManyRows
	ErrorCode_ConstraintViolation
	ErrorCode_EmptyUpdate
)

type Error struct {
	Err         error
	Code        ErrorCode
	Driver      string
	Constraint  string
	QuerySuffix string
}

func (e *Error) Error() string {
	return e.Err.Error()
}

func wrapErr(e *Error) error {
	if WrapErr == nil {
		return e
	}
	return WrapErr(e)
}

func makeErr(err error) error {
	if err == nil {
		return nil
	}
	e := &Error{Err: err}
	switch err {
	case sql.ErrNoRows:
		e.Code = ErrorCode_NoRows
	case sql.ErrTxDone:
		e.Code = ErrorCode_TxDone
	}
	return wrapErr(e)
}

func unsupportedDriver(driver string) error {
	return wrapErr(&Error{
		Err:    errUnsupportedDriver,
		Code:   ErrorCode_UnsupportedDriver,
		Driver: driver,
	})
}

func emptyUpdate() error {
	return wrapErr(&Error{
		Err:  errEmptyUpdate,
		Code: ErrorCode_EmptyUpdate,
	})
}

func tooManyRows(query_suffix string) error {
	return wrapErr(&Error{
		Err:         errTooManyRows,
		Code:        ErrorCode_TooManyRows,
		QuerySuffix: query_suffix,
	})
}

func constraintViolation(err error, constraint string) error {
	return wrapErr(&Error{
		Err:        err,
		Code:       ErrorCode_ConstraintViolation,
		Constraint: constraint,
	})
}

type driver interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

var (
	notAPointer     = errors.New("destination not a pointer")
	lossyConversion = errors.New("lossy conversion")
)

type DB struct {
	*sql.DB
	dbMethods

	Hooks struct {
		Now func() time.Time
	}
}

func Open(driver, source string) (db *DB, err error) {
	var sql_db *sql.DB
	switch driver {
	case "sqlite3":
		sql_db, err = opensqlite3(source)
	default:
		return nil, unsupportedDriver(driver)
	}
	if err != nil {
		return nil, makeErr(err)
	}
	defer func(sql_db *sql.DB) {
		if err != nil {
			sql_db.Close()
		}
	}(sql_db)

	if err := sql_db.Ping(); err != nil {
		return nil, makeErr(err)
	}

	db = &DB{
		DB: sql_db,
	}
	db.Hooks.Now = time.Now

	switch driver {
	case "sqlite3":
		db.dbMethods = newsqlite3(db)
	default:
		return nil, unsupportedDriver(driver)
	}

	return db, nil
}

func (obj *DB) Close() (err error) {
	return obj.makeErr(obj.DB.Close())
}

func (obj *DB) Open(ctx context.Context) (*Tx, error) {
	tx, err := obj.DB.Begin()
	if err != nil {
		return nil, obj.makeErr(err)
	}

	return &Tx{
		Tx:        tx,
		txMethods: obj.wrapTx(tx),
	}, nil
}

func (obj *DB) NewRx() *Rx {
	return &Rx{db: obj}
}

func DeleteAll(ctx context.Context, db *DB) (int64, error) {
	tx, err := db.Open(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err == nil {
			err = db.makeErr(tx.Commit())
			return
		}

		if err_rollback := tx.Rollback(); err_rollback != nil {
			logError("delete-all: rollback failed: %v", db.makeErr(err_rollback))
		}
	}()
	return tx.deleteAll(ctx)
}

type Tx struct {
	Tx *sql.Tx
	txMethods
}

type dialectTx struct {
	tx *sql.Tx
}

func (tx *dialectTx) Commit() (err error) {
	return makeErr(tx.tx.Commit())
}

func (tx *dialectTx) Rollback() (err error) {
	return makeErr(tx.tx.Rollback())
}

type sqlite3Impl struct {
	db      *DB
	dialect __sqlbundle_sqlite3
	driver  driver
}

func (obj *sqlite3Impl) Rebind(s string) string {
	return obj.dialect.Rebind(s)
}

func (obj *sqlite3Impl) logStmt(stmt string, args ...interface{}) {
	sqlite3LogStmt(stmt, args...)
}

func (obj *sqlite3Impl) makeErr(err error) error {
	constraint, ok := obj.isConstraintError(err)
	if ok {
		return constraintViolation(err, constraint)
	}
	return makeErr(err)
}

type sqlite3DB struct {
	db *DB
	*sqlite3Impl
}

func newsqlite3(db *DB) *sqlite3DB {
	return &sqlite3DB{
		db: db,
		sqlite3Impl: &sqlite3Impl{
			db:     db,
			driver: db.DB,
		},
	}
}

func (obj *sqlite3DB) Schema() string {
	return `CREATE TABLE users (
	name TEXT NOT NULL,
	timezone INTEGER NOT NULL,
	snacks INTEGER NOT NULL,
	PRIMARY KEY ( name )
);
CREATE TABLE missions (
	id INTEGER NOT NULL,
	user TEXT NOT NULL REFERENCES users( name ) ON DELETE CASCADE,
	description TEXT NOT NULL,
	seconds INTEGER NOT NULL,
	snacks INTEGER NOT NULL,
	last_complete TIMESTAMP,
	PRIMARY KEY ( id )
);`
}

func (obj *sqlite3DB) wrapTx(tx *sql.Tx) txMethods {
	return &sqlite3Tx{
		dialectTx: dialectTx{tx: tx},
		sqlite3Impl: &sqlite3Impl{
			db:     obj.db,
			driver: tx,
		},
	}
}

type sqlite3Tx struct {
	dialectTx
	*sqlite3Impl
}

func sqlite3LogStmt(stmt string, args ...interface{}) {
	fields := []zap.Field{zap.String("statement", stmt)}
	for n, arg := range args {
		fields = append(fields,
			zap.Stringer(fmt.Sprintf("arg%d", n), pretty{arg}))
	}
	logger.Logger.Info("query", fields...)
}

type pretty struct{ v interface{} }

func (p pretty) String() string {
	v := p.v
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return "NULL"
		}
		v = rv.Elem().Interface()
	}
	switch v := v.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case time.Time:
		return fmt.Sprintf("%s", v.Format(time.RFC3339Nano))
	case []byte:
		for _, b := range v {
			if !unicode.IsPrint(rune(b)) {
				return fmt.Sprintf("%#x", v)
			}
		}
		return fmt.Sprintf("%q", v)
	}
	return fmt.Sprintf("%v", v)
}

type User struct {
	Name     string
	Timezone int
	Snacks   int
}

func (User) _Table() string { return "users" }

type User_Update_Fields struct {
	Snacks User_Snacks_Field
}

type User_Name_Field struct {
	_set   bool
	_null  bool
	_value string
}

func User_Name(v string) User_Name_Field {
	return User_Name_Field{_set: true, _value: v}
}

func (f User_Name_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (User_Name_Field) _Column() string { return "name" }

type User_Timezone_Field struct {
	_set   bool
	_null  bool
	_value int
}

func User_Timezone(v int) User_Timezone_Field {
	return User_Timezone_Field{_set: true, _value: v}
}

func (f User_Timezone_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (User_Timezone_Field) _Column() string { return "timezone" }

type User_Snacks_Field struct {
	_set   bool
	_null  bool
	_value int
}

func User_Snacks(v int) User_Snacks_Field {
	return User_Snacks_Field{_set: true, _value: v}
}

func (f User_Snacks_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (User_Snacks_Field) _Column() string { return "snacks" }

type Mission struct {
	Id           int64
	User         string
	Description  string
	Seconds      int
	Snacks       int
	LastComplete *time.Time
}

func (Mission) _Table() string { return "missions" }

type Mission_Create_Fields struct {
	LastComplete Mission_LastComplete_Field
}

type Mission_Update_Fields struct {
	LastComplete Mission_LastComplete_Field
}

type Mission_Id_Field struct {
	_set   bool
	_null  bool
	_value int64
}

func Mission_Id(v int64) Mission_Id_Field {
	return Mission_Id_Field{_set: true, _value: v}
}

func (f Mission_Id_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Mission_Id_Field) _Column() string { return "id" }

type Mission_User_Field struct {
	_set   bool
	_null  bool
	_value string
}

func Mission_User(v string) Mission_User_Field {
	return Mission_User_Field{_set: true, _value: v}
}

func (f Mission_User_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Mission_User_Field) _Column() string { return "user" }

type Mission_Description_Field struct {
	_set   bool
	_null  bool
	_value string
}

func Mission_Description(v string) Mission_Description_Field {
	return Mission_Description_Field{_set: true, _value: v}
}

func (f Mission_Description_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Mission_Description_Field) _Column() string { return "description" }

type Mission_Seconds_Field struct {
	_set   bool
	_null  bool
	_value int
}

func Mission_Seconds(v int) Mission_Seconds_Field {
	return Mission_Seconds_Field{_set: true, _value: v}
}

func (f Mission_Seconds_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Mission_Seconds_Field) _Column() string { return "seconds" }

type Mission_Snacks_Field struct {
	_set   bool
	_null  bool
	_value int
}

func Mission_Snacks(v int) Mission_Snacks_Field {
	return Mission_Snacks_Field{_set: true, _value: v}
}

func (f Mission_Snacks_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Mission_Snacks_Field) _Column() string { return "snacks" }

type Mission_LastComplete_Field struct {
	_set   bool
	_null  bool
	_value *time.Time
}

func Mission_LastComplete(v time.Time) Mission_LastComplete_Field {
	return Mission_LastComplete_Field{_set: true, _value: &v}
}

func Mission_LastComplete_Raw(v *time.Time) Mission_LastComplete_Field {
	if v == nil {
		return Mission_LastComplete_Null()
	}
	return Mission_LastComplete(*v)
}

func Mission_LastComplete_Null() Mission_LastComplete_Field {
	return Mission_LastComplete_Field{_set: true, _null: true}
}

func (f Mission_LastComplete_Field) isnull() bool { return !f._set || f._null || f._value == nil }

func (f Mission_LastComplete_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Mission_LastComplete_Field) _Column() string { return "last_complete" }

func toUTC(t time.Time) time.Time {
	return t.UTC()
}

func toDate(t time.Time) time.Time {
	// keep up the minute portion so that translations between timezones will
	// continue to reflect properly.
	return t.Truncate(time.Minute)
}

//
// runtime support for building sql statements
//

type __sqlbundle_SQL interface {
	Render() string

	private()
}

type __sqlbundle_Dialect interface {
	Rebind(sql string) string
}

type __sqlbundle_RenderOp int

const (
	__sqlbundle_NoFlatten __sqlbundle_RenderOp = iota
	__sqlbundle_NoTerminate
)

func __sqlbundle_Render(dialect __sqlbundle_Dialect, sql __sqlbundle_SQL, ops ...__sqlbundle_RenderOp) string {
	out := sql.Render()

	flatten := true
	terminate := true
	for _, op := range ops {
		switch op {
		case __sqlbundle_NoFlatten:
			flatten = false
		case __sqlbundle_NoTerminate:
			terminate = false
		}
	}

	if flatten {
		out = __sqlbundle_flattenSQL(out)
	}
	if terminate {
		out += ";"
	}

	return dialect.Rebind(out)
}

func __sqlbundle_flattenSQL(x string) string {
	// trim whitespace from beginning and end
	s, e := 0, len(x)-1
	for s < len(x) && (x[s] == ' ' || x[s] == '\t' || x[s] == '\n') {
		s++
	}
	for s <= e && (x[e] == ' ' || x[e] == '\t' || x[e] == '\n') {
		e--
	}
	if s > e {
		return ""
	}
	x = x[s : e+1]

	// check for whitespace that needs fixing
	wasSpace := false
	for i := 0; i < len(x); i++ {
		r := x[i]
		justSpace := r == ' '
		if (wasSpace && justSpace) || r == '\t' || r == '\n' {
			// whitespace detected, start writing a new string
			var result strings.Builder
			result.Grow(len(x))
			if wasSpace {
				result.WriteString(x[:i-1])
			} else {
				result.WriteString(x[:i])
			}
			for p := i; p < len(x); p++ {
				for p < len(x) && (x[p] == ' ' || x[p] == '\t' || x[p] == '\n') {
					p++
				}
				result.WriteByte(' ')

				start := p
				for p < len(x) && !(x[p] == ' ' || x[p] == '\t' || x[p] == '\n') {
					p++
				}
				result.WriteString(x[start:p])
			}

			return result.String()
		}
		wasSpace = justSpace
	}

	// no problematic whitespace found
	return x
}

// this type is specially named to match up with the name returned by the
// dialect impl in the sql package.
type __sqlbundle_postgres struct{}

func (p __sqlbundle_postgres) Rebind(sql string) string {
	out := make([]byte, 0, len(sql)+10)

	j := 1
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if ch != '?' {
			out = append(out, ch)
			continue
		}

		out = append(out, '$')
		out = append(out, strconv.Itoa(j)...)
		j++
	}

	return string(out)
}

// this type is specially named to match up with the name returned by the
// dialect impl in the sql package.
type __sqlbundle_sqlite3 struct{}

func (s __sqlbundle_sqlite3) Rebind(sql string) string {
	return sql
}

type __sqlbundle_Literal string

func (__sqlbundle_Literal) private() {}

func (l __sqlbundle_Literal) Render() string { return string(l) }

type __sqlbundle_Literals struct {
	Join string
	SQLs []__sqlbundle_SQL
}

func (__sqlbundle_Literals) private() {}

func (l __sqlbundle_Literals) Render() string {
	var out bytes.Buffer

	first := true
	for _, sql := range l.SQLs {
		if sql == nil {
			continue
		}
		if !first {
			out.WriteString(l.Join)
		}
		first = false
		out.WriteString(sql.Render())
	}

	return out.String()
}

type __sqlbundle_Condition struct {
	// set at compile/embed time
	Name  string
	Left  string
	Equal bool
	Right string

	// set at runtime
	Null bool
}

func (*__sqlbundle_Condition) private() {}

func (c *__sqlbundle_Condition) Render() string {
	// TODO(jeff): maybe check if we can use placeholders instead of the
	// literal null: this would make the templates easier.

	switch {
	case c.Equal && c.Null:
		return c.Left + " is null"
	case c.Equal && !c.Null:
		return c.Left + " = " + c.Right
	case !c.Equal && c.Null:
		return c.Left + " is not null"
	case !c.Equal && !c.Null:
		return c.Left + " != " + c.Right
	default:
		panic("unhandled case")
	}
}

type __sqlbundle_Hole struct {
	// set at compiile/embed time
	Name string

	// set at runtime
	SQL __sqlbundle_SQL
}

func (*__sqlbundle_Hole) private() {}

func (h *__sqlbundle_Hole) Render() string { return h.SQL.Render() }

//
// end runtime support for building sql statements
//
func (obj *sqlite3Impl) CreateNoReturn_User(ctx context.Context,
	user_name User_Name_Field,
	user_timezone User_Timezone_Field) (
	err error) {
	__name_val := user_name.value()
	__timezone_val := user_timezone.value()
	__snacks_val := int(0)

	var __embed_stmt = __sqlbundle_Literal("INSERT INTO users ( name, timezone, snacks ) VALUES ( ?, ?, ? )")

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __name_val, __timezone_val, __snacks_val)

	_, err = obj.driver.Exec(__stmt, __name_val, __timezone_val, __snacks_val)
	if err != nil {
		return obj.makeErr(err)
	}
	return nil

}

func (obj *sqlite3Impl) CreateNoReturn_Mission(ctx context.Context,
	mission_user Mission_User_Field,
	mission_description Mission_Description_Field,
	mission_seconds Mission_Seconds_Field,
	mission_snacks Mission_Snacks_Field,
	optional Mission_Create_Fields) (
	err error) {
	__user_val := mission_user.value()
	__description_val := mission_description.value()
	__seconds_val := mission_seconds.value()
	__snacks_val := mission_snacks.value()
	__last_complete_val := optional.LastComplete.value()

	var __embed_stmt = __sqlbundle_Literal("INSERT INTO missions ( user, description, seconds, snacks, last_complete ) VALUES ( ?, ?, ?, ?, ? )")

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __user_val, __description_val, __seconds_val, __snacks_val, __last_complete_val)

	_, err = obj.driver.Exec(__stmt, __user_val, __description_val, __seconds_val, __snacks_val, __last_complete_val)
	if err != nil {
		return obj.makeErr(err)
	}
	return nil

}

func (obj *sqlite3Impl) Get_User_By_Name(ctx context.Context,
	user_name User_Name_Field) (
	user *User, err error) {

	var __embed_stmt = __sqlbundle_Literal("SELECT users.name, users.timezone, users.snacks FROM users WHERE users.name = ?")

	var __values []interface{}
	__values = append(__values, user_name.value())

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	user = &User{}
	err = obj.driver.QueryRow(__stmt, __values...).Scan(&user.Name, &user.Timezone, &user.Snacks)
	if err != nil {
		return nil, obj.makeErr(err)
	}
	return user, nil

}

func (obj *sqlite3Impl) All_Mission_By_User(ctx context.Context,
	mission_user Mission_User_Field) (
	rows []*Mission, err error) {

	var __embed_stmt = __sqlbundle_Literal("SELECT missions.id, missions.user, missions.description, missions.seconds, missions.snacks, missions.last_complete FROM missions WHERE missions.user = ?")

	var __values []interface{}
	__values = append(__values, mission_user.value())

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	__rows, err := obj.driver.Query(__stmt, __values...)
	if err != nil {
		return nil, obj.makeErr(err)
	}
	defer __rows.Close()

	for __rows.Next() {
		mission := &Mission{}
		err = __rows.Scan(&mission.Id, &mission.User, &mission.Description, &mission.Seconds, &mission.Snacks, &mission.LastComplete)
		if err != nil {
			return nil, obj.makeErr(err)
		}
		rows = append(rows, mission)
	}
	if err := __rows.Err(); err != nil {
		return nil, obj.makeErr(err)
	}
	return rows, nil

}

func (obj *sqlite3Impl) Get_Mission_By_Id(ctx context.Context,
	mission_id Mission_Id_Field) (
	mission *Mission, err error) {

	var __embed_stmt = __sqlbundle_Literal("SELECT missions.id, missions.user, missions.description, missions.seconds, missions.snacks, missions.last_complete FROM missions WHERE missions.id = ?")

	var __values []interface{}
	__values = append(__values, mission_id.value())

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	mission = &Mission{}
	err = obj.driver.QueryRow(__stmt, __values...).Scan(&mission.Id, &mission.User, &mission.Description, &mission.Seconds, &mission.Snacks, &mission.LastComplete)
	if err != nil {
		return nil, obj.makeErr(err)
	}
	return mission, nil

}

func (obj *sqlite3Impl) UpdateNoReturn_User_By_Name(ctx context.Context,
	user_name User_Name_Field,
	update User_Update_Fields) (
	err error) {
	var __sets = &__sqlbundle_Hole{}

	var __embed_stmt = __sqlbundle_Literals{Join: "", SQLs: []__sqlbundle_SQL{__sqlbundle_Literal("UPDATE users SET "), __sets, __sqlbundle_Literal(" WHERE users.name = ?")}}

	__sets_sql := __sqlbundle_Literals{Join: ", "}
	var __values []interface{}
	var __args []interface{}

	if update.Snacks._set {
		__values = append(__values, update.Snacks.value())
		__sets_sql.SQLs = append(__sets_sql.SQLs, __sqlbundle_Literal("snacks = ?"))
	}

	if len(__sets_sql.SQLs) == 0 {
		return emptyUpdate()
	}

	__args = append(__args, user_name.value())

	__values = append(__values, __args...)
	__sets.SQL = __sets_sql

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	_, err = obj.driver.Exec(__stmt, __values...)
	if err != nil {
		return obj.makeErr(err)
	}
	return nil
}

func (obj *sqlite3Impl) UpdateNoReturn_Mission_By_Id(ctx context.Context,
	mission_id Mission_Id_Field,
	update Mission_Update_Fields) (
	err error) {
	var __sets = &__sqlbundle_Hole{}

	var __embed_stmt = __sqlbundle_Literals{Join: "", SQLs: []__sqlbundle_SQL{__sqlbundle_Literal("UPDATE missions SET "), __sets, __sqlbundle_Literal(" WHERE missions.id = ?")}}

	__sets_sql := __sqlbundle_Literals{Join: ", "}
	var __values []interface{}
	var __args []interface{}

	if update.LastComplete._set {
		__values = append(__values, update.LastComplete.value())
		__sets_sql.SQLs = append(__sets_sql.SQLs, __sqlbundle_Literal("last_complete = ?"))
	}

	if len(__sets_sql.SQLs) == 0 {
		return emptyUpdate()
	}

	__args = append(__args, mission_id.value())

	__values = append(__values, __args...)
	__sets.SQL = __sets_sql

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	_, err = obj.driver.Exec(__stmt, __values...)
	if err != nil {
		return obj.makeErr(err)
	}
	return nil
}

func (obj *sqlite3Impl) Delete_Mission_By_Id(ctx context.Context,
	mission_id Mission_Id_Field) (
	deleted bool, err error) {

	var __embed_stmt = __sqlbundle_Literal("DELETE FROM missions WHERE missions.id = ?")

	var __values []interface{}
	__values = append(__values, mission_id.value())

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	__res, err := obj.driver.Exec(__stmt, __values...)
	if err != nil {
		return false, obj.makeErr(err)
	}

	__count, err := __res.RowsAffected()
	if err != nil {
		return false, obj.makeErr(err)
	}

	return __count > 0, nil

}

func (obj *sqlite3Impl) getLastUser(ctx context.Context,
	pk int64) (
	user *User, err error) {

	var __embed_stmt = __sqlbundle_Literal("SELECT users.name, users.timezone, users.snacks FROM users WHERE _rowid_ = ?")

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, pk)

	user = &User{}
	err = obj.driver.QueryRow(__stmt, pk).Scan(&user.Name, &user.Timezone, &user.Snacks)
	if err != nil {
		return nil, obj.makeErr(err)
	}
	return user, nil

}

func (obj *sqlite3Impl) getLastMission(ctx context.Context,
	pk int64) (
	mission *Mission, err error) {

	var __embed_stmt = __sqlbundle_Literal("SELECT missions.id, missions.user, missions.description, missions.seconds, missions.snacks, missions.last_complete FROM missions WHERE _rowid_ = ?")

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, pk)

	mission = &Mission{}
	err = obj.driver.QueryRow(__stmt, pk).Scan(&mission.Id, &mission.User, &mission.Description, &mission.Seconds, &mission.Snacks, &mission.LastComplete)
	if err != nil {
		return nil, obj.makeErr(err)
	}
	return mission, nil

}

func (impl sqlite3Impl) isConstraintError(err error) (
	constraint string, ok bool) {
	if e, ok := err.(sqlite3.Error); ok {
		if e.Code == sqlite3.ErrConstraint {
			msg := err.Error()
			colon := strings.LastIndex(msg, ":")
			if colon != -1 {
				return strings.TrimSpace(msg[colon:]), true
			}
			return "", true
		}
	}
	return "", false
}

func (obj *sqlite3Impl) deleteAll(ctx context.Context) (count int64, err error) {
	var __res sql.Result
	var __count int64
	__res, err = obj.driver.Exec("DELETE FROM missions;")
	if err != nil {
		return 0, obj.makeErr(err)
	}

	__count, err = __res.RowsAffected()
	if err != nil {
		return 0, obj.makeErr(err)
	}
	count += __count
	__res, err = obj.driver.Exec("DELETE FROM users;")
	if err != nil {
		return 0, obj.makeErr(err)
	}

	__count, err = __res.RowsAffected()
	if err != nil {
		return 0, obj.makeErr(err)
	}
	count += __count

	return count, nil

}

type Rx struct {
	db *DB
	tx *Tx
}

func (rx *Rx) UnsafeTx(ctx context.Context) (unsafe_tx *sql.Tx, err error) {
	tx, err := rx.getTx(ctx)
	if err != nil {
		return nil, err
	}
	return tx.Tx, nil
}

func (rx *Rx) getTx(ctx context.Context) (tx *Tx, err error) {
	if rx.tx == nil {
		if rx.tx, err = rx.db.Open(ctx); err != nil {
			return nil, err
		}
	}
	return rx.tx, nil
}

func (rx *Rx) Rebind(s string) string {
	return rx.db.Rebind(s)
}

func (rx *Rx) Commit() (err error) {
	if rx.tx != nil {
		err = rx.tx.Commit()
		rx.tx = nil
	}
	return err
}

func (rx *Rx) Rollback() (err error) {
	if rx.tx != nil {
		err = rx.tx.Rollback()
		rx.tx = nil
	}
	return err
}

func (rx *Rx) All_Mission_By_User(ctx context.Context,
	mission_user Mission_User_Field) (
	rows []*Mission, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.All_Mission_By_User(ctx, mission_user)
}

func (rx *Rx) CreateNoReturn_Mission(ctx context.Context,
	mission_user Mission_User_Field,
	mission_description Mission_Description_Field,
	mission_seconds Mission_Seconds_Field,
	mission_snacks Mission_Snacks_Field,
	optional Mission_Create_Fields) (
	err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.CreateNoReturn_Mission(ctx, mission_user, mission_description, mission_seconds, mission_snacks, optional)

}

func (rx *Rx) CreateNoReturn_User(ctx context.Context,
	user_name User_Name_Field,
	user_timezone User_Timezone_Field) (
	err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.CreateNoReturn_User(ctx, user_name, user_timezone)

}

func (rx *Rx) Delete_Mission_By_Id(ctx context.Context,
	mission_id Mission_Id_Field) (
	deleted bool, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.Delete_Mission_By_Id(ctx, mission_id)
}

func (rx *Rx) Get_Mission_By_Id(ctx context.Context,
	mission_id Mission_Id_Field) (
	mission *Mission, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.Get_Mission_By_Id(ctx, mission_id)
}

func (rx *Rx) Get_User_By_Name(ctx context.Context,
	user_name User_Name_Field) (
	user *User, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.Get_User_By_Name(ctx, user_name)
}

func (rx *Rx) UpdateNoReturn_Mission_By_Id(ctx context.Context,
	mission_id Mission_Id_Field,
	update Mission_Update_Fields) (
	err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.UpdateNoReturn_Mission_By_Id(ctx, mission_id, update)
}

func (rx *Rx) UpdateNoReturn_User_By_Name(ctx context.Context,
	user_name User_Name_Field,
	update User_Update_Fields) (
	err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.UpdateNoReturn_User_By_Name(ctx, user_name, update)
}

type Methods interface {
	All_Mission_By_User(ctx context.Context,
		mission_user Mission_User_Field) (
		rows []*Mission, err error)

	CreateNoReturn_Mission(ctx context.Context,
		mission_user Mission_User_Field,
		mission_description Mission_Description_Field,
		mission_seconds Mission_Seconds_Field,
		mission_snacks Mission_Snacks_Field,
		optional Mission_Create_Fields) (
		err error)

	CreateNoReturn_User(ctx context.Context,
		user_name User_Name_Field,
		user_timezone User_Timezone_Field) (
		err error)

	Delete_Mission_By_Id(ctx context.Context,
		mission_id Mission_Id_Field) (
		deleted bool, err error)

	Get_Mission_By_Id(ctx context.Context,
		mission_id Mission_Id_Field) (
		mission *Mission, err error)

	Get_User_By_Name(ctx context.Context,
		user_name User_Name_Field) (
		user *User, err error)

	UpdateNoReturn_Mission_By_Id(ctx context.Context,
		mission_id Mission_Id_Field,
		update Mission_Update_Fields) (
		err error)

	UpdateNoReturn_User_By_Name(ctx context.Context,
		user_name User_Name_Field,
		update User_Update_Fields) (
		err error)
}

type TxMethods interface {
	Methods

	Rebind(s string) string
	Commit() error
	Rollback() error
}

type txMethods interface {
	TxMethods

	deleteAll(ctx context.Context) (int64, error)
	makeErr(err error) error
}

type DBMethods interface {
	Methods

	Schema() string
	Rebind(sql string) string
}

type dbMethods interface {
	DBMethods

	wrapTx(tx *sql.Tx) txMethods
	makeErr(err error) error
}

var sqlite3DriverName = func() string {
	var id [16]byte
	rand.Read(id[:])
	return fmt.Sprintf("sqlite3_%x", string(id[:]))
}()

func init() {
	sql.Register(sqlite3DriverName, &sqlite3.SQLiteDriver{
		ConnectHook: sqlite3SetupConn,
	})
}

// SQLite3JournalMode controls the journal_mode pragma for all new connections.
// Since it is read without a mutex, it must be changed to the value you want
// before any Open calls.
var SQLite3JournalMode = "WAL"

func sqlite3SetupConn(conn *sqlite3.SQLiteConn) (err error) {
	_, err = conn.Exec("PRAGMA foreign_keys = ON", nil)
	if err != nil {
		return makeErr(err)
	}
	_, err = conn.Exec("PRAGMA journal_mode = "+SQLite3JournalMode, nil)
	if err != nil {
		return makeErr(err)
	}
	return nil
}

func opensqlite3(source string) (*sql.DB, error) {
	return sql.Open(sqlite3DriverName, source)
}
