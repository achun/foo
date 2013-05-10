package rest

import (
	"net/http"
)

type Fuler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	Before(http.ResponseWriter, *http.Request) bool
	After(http.ResponseWriter, *http.Request) bool
	Get(http.ResponseWriter, *http.Request)
	Post(http.ResponseWriter, *http.Request)
	Put(http.ResponseWriter, *http.Request)
	Delete(http.ResponseWriter, *http.Request)
}

//  RESTful ServeHTTP 结构
type Ful struct {
	Path   string
	Before func(http.ResponseWriter, *http.Request) bool //HiJack
	After  func(http.ResponseWriter, *http.Request) bool
	Get    func(http.ResponseWriter, *http.Request)
	Post   func(http.ResponseWriter, *http.Request)
	Put    func(http.ResponseWriter, *http.Request)
	Delete func(http.ResponseWriter, *http.Request)
}

// RESTful ServeHTTP 分派
func (p *Ful) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.Before != nil && p.Before(w, r) {
		return
	}
	defer func() {
		if p.After != nil {
			p.After(w, r)
		}
	}()
	var f func(http.ResponseWriter, *http.Request)

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
		f(w, r)
	}
}
