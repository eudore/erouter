package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/eudore/erouter"
	"golang.org/x/time/rate"
)

// Rate 定义限流器
type Rate struct {
	visitors map[string]*visitor
	mtx      sync.Mutex
	Rate     rate.Limit
	Burst    int
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRate 创建一个限流器。
//
// 周期内增加r2个令牌，最多拥有burst个。
func NewRate(r2, burst int) *Rate {
	r := &Rate{
		visitors: make(map[string]*visitor),
		Rate:     rate.Limit(r2),
		Burst:    burst,
	}
	go r.cleanupVisitors()
	return r
}

// NewMiddleware 返回一个限流处理函数。
func (r *Rate) NewMiddleware() erouter.Middleware {
	return func(h erouter.Handler) erouter.Handler {
		return func(w http.ResponseWriter, req *http.Request, p erouter.Params) {
			ip := GetRealClientIP(req)
			limiter := r.GetVisitor(ip)
			if !limiter.Allow() {
				http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
				return
			}
			h(w, req, p)
		}
	}
}

// GetVisitor 方法通过ip获得*rate.Limiter。
func (r *Rate) GetVisitor(ip string) *rate.Limiter {
	r.mtx.Lock()
	v, exists := r.visitors[ip]
	if !exists {
		r.mtx.Unlock()
		return r.AddVisitor(ip)
	}
	// Update the last seen time for the visitor.
	v.lastSeen = time.Now()
	r.mtx.Unlock()
	return v.limiter
}

// AddVisitor Change the the map to hold values of the type visitor.
func (r *Rate) AddVisitor(ip string) *rate.Limiter {
	limiter := rate.NewLimiter(r.Rate, r.Burst)
	r.mtx.Lock()
	// Include the current time when creating a new visitor.
	r.visitors[ip] = &visitor{limiter, time.Now()}
	r.mtx.Unlock()
	return limiter
}

func (r *Rate) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		r.mtx.Lock()
		for ip, v := range r.visitors {
			if time.Now().Sub(v.lastSeen) > 3*time.Minute {
				delete(r.visitors, ip)
			}
		}
		r.mtx.Unlock()
	}
}

// GetRealClientIP 函数获取http请求的真实ip
func GetRealClientIP(r *http.Request) string {
	xforward := r.Header.Get("X-Forwarded-For")
	if "" == xforward {
		return strings.SplitN(r.RemoteAddr, ":", 2)[0]
	}

	return strings.SplitN(string(xforward), ",", 2)[0]
}
