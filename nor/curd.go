package nor

import (
	"database/sql"
	"errors"
)

// Curder 提供了一种简易方法操作数据库
// Get 方法返回 Rowser，细节见 Rowser
// Post,Put,Delete都是单条 update 操作的情况,没有对重复SQL进行Prepare优化支持
// 如果要使用Prepare优化SQL请使用 Prepare,Exec
type Curder interface {
	Err() error
	Close() error
	Closed() bool
	Get(query string, args ...interface{}) Rowser
	Post(query string, args ...interface{}) sql.Result
	Put(query string, args ...interface{}) sql.Result
	Delete(query string, args ...interface{}) sql.Result
	Prepare(query string) error
	Exec(args ...interface{}) sql.Result
}

//  RESTful ServeHTTP 结构
type Curd struct {
	err    error
	closed bool
	Db     *sql.DB
	stmt   *sql.Stmt
}

func NewCurd(db *sql.DB) Curder {
	ret := Curd{Db: db}
	return &ret
}
func (p *Curd) Err() (err error) {
	return p.err
}
func (p *Curd) Closed() bool {
	return p.closed
}

func (p *Curd) Close() (err error) {
	p.closed = true
	return p.Db.Close()
}

func (p *Curd) Get(query string, args ...interface{}) (ret Rowser) {
	stmt, err := p.Db.Prepare(query)
	if err != nil {
		p.err = err
		return
	}
	rows, err := stmt.Query(args...)
	if err != nil {
		defer stmt.Close()
		p.err = err
		return
	}
	ret, p.err = NewRows(rows)
	return
}

func (p *Curd) Prepare(query string) (err error) {
	if p.stmt != nil {
		err = p.stmt.Close()
		if err != nil {
			p.err = err
			return
		}
	}
	stmt, err := p.Db.Prepare(query)
	if err != nil {
		p.err = err
		return err
	}
	p.stmt = stmt
	return
}

func (p *Curd) Exec(args ...interface{}) (ret sql.Result) {
	if p.stmt == nil {
		p.err = errors.New("to Prepare before Exec")
		return
	}
	ret, err := p.stmt.Exec(args...)
	if err != nil {
		p.err = err
	}
	return
}

func (p *Curd) Post(query string, args ...interface{}) (ret sql.Result) {
	stmt, err := p.Db.Prepare(query)
	if err != nil {
		p.err = err
		return
	}
	defer stmt.Close()
	ret, err = stmt.Exec(args...)
	if err != nil {
		p.err = err
	}
	return
}

func (p *Curd) Put(query string, args ...interface{}) (ret sql.Result) {
	stmt, err := p.Db.Prepare(query)
	if err != nil {
		p.err = err
		return
	}
	defer stmt.Close()
	ret, err = stmt.Exec(args...)
	if err != nil {
		p.err = err
	}
	return
}

func (p *Curd) Delete(query string, args ...interface{}) (ret sql.Result) {
	stmt, err := p.Db.Prepare(query)
	if err != nil {
		p.err = err
		return
	}
	defer stmt.Close()
	ret, err = stmt.Exec(args...)
	if err != nil {
		p.err = err
	}
	return
}
