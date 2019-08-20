# erouter

[![Go Report Card](https://goreportcard.com/badge/github.com/eudore/erouter)](https://goreportcard.com/report/github.com/eudore/erouter)
[![GoDoc](https://godoc.org/github.com/eudore/erouter?status.svg)](https://godoc.org/github.com/eudore/erouter)

erouter是高性能高扩展http路由库，具有零内存复制、严格路由匹配顺序、代码复制度低、组路由、中间件功能、默认参数、常量匹配、变量匹配、通配符匹配、变量校验匹配、通配符校验匹配、基于Host路由这些特点功能。

[设计说明](Design.md)

基于[eudore](https://github.com/eudore/eudore)框架路由分离，修改中间件机制并移除MVC。

## RouterRadix

RouterRadix使用基数树实现，具有零内存复制、严格路由匹配顺序、组路由、中间件功能、默认参数、常量匹配、变量匹配、通配符匹配功能。

example:

```golang
package main

import "log"
import "net/http"
import "github.com/eudore/erouter"

func main() {
	router := erouter.NewRouterRadix()
	router.AddMiddleware("ANY", "", func(h erouter.Handler) erouter.Handler {
		return func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
			log.Printf("%s %s route: %s", r.Method, r.URL.Path, p.GetParam("route"))
			h(w, r, p)
		}
	})
	router.Any("/*", func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
		w.Write([]byte("hello\n"))
	})
	router.Get("/api/*action version=v0", func(w http.ResponseWriter, _ *http.Request, p erouter.Params) {
		w.Write([]byte("access api " + p.GetParam("version") +": " + p.GetParam("action") + "\n"))
	})
	router.Get("/api/v1/*action version=v1", func(w http.ResponseWriter, _ *http.Request, p erouter.Params) {
		w.Write([]byte("access api " + p.GetParam("version") +": " + p.GetParam("action") + "\n"))
	})
	apiv2 := router.Group("/api/v2 version=v2")
	apiv2.Any("/*action", func(w http.ResponseWriter, _ *http.Request, p erouter.Params) {
		w.Write([]byte("access api " + p.GetParam("version") +": " + p.GetParam("action") + "\n"))
	})
	http.ListenAndServe(":8080", router)
}
```

测试命令：

```bash
curl 127.0.0.1:8080/get
curl 127.0.0.1:8080/api/getuser
curl 127.0.0.1:8080/api/v1/getuser
curl 127.0.0.1:8080/api/v2/getuser
```

## RouterFull

RouterFull基于RouterRadix扩展，实现变量校验匹配、通配符校验匹配功能。

用法：在正常变量和通配符后，使用'|'符号分割，后为校验规则，isnum是校验函数；min:100为动态检验函数，min是动态校验函数名称，':'后为参数；如果为'^'开头为正则校验,并且要使用'$'作为结尾。

**注意: 正则表达式不要使用空格，会导致参数切割错误，使用\u002代替空格。**

```
:num|isnum
:num|min:100
:num|^0.*$
*num|isnum
*num|min:100
*num|^0.*$
```

example:

```golang
package main

import "net/http"
import "github.com/eudore/erouter"

func main() {
	router := erouter.NewRouterFull()
	router.Any("/*", func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
		w.Write([]byte("hello\n"))
	})
	router.Get("/:num|^0.*$", func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
		w.Write([]byte("first char is '0', num is: " + p.GetParam("num") + "\n"))
	})
	router.Get("/:num|min:100", func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
		w.Write([]byte("num great 100, num is: " + p.GetParam("num") + "\n"))
	})
	router.Get("/:num|isnum", func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
		w.Write([]byte("num is: " + p.GetParam("num") + "\n"))
	})
	router.Get("/*var|^E.*$", func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
		w.Write([]byte("first char is 'E', var is: " + p.GetParam("var") + "\n"))
	})
	http.ListenAndServe(":8080", router)
}
```

测试命令：

```bash
curl 127.0.0.1:8080/get
curl 127.0.0.1:8080/012
curl 127.0.0.1:8080/123
curl 127.0.0.1:8080/12
curl 127.0.0.1:8080/Erouter/123
```

## RouterHost

RouterHost基于Host匹配，通过选择Host对应子路由器执行注册和匹配，实现基于Host路由功能。

当前使用遍历匹配，Host匹配函数为[path.Match](https://golang.google.cn/pkg/path/#Match)，未来匹配规则仅保留'*'通配符和常量。

用法：需要给Host路由器注册域名规则下的子路由器，注册路由时使用host参数匹配注册的路由器添加路由，匹配时使用请求的Host来匹配注册的路由。

example:

```golang
package main

import "net/http"
import "github.com/eudore/erouter"

func main() {
	router := erouter.NewRouterHost().(*erouter.RouterHost)
	router.RegisterHost("*.example.com", erouter.NewRouterRadix())
	router.Any("/*", func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
		w.Write([]byte("hello\n"))
	})
	router.Get("/* host=*.example.com", func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
		w.Write([]byte("host is " + r.Host + ", match host: " + p.GetParam("host") + "\n"))
	})
	http.ListenAndServe(":8080", router)
}
```

测试命令：

```bash
curl 127.0.0.1:8080
curl -XPUT 127.0.0.1:8080
curl -H 'Host: www.example.com' 127.0.0.1:8080
curl -H 'Host: www.example.com' -XPUT 127.0.0.1:8080
curl -H 'Host: www.example.com' -Xput 127.0.0.1:8080
```

# Middleware

```golang
package main

import (
	"net/http"
	"github.com/eudore/erouter"
	"github.com/eudore/erouter/middleware"
)

func main() {
	router := erouter.NewRouterRadix()
	router.AddMiddleware("ANY", "", 
		middleware.NewLoggerFunc(),
		middleware.NewCors(nil, map[string]string{
			"Access-Control-Allow-Credentials": "true",
			"Access-Control-Allow-Headers": "Authorization,DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,X-Parent-Id",	
			"Access-Control-Expose-Headers": "X-Request-Id",
			"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, HEAD",
			"Access-Control-Max-Age": "1000",
		}).NewMiddleware(),
		middleware.NewCircuitBreaker().InjectRoutes(router.Group("/debug/breaker")).NewMiddleware(),
		middleware.NewRate(10, 30).NewMiddleware(),
	)
	router.Any("/*", func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
		w.Write([]byte("hello\n"))
	})
	router.Get("/api/*action version=v0", func(w http.ResponseWriter, _ *http.Request, p erouter.Params) {
		w.Write([]byte("access api " + p.GetParam("version") +": " + p.GetParam("action") + "\n"))
	})
	router.Get("/api/v1/*action version=v1", func(w http.ResponseWriter, _ *http.Request, p erouter.Params) {
		w.Write([]byte("access api " + p.GetParam("version") +": " + p.GetParam("action") + "\n"))
	})
	apiv2 := router.Group("/api/v2 version=v2")
	apiv2.Any("/*action", func(w http.ResponseWriter, _ *http.Request, p erouter.Params) {
		w.Write([]byte("access api " + p.GetParam("version") +": " + p.GetParam("action") + "\n"))
	})
	http.ListenAndServe(":8080", router)
}
```
# Benchmark

使用GithubApi进行[Benchmark性能测试](https://github.com/eudore/web-framework-benchmark)，Erouter匹配性能静态路由具有httprouter的70%，api匹配性能具有90%且内存分配仅消耗httprouter六分之一，但是具有严格路由匹配顺序、易扩展重写和代码复杂度低的特点。

测试命令：

```bash
go get github.com/eudore/web-framework-benchmark
go test -bench=router github.com/eudore/web-framework-benchmark
```

测试结果：

```
goos: linux
goarch: amd64
pkg: github.com/eudore/web-framework-benchmark
BenchmarkHttprouterStatic-2        	   50000	     25686 ns/op	    1949 B/op	     157 allocs/op
BenchmarkHttprouterGitHubAPI-2     	   30000	     52997 ns/op	   16571 B/op	     370 allocs/op
BenchmarkHttprouterGplusAPI-2      	  500000	      2570 ns/op	     813 B/op	      24 allocs/op
BenchmarkHttprouterParseAPI-2      	  500000	      3791 ns/op	     986 B/op	      42 allocs/op
BenchmarkErouterRadixStatic-2      	   50000	     34314 ns/op	    1950 B/op	     157 allocs/op
BenchmarkErouterRadixGitHubAPI-2   	   30000	     57850 ns/op	    2786 B/op	     203 allocs/op
BenchmarkErouterRadixGplusAPI-2    	  500000	      2468 ns/op	     173 B/op	      13 allocs/op
BenchmarkErouterRadixParseAPI-2    	  300000	      4551 ns/op	     323 B/op	      26 allocs/op
BenchmarkErouterFullStatic-2       	   50000	     34728 ns/op	    1950 B/op	     157 allocs/op
BenchmarkErouterFullGitHubAPI-2    	   30000	     62151 ns/op	    2787 B/op	     203 allocs/op
BenchmarkErouterFullGplusAPI-2     	  500000	      2570 ns/op	     173 B/op	      13 allocs/op
BenchmarkErouterFullParseAPI-2     	  300000	      4362 ns/op	     323 B/op	      26 allocs/op
PASS
ok  	github.com/eudore/web-framework-benchmark	22.356s
```

# Api

列出了主要使用的方法，具体参考[文档](https://godoc.org/github.com/eudore/erouter)。

Router接口定义:

```golang
type (
	// Params读写请求处理中的参数。
	Params interface {
		GetParam(string) string
		AddParam(string, string)
		SetParam(string, string)
	}
	// Erouter处理一个请求的方法，在http.HandlerFunc基础上增加了Parmas。
	Handler func(http.ResponseWriter, *http.Request, Params)
	// 定义请求处理中间件函数，通过传入处理然后返回一个处理，使用装饰器组装处理请求。
	Middleware func(Handler) Handler
	// The route is directly registered by default. Other methods can be directly registered using the RouterRegister interface.
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
	// Router core interface, performing routing, middleware registration, and processing http requests.
	//
	// 路由器核心接口，执行路由、中间件的注册和处理http请求。
	RouterCore interface {
		RegisterMiddleware(string, string, []Middleware)
		RegisterHandler(string, string, Handler)
		ServeHTTP(http.ResponseWriter, *http.Request)
	}
	// The router interface needs to implement two methods: the router method and the router core.
	//
	// 路由器接口，需要实现路由器方法、路由器核心两个接口。
	Router interface {
		RouterCore
		RouterMethod
	}
)
```

## NewRouter

当前拥有三种实现，每种路由器都实现了Router接口。

```
func NewRouterRadix() Router
func NewRouterFull() Router
func NewRouterHost() Router
```

```golang
router1 := erouter.NewRouterRadix()
router2 := erouter.NewRouterFull()
router3 := erouter.NewRouterHost()
```

## Group

`func Group(path string) RouterMethod`

Group实现路由器分组。

```golang
router := erouter.NewRouterRadix()
apiv1 := router.Group("/api/v1 version=v1")
apiv1.Get("/*", ...)
```
## AddHandler

`func AddHandler(method string, path string, handler Handler) RouterMethod`

AddHandler用于添加新路由。

```golang
router := erouter.NewRouterRadix()
router.AddHandle("GET", "/*", ...)
```

## AddMiddleware

`func AddMiddleware(method string, path string, midds ...Middleware) RouterMethod`

AddMiddleware给当前路由方法添加处理中间件。

```golang
router := erouter.NewRouterRadix()
router.AddMiddleware("ANY", "", func(h erouter.Handler) erouter.Handler {
	return func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
		// befor执行
		...
		// 调用next处理汇总函数
		h(w, r, p)
		// after执行
		...
	}
})
```

## NotFound

`func NotFound(Handler)`

设置路由器404处理。

## MethodNotAllowed

`func MethodNotAllowed(Handler)`

设置路由器405处理。

## Any

`func Any(path string, handler Handler)`

注册Any方法，相当于AddHandler的方法为"ANY"。

Any方法的集合为erouter.RouterAllMethod，扩展新方法Radix和Full不支持。

## Get

`func Get(path string, handler Handler)`

注册Get方法，相当于AddHandler的方法为"GET"，post、put等方法函数类似。

```golang
router := erouter.NewRouterRadix()
router.Get("/*", func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
	// 执行处理
})
```