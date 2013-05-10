package nor

import (
	"database/sql"
)

type Curder interface {
	Err() error
	Close() error
	Closed() bool
	Get(query string, args ...interface{}) Rowser
	Post(query string, args ...interface{}) sql.Result
	Put(query string, args ...interface{}) sql.Result
	Delete(query string, args ...interface{}) sql.Result
}

//  RESTful ServeHTTP 结构
type Curd struct {
	err     error
	closed  bool
	sqlpool *SqlDBPool
	Db      *sql.DB
}

func NewCurd(db *sql.DB) Curder {
	ret := Curd{Db: db}
	return &ret
}
func (p *Curd) Err() (err error) {
	err, p.err = p.err, nil
	return
}
func (p *Curd) Closed() bool {
	return p.closed
}

func (p *Curd) SqlDBPool(sqlpool *SqlDBPool) *Curd {
	if p.sqlpool == nil {
		p.sqlpool = sqlpool
		p.Db = sqlpool.get()
	}
	return p
}

func (p *Curd) Close() (err error) {
	p.closed = true
	if p.sqlpool != nil {
		p.sqlpool.Recede(p.Db)
		return nil
	}
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
	ret, err = NewRows(rows)
	return
}

func (p *Curd) Post(query string, args ...interface{}) (ret sql.Result) {
	stmt, err := p.Db.Prepare(query)
	if err != nil {
		p.err = err
		return
	}
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
	ret, err = stmt.Exec(args...)
	if err != nil {
		p.err = err
	}
	return
}
