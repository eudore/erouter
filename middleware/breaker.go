package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/eudore/erouter"
)

// 获取文件定义位置，静态ui文件在同目录。
func init() {
	_, file, _, ok := runtime.Caller(0)
	if ok {
		StaticHtml = file[:len(file)-2] + "html"
	}
}

// 定义熔断器状态。
const (
	CircuitBreakerStatueClosed State = iota
	CircuitBreakerStatueHalfOpen
	CircuitBreakerStatueOpen
)

// 半开状态时最大连续失败和最大连续成功次数。
var (
	MaxConsecutiveSuccesses uint32 = 10
	MaxConsecutiveFailures  uint32 = 10
	// CircuitBreakerStatues 定义熔断状态字符串
	StaticHtml            = ""
	CircuitBreakerStatues = []string{"closed", "half-open", "open"}
)

type (
	// State 是熔断器状态。
	State int8
	// CircuitBreaker 定义熔断器。
	CircuitBreaker struct {
		mu            sync.RWMutex
		num           int
		Mapping       map[int]string             `json:"mapping"`
		Routes        map[string]*Route          `json:"routes"`
		OnStateChange func(string, State, State) `json:"-"`
	}
	// Route 定义单词路由的熔断数据。
	Route struct {
		mu                   sync.Mutex
		Id                   int
		Name                 string
		State                State
		LastTime             time.Time
		TotalSuccesses       uint64
		TotalFailures        uint64
		ConsecutiveSuccesses uint32
		ConsecutiveFailures  uint32
		OnStateChange        func(string, State, State) `json:"-"`
	}
	// ResponseWriter 封装http.ResponseWriter对象，用于获得写入的状态码。
	ResponseWriter struct {
		http.ResponseWriter
		statue int
	}
	// ResponseWriteStatuer 定义http.ResponseWriter获得状态码的接口。
	ResponseWriteStatuer interface {
		http.ResponseWriter
		GetStatue() int
	}
)

// NewCircuitBreaker 函数创建一个熔断器
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		Mapping: make(map[int]string),
		Routes:  make(map[string]*Route),
		OnStateChange: func(name string, from State, to State) {
			log.Printf("CircuitBreaker route %s change state from %s to %s", name, from, to)
		},
	}
}

// NewMiddleware 创建一个熔断器中间件。
func (cb *CircuitBreaker) NewMiddleware() erouter.Middleware {
	return func(h erouter.Handler) erouter.Handler {
		return func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
			name := p.GetParam("route")
			cb.mu.RLock()
			route, ok := cb.Routes[name]
			cb.mu.RUnlock()
			if !ok {
				cb.mu.Lock()
				route = &Route{
					Id:            cb.num,
					Name:          name,
					LastTime:      time.Now(),
					OnStateChange: cb.OnStateChange,
				}
				cb.Mapping[cb.num] = name
				cb.Routes[name] = route
				cb.num++
				cb.mu.Unlock()
			}

			if route.IsDeny() {
				w.WriteHeader(503)
				return
			}
			response, ok := w.(ResponseWriteStatuer)
			if !ok {
				response = &ResponseWriter{w, 200}
			}
			h(response, r, p)
			if response.GetStatue() < 500 {
				route.onSuccess()
			} else {
				route.onFailure()
			}
		}
	}
}

// InjectRoutes 方法给给路由器注入熔断器的路由。
func (cb *CircuitBreaker) InjectRoutes(r erouter.RouterMethod) *CircuitBreaker {
	r.Get("/ui", func(w http.ResponseWriter, r *http.Request, _ erouter.Params) {
		http.ServeFile(w, r, StaticHtml)
	})
	r.Get("/list", func(w http.ResponseWriter, _ *http.Request, _ erouter.Params) {
		body, err := json.Marshal(cb.Routes)
		if err != nil {
			http.Error(w, fmt.Sprint(err), 50)
			return
		}
		w.Write(body)
	})
	r.Get("/:id", func(w http.ResponseWriter, _ *http.Request, p erouter.Params) {
		id := GetStringDefaultInt(p.GetParam("id"), -1)
		if id < 0 || id >= cb.num {
			http.Error(w, "id is invalid", 500)
			return
		}
		cb.mu.RLock()
		route := cb.Routes[cb.Mapping[id]]
		cb.mu.RUnlock()
		body, err := json.Marshal(route)
		if err != nil {
			http.Error(w, fmt.Sprint(err), 50)
			return
		}
		w.Write(body)

	})
	r.Put("/:id/state/:state", func(w http.ResponseWriter, _ *http.Request, p erouter.Params) {
		id := GetStringDefaultInt(p.GetParam("id"), -1)
		state := GetStringDefaultInt(p.GetParam("state"), -1)
		if id < 0 || id >= cb.num {
			http.Error(w, "id is invalid", 500)
			return
		}
		if state < -1 || state > 2 {
			http.Error(w, "state is invalid", 500)
			return
		}
		cb.mu.RLock()
		route := cb.Routes[cb.Mapping[id]]
		cb.mu.RUnlock()
		route.OnStateChange(route.Name, route.State, State(state))
		route.State = State(state)
		route.ConsecutiveSuccesses = 0
		route.ConsecutiveFailures = 0
	})
	return cb
}

// IsDeny 方法实现熔断器条目是否可以通过。
func (c *Route) IsDeny() (b bool) {
	if c.State == CircuitBreakerStatueHalfOpen {
		b = time.Now().Before(c.LastTime.Add(400 * time.Millisecond))
		if b {
			c.LastTime = time.Now()
		}
		return b
	}
	return c.State == CircuitBreakerStatueOpen
}

// onSuccess 方法处理熔断器条目成功的情况。
func (c *Route) onSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
	if c.State != CircuitBreakerStatueClosed && c.ConsecutiveSuccesses > MaxConsecutiveSuccesses {
		c.ConsecutiveSuccesses = 0
		c.State--
		c.LastTime = time.Now()
	}
}

// onFailure 方法处理熔断器条目失败的情况。
func (c *Route) onFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
	if c.State != CircuitBreakerStatueOpen && c.ConsecutiveFailures > MaxConsecutiveFailures {
		c.ConsecutiveFailures = 0
		c.State++
		c.LastTime = time.Now()
	}
}

// String 方法实现string接口
func (state State) String() string {
	return CircuitBreakerStatues[state]
}

// MarshalText 方法实现encoding.TextMarshaler接口。
func (state State) MarshalText() (text []byte, err error) {
	text = []byte(state.String())
	return
}

// WriteHeader 实现http.ResponseWriter的WriteHeader方法，将响应状态码记录。
func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.statue = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// GetStatue 获取响应的状态码。
func (w *ResponseWriter) GetStatue() int {
	return w.statue
}

// GetStringDefaultInt 实现字符串转换成int。
func GetStringDefaultInt(str string, n int) int {
	if v, err := strconv.Atoi(str); err == nil {
		return v
	}
	return n
}
