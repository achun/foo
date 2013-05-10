package nor

import (
	"database/sql"
	"database/sql/driver"
)

type SqlDB interface {
	Begin() (*sql.Tx, error)
	//Close() error hide method
	Driver() driver.Driver
	Exec(query string, args ...interface{}) (sql.Result, error)
	Ping() error
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}
type SqlDBPool struct {
	driverName     string
	dataSourceName string
	size           uint
	inited         bool
	mi             map[interface{}]*sql.DB
	pool           chan *sql.DB
}

func (p *SqlDBPool) InitPool(driverName string, dataSourceName string, size uint) error {
	if p.inited {
		return nil
	}
	p.inited = true
	p.mi = map[interface{}]*sql.DB{}
	p.driverName = driverName
	p.dataSourceName = dataSourceName
	p.pool = make(chan *sql.DB, size)
	for x := uint(0); x < size; x++ {
		err := p.push()
		if err != nil {
			return err
		}
	}
	p.size = size
	return nil
}
func (p *SqlDBPool) push() error {
	conn, err := sql.Open(p.driverName, p.dataSourceName)
	if err == nil {
		p.mi[interface{}(conn)] = conn
		p.pool <- conn
	}
	return err
}
func (p *SqlDBPool) Recede(s SqlDB) {
	if !p.inited {
		return
	}
	conn, ok := p.mi[s]
	if ok {
		if conn.Ping() == nil {
			p.pool <- conn
		} else {
			delete(p.mi, s)
			p.push()
		}
	}
}
func (p *SqlDBPool) Get() SqlDB {
	if !p.inited {
		return nil
	}
	c := <-p.pool
	return c
}

func (p *SqlDBPool) get() *sql.DB {
	if !p.inited {
		return nil
	}
	c := <-p.pool
	return c
}

func (p *SqlDBPool) ClearAll() {
	for len(p.pool) > 0 {
		c := <-p.pool
		c.Close()
	}
}

// Example:
//	var dbPool = &SqlDBPool{}
//	dbPool.InitPool("sqlite3",":memory:",3)
//	c := dbPool.Get()
//	dbPool.Recede(c)
//	dbPool.ClearAll()
