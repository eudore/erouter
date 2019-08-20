package middleware

import (
	"github.com/eudore/erouter"
	"log"
	"net/http"
)

// NewLoggerFunc 函数创建一个日志输出中间件。
func NewLoggerFunc() erouter.Middleware {
	return func(h erouter.Handler) erouter.Handler {
		return func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
			ws, ok := w.(ResponseWriteStatuer)
			if !ok {
				ws = &ResponseWriter{w, 200}
			}
			h(ws, r, p)
			log.Printf("%s %d %s %s", GetRealClientIP(r), ws.GetStatue(), r.Method, r.URL.Path)
		}
	}
}
