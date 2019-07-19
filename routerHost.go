package erouter

import (
	"net/http"
	"path"
	"strings"
)

// RouterHost 基于Host匹配进行路由。
type RouterHost struct {
	RouterMethod
	Default Router
	Hosts   []string
	Routers []Router
}

// NewRouterHost 创建一个Host路由器，默认子路由器为Radix，其他Host匹配需要将Router类型转换成*RouterHost,然后使用RegisterHost方法注册。
func NewRouterHost() Router {
	router := &RouterHost{
		Default: NewRouterRadix(),
	}
	router.RouterMethod = &RouterMethodStd{RouterCore: router}
	return router
}

func (r *RouterHost) getRouter(path string) Router {
	args := strings.Split(path, " ")
	for _, arg := range args[1:] {
		if arg[:5] == "host=" {
			return r.matchRouter(arg[5:])
		}
	}
	return r.Default
}

func (r *RouterHost) matchRouter(host string) Router {
	for i, h := range r.Hosts {
		if b, _ := path.Match(h, host); b {
			return r.Routers[i]
		}
	}
	return r.Default
}

// RegisterHost 给Host路由器注册域名的子路由器。
//
// 如果host为空字符串，设置为默认子路由器。
func (r *RouterHost) RegisterHost(host string, router Router) {
	if host == "" {
		r.Default = router
		return
	}
	for i, h := range r.Hosts {
		if h == host {
			r.Routers[i] = router
			return
		}
	}
	r.Hosts = append(r.Hosts, host)
	r.Routers = append(r.Routers, router)
}

// RegisterMiddleware 从路径参数中获得host参数，选择对应子路由器注册中间件函数。
func (r *RouterHost) RegisterMiddleware(method, path string, hs []Middleware) {
	r.getRouter(path).RegisterMiddleware(method, path, hs)
}

// RegisterHandler 从路径参数中获得host参数，选择对应子路由器注册新路由。
func (r *RouterHost) RegisterHandler(method string, path string, handler Handler) {
	r.getRouter(path).RegisterHandler(method, path, handler)
}

// ServeHTTP 获取请求的Host匹配对应子路由器处理http请求。
func (r *RouterHost) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.matchRouter(req.Host).ServeHTTP(w, req)
}
