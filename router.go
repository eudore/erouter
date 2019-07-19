package erouter

import (
	"net/http"
	"strings"
)

// 默认http请求方法
const (
	MethodAny     = "ANY"
	MethodGet     = "GET"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodDelete  = "DELETE"
	MethodHead    = "HEAD"
	MethodPatch   = "PATCH"
	MethodOptions = "OPTIONS"
	MethodConnect = "CONNECT"
	MethodTrace   = "TRACE"
)

type (
	// Params 读写请求处理中的参数。
	Params interface {
		GetParam(string) string
		AddParam(string, string)
		SetParam(string, string)
	}
	// Handler 是Erouter处理一个请求的方法，在http.HandlerFunc基础上增加了Parmas。
	Handler func(http.ResponseWriter, *http.Request, Params)
	// Middleware 定义请求处理中间件函数，通过传入处理然后返回一个处理，使用装饰器组装处理请求。
	Middleware func(Handler) Handler
	// RouterMethod route is directly registered by default. Other methods can be directly registered using the RouterRegister interface.
	//
	// 路由默认直接注册的方法，其他方法可以使用RouterRegister接口直接注册。
	RouterMethod interface {
		Group(string) RouterMethod
		AddHandler(string, string, Handler) RouterMethod
		AddMiddleware(string, string, ...Middleware) RouterMethod
		NotFound(Handler)
		MethodNotAllowed(Handler)
		Any(string, Handler)
		Delete(string, Handler)
		Get(string, Handler)
		Head(string, Handler)
		Options(string, Handler)
		Patch(string, Handler)
		Post(string, Handler)
		Put(string, Handler)
	}
	// RouterCore interface, performing routing, middleware registration, and processing http requests.
	//
	// 路由器核心接口，执行路由、中间件的注册和处理http请求。
	RouterCore interface {
		RegisterMiddleware(string, string, []Middleware)
		RegisterHandler(string, string, Handler)
		ServeHTTP(http.ResponseWriter, *http.Request)
	}
	// Router interface needs to implement two methods: the router method and the router core.
	//
	// 路由器接口，需要实现路由器方法、路由器核心两个接口。
	Router interface {
		RouterCore
		RouterMethod
	}
)

var (
	// ParamRoute 是路由参数键值
	ParamRoute = "route"
	// Page404 是404返回的body
	Page404 = []byte("404 page not found\n")
	// Page405 是405返回的body
	Page405 = []byte("405 method not allowed\n")
	// RouterAllMethod 是默认Any的全部方法
	RouterAllMethod              = []string{MethodGet, MethodPost, MethodPut, MethodDelete, MethodHead, MethodPatch, MethodOptions}
	_               Params       = (*ParamsArray)(nil)
	_               RouterMethod = (*RouterMethodStd)(nil)
	_               Router       = (*RouterRadix)(nil)
	_               Router       = (*RouterFull)(nil)
	_               Router       = (*RouterHost)(nil)
)

// NewHandler 根据http.Handler和http.HandlerFunc返回erouter.Handler
func NewHandler(i interface{}) Handler {
	switch v := i.(type) {
	case Handler:
		return v
	case func(http.ResponseWriter, *http.Request, Params):
		return v
	case http.Handler:
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			v.ServeHTTP(w, r)
		}
	case http.HandlerFunc:
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			v(w, r)
		}
	case func(http.ResponseWriter, *http.Request):
		return func(w http.ResponseWriter, r *http.Request, p Params) {
			v(w, r)
		}
	}
	return nil
}

// CombineHandler 合并全部中间件处理函数。
func CombineHandler(handler Handler, m []Middleware) Handler {
	if m == nil {
		return handler
	}
	h := handler
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

// 默认405处理，返回405状态码和允许的方法
func defaultRouter405Func(w http.ResponseWriter, req *http.Request, param Params) {
	w.Header().Add("Allow", strings.Join(RouterAllMethod, ", "))
	w.WriteHeader(405)
	w.Write(Page405)
}

// 默认404处理，返回404状态码
func defaultRouter404Func(w http.ResponseWriter, req *http.Request, param Params) {
	w.WriteHeader(404)
	w.Write(Page404)
}
