package middleware

import (
	"fmt"
	"github.com/eudore/erouter"
	"net/http"
)

// NewRecoverFunc 函数创建一个异常捕捉中间件。
func NewRecoverFunc() erouter.Middleware {
	return func(h erouter.Handler) erouter.Handler {
		return func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
			defer func() {
				if r := recover(); r != nil {
					http.Error(w, fmt.Sprint(r), 500)
				}
			}()
			h(w, r, p)
		}
	}
}
