package erouter

import (
	"strings"
)

type (
	// RouterMethodStd 默认路由器方法添加一个实现
	RouterMethodStd struct {
		RouterCore
		prefix string
		tags   string
	}
)

// Group 返回一个组路由方法。
func (m *RouterMethodStd) Group(path string) RouterMethod {
	// 将路径前缀和路径参数分割出来
	args := strings.Split(path, " ")
	prefix := args[0]
	tags := path[len(prefix):]

	// 构建新的路由方法配置器
	return &RouterMethodStd{
		RouterCore: m.RouterCore,
		prefix:     m.prefix + prefix,
		tags:       tags + m.tags,
	}
}

func (m *RouterMethodStd) registerHandlers(method, path string, hs Handler) {
	m.RouterCore.RegisterHandler(method, m.prefix+path+m.tags, hs)
}

// AddHandler 添加一个新路由。
//
// 方法和RegisterHandler方法的区别在于AddHandler方法不会继承Group的路径和参数信息，AddMiddleware相同。
func (m *RouterMethodStd) AddHandler(method, path string, hs Handler) RouterMethod {
	m.registerHandlers(method, path, hs)
	return m
}

// AddMiddleware 给路由器添加一个中间件函数。
func (m *RouterMethodStd) AddMiddleware(method, path string, hs ...Middleware) RouterMethod {
	if len(hs) > 0 {
		m.RegisterMiddleware(method, m.prefix+path+m.tags, hs)
	}
	return m
}

// NotFound 设置404处理。
func (m *RouterMethodStd) NotFound(h Handler) {
	m.RouterCore.RegisterHandler("404", "", h)
}

// MethodNotAllowed 设置405处理。
func (m *RouterMethodStd) MethodNotAllowed(h Handler) {
	m.RouterCore.RegisterHandler("405", "", h)
}

// Any Router Register handler。
func (m *RouterMethodStd) Any(path string, h Handler) {
	m.registerHandlers(MethodAny, path, h)
}

// Get 添加一个GET方法请求处理。
func (m *RouterMethodStd) Get(path string, h Handler) {
	m.registerHandlers(MethodGet, path, h)
}

// Post 添加一个POST方法请求处理。
func (m *RouterMethodStd) Post(path string, h Handler) {
	m.registerHandlers(MethodPost, path, h)
}

// Put 添加一个PUT方法请求处理。
func (m *RouterMethodStd) Put(path string, h Handler) {
	m.registerHandlers(MethodPut, path, h)
}

// Delete 添加一个DELETE方法请求处理。
func (m *RouterMethodStd) Delete(path string, h Handler) {
	m.registerHandlers(MethodDelete, path, h)
}

// Head 添加一个HEAD方法请求处理。
func (m *RouterMethodStd) Head(path string, h Handler) {
	m.registerHandlers(MethodHead, path, h)
}

// Patch 添加一个PATCH方法请求处理。
func (m *RouterMethodStd) Patch(path string, h Handler) {
	m.registerHandlers(MethodPatch, path, h)
}

// Options 添加一个OPTIONS方法请求处理。
func (m *RouterMethodStd) Options(path string, h Handler) {
	m.registerHandlers(MethodOptions, path, h)
}
