package middleware

import (
	"github.com/eudore/erouter"
	"net/http"
	"strings"
)

type (
	// Cors 定义Cors对象。
	Cors struct {
		origins []string
		headers map[string]string
	}
)

// NewCors 函数创建应该Cors对象。
//
// 如果origins为空，设置为*。
// 如果Access-Control-Allow-Methods header为空，设置为*。
func NewCors(origins []string, headers map[string]string) *Cors {
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	if headers["Access-Control-Allow-Methods"] == "" {
		headers["Access-Control-Allow-Methods"] = "*"
	}
	return &Cors{
		origins: origins,
		headers: headers,
	}
}

// NewMiddleware 函数创建应该CORES中间件。
func (cors *Cors) NewMiddleware() erouter.Middleware {
	return func(h erouter.Handler) erouter.Handler {
		return func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				h(w, r, p)
				return
			}

			// 检查是否未同源请求。
			host := r.Host
			if origin == "http://"+host || origin == "https://"+host {
				h(w, r, p)
				return
			}

			if !cors.validateOrigin(origin) {
				w.WriteHeader(403)
				return
			}

			header := w.Header()
			if r.Method == http.MethodOptions {
				for k, v := range cors.headers {
					header.Add(k, v)
				}
				w.WriteHeader(204)
				return
			}
			header.Add("Access-Control-Allow-Origin", origin)
			h(w, r, p)
		}
	}
}

// validateOrigin 方法检查origin是否合法。
func (cors *Cors) validateOrigin(origin string) bool {
	for _, i := range cors.origins {
		if MatchStar(origin, i) {
			return true
		}
	}
	return false
}

// MatchStar 模式匹配对象，允许使用带'*'的模式。
func MatchStar(obj, patten string) bool {
	ps := strings.Split(patten, "*")
	if len(ps) == 0 {
		return patten == obj
	}
	if !strings.HasPrefix(obj, ps[0]) {
		return false
	}
	for _, i := range ps {
		if i == "" {
			continue
		}
		pos := strings.Index(obj, i)
		if pos == -1 {
			return false
		}
		obj = obj[pos+len(i):]
	}
	return true
}
