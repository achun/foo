package rest

import (
	"net/http"
)

type Fuler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	Before(*Ful) bool
	After(*Ful) bool
	Get(*Ful)
	Post(*Ful)
	Put(*Ful)
	Delete(*Ful)
}

//  RESTful ServeHTTP 结构
type Ful struct {
	W      http.ResponseWriter
	R      *http.Request
	Path   string
	Before func(*Ful) bool //HiJack
	After  func(*Ful) bool
	Get    func(*Ful)
	Post   func(*Ful)
	Put    func(*Ful)
	Delete func(*Ful)
}

// 如果有错误，写错误信息，并返回是否有错误
func (p *Ful) WriteErr(err error) bool {
	if err != nil {
		p.W.WriteHeader(http.StatusBadRequest)
		p.W.Write([]byte(err.Error()))
	}
	return err != nil
}

// RESTful ServeHTTP 分派
func (p *Ful) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.W = w
	p.R = r
	if p.Before != nil && p.Before(p) {
		return
	}
	defer func() {
		if p.After != nil {
			p.After(p)
		}
	}()
	var f func(*Ful)

	switch r.Method {
	case "GET":
		f = p.Get
	case "POST":
		f = p.Post
	case "PUT":
		f = p.Put
	case "DELETE":
		f = p.Delete
	}
	if f == nil {
		http.Error(w, "Method Not Allowed", 405)
	} else {
		f(p)
	}
}
