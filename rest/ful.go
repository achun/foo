// 类型复合在开发中很常见，多数情况下，内层类型访问外层类型的属性，需要使用 reflect 包完成。
// 虽然 reflect 相当高效，但是随着代码规模的扩大，问题会越来越复杂。
// rest 包推荐直接使用类型完成任务的方式，而不是使用类型复合。
// 这种使用方式看上去有些古怪，不过开发者将获得很大的自由度，同时也降低了 rest 包本身的维护。
package rest

import (
	"net/http"
)

// Ful 是一个简单的 http.Handler 实现
// Ful 依据 Request.Method 调用对应的函数
// 示例: 直接使用而不是用struct再次包装
// 		http.Handle(pattern, &Ful{
// 			Get: func(fu *Ful) {
// 				something()
// 			},
//			After: func(fu *Ful) {
// 				something()
// 			},
// 		})
//
type Ful struct {
	W      http.ResponseWriter
	R      *http.Request
	Before func(fu *Ful) bool             // 调用对应函数前，先调用 Before ，返回 false,表示跳过函数调用
	After  func(fu *Ful, err interface{}) // 最后调用的函数，err 是函数调用中 recover 到的。
	Get    func(fu *Ful)
	Post   func(fu *Ful)
	Put    func(fu *Ful)
	Delete func(fu *Ful)
}

// HandleFunc 分派
// 如果没有设置对应 Request.Method 的函数，将向 ResponseWriter 写入 405 Method Not Allowed
// 如果
func (fu *Ful) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := new(Ful)
	p.W, p.R = w, r
	p.Before, p.After, p.Get, p.Post, p.Put, p.Delete = fu.Before, fu.After, fu.Get, fu.Post, fu.Put, fu.Delete

	defer func() {
		err := recover()
		if p.After != nil {
			p.After(p, err)
			return
		}
		if err != nil {
			p.WriteHeader(500)
		}
	}()

	if p.Before != nil && !p.Before(p) {
		return
	}
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
		p.WriteHeader(405).Write("Method Not Allowed")
	} else {
		f(p)
	}
}

func (p *Ful) Write(content string) *Ful {
	p.W.Write([]byte(content))
	return p
}

func (p *Ful) WriteHeader(stat int) *Ful {
	p.W.WriteHeader(stat)
	return p
}

func (p *Ful) Redirect(stat int, url string) *Ful {
	p.W.Header().Set("Location", url)
	p.W.WriteHeader(stat)
	return p
}

func (p *Ful) SetHeader(hdr string, val string) *Ful {
	p.W.Header().Set(hdr, val)
	return p
}

func (p *Ful) AddHeader(hdr string, val string) *Ful {
	p.W.Header().Add(hdr, val)
	return p
}

// 设置一个 Path=="/" 的 cookie
func (p *Ful) SetCookie(name string, value string, maxAge int) *Ful {
	cookie := &http.Cookie{Path: "/", Name: name, Value: value, MaxAge: maxAge}
	http.SetCookie(p.W, cookie)
	return p
}

// Fu 与 Ful 类似，区别在使用上，需要提供一个生成器或者使用闭包的形式
// 示例: 因为闭包的存在，Get 之类的函数不再需要定义参数
// 		http.Handle(pattern,FuGen(func() *Fu{
//			var fu *Fu
//			fu = &Fu{
// 				Get: func() {
//	 				something(fu)
//				},
//				After: func() {
//					something(fu)
//				},
// 			}
//			return fu
// 		}))
// 看上去确实有些古怪，此方式结合了闭包和生成器
type Fu struct {
	gen    func() *Fu
	W      http.ResponseWriter
	R      *http.Request
	Before func() bool           // 调用对应函数前，先调用 Before ，返回 false,表示跳过函数调用
	After  func(err interface{}) // 最后调用的函数，err 是函数调用中 recover 到的。
	Get    func()
	Post   func()
	Put    func()
	Delete func()
}

// 由生成 type Fu 的函数 gen,构建 http.Handel
func FuGen(gen func() *Fu) *Fu {
	return &Fu{gen: gen}
}

// HandleFunc 分派
// 如果没有设置对应 Request.Method 的函数，将向 ResponseWriter 写入 405 Method Not Allowed
// 如果
func (fu *Fu) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var p *Fu
	defer func() {
		err := recover()
		if p != nil && p.After != nil {
			p.After(err)
			return
		}
		if err != nil {
			w.WriteHeader(500)
		}
	}()
	p = fu.gen()
	p.W, p.R = w, r

	if p.Before != nil && !p.Before() {
		return
	}
	var f func()

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
		p.WriteHeader(405).Write("Method Not Allowed")
	} else {
		f()
	}
}

func (p *Fu) Write(content string) *Fu {
	p.W.Write([]byte(content))
	return p
}

func (p *Fu) WriteHeader(stat int) *Fu {
	p.W.WriteHeader(stat)
	return p
}

func (p *Fu) Redirect(stat int, url string) *Fu {
	p.W.Header().Set("Location", url)
	p.W.WriteHeader(stat)
	return p
}

func (p *Fu) SetHeader(hdr string, val string) *Fu {
	p.W.Header().Set(hdr, val)
	return p
}

func (p *Fu) AddHeader(hdr string, val string) *Fu {
	p.W.Header().Add(hdr, val)
	return p
}

// 设置一个 Path=="/" 的 cookie
func (p *Fu) SetCookie(name string, value string, maxAge int) *Fu {
	cookie := &http.Cookie{Path: "/", Name: name, Value: value, MaxAge: maxAge}
	http.SetCookie(p.W, cookie)
	return p
}
