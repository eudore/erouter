package erouter

/*
基于基数树算法实现一个完整路由器
*/

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const (
	fullNodeKindConst    uint8 = iota // 常量
	fullNodeKindRegex                 // 参数正则或函数校验
	fullNodeKindParam                 // 参数
	fullNodeKindValid                 // 通配符正则或函数校验
	fullNodeKindWildcard              // 通配符
)

type (
	// RouterCheckFunc Route data validation function to check if a string parameter is valid.
	//
	// RouterCheckFunc路由数据校验函数，检查一个字符串参数是否有效。
	RouterCheckFunc func(string) bool
	// RouterNewCheckFunc Routing data validation function creation function
	//
	// Construct a new validation function by specifying a string.
	//
	// RouterNewCheckFunc路由数据校验函数的创建函数
	//
	// 通过指定字符串构造出一个新的校验函数。
	RouterNewCheckFunc func(string) RouterCheckFunc
	// RouterFull is implemented based on the radix tree to implement all router related features.
	//
	// Based on the RouterRadix extension, RouterFull implements variable check matching and wildcard check matching.
	//
	// RouterFull基于基数树实现，实现全部路由器相关特性。
	//
	// RouterFull基于RouterRadix扩展，实现变量校验匹配、通配符校验匹配功能。
	RouterFull struct {
		RouterMethod
		// save middleware
		// 保存注册的中间件信息
		middtree    *middNode
		node404     fullNode
		nodefunc404 Handler
		node405     fullNode
		nodefunc405 Handler
		root        fullNode
		get         fullNode
		post        fullNode
		put         fullNode
		delete      fullNode
		options     fullNode
		head        fullNode
		patch       fullNode
	}
	fullNode struct {
		path string
		kind uint8
		pnum uint8
		name string
		// 保存各类子节点
		Cchildren []*fullNode
		Rchildren []*fullNode
		Pchildren []*fullNode
		Vchildren []*fullNode
		Wchildren *fullNode
		// 默认标签的名称和值
		tags []string
		vals []string
		// 校验函数
		check RouterCheckFunc
		// 正则捕获名称和函数
		// names		[]string
		// find		RouterFindFunc
		// 路由匹配的处理者
		handlers Handler
	}
)

// NewRouterFull 创建一个Full路由器，基于基数数实现，使用Radix路由器扩展，新增参数校验功能。
func NewRouterFull() Router {
	router := &RouterFull{
		middtree:    &middNode{},
		nodefunc404: defaultRouter404Func,
		nodefunc405: defaultRouter405Func,
		node404: fullNode{
			tags:     []string{ParamRoute},
			vals:     []string{"404"},
			handlers: defaultRouter404Func,
		},
		node405: fullNode{
			Wchildren: &fullNode{
				tags:     []string{ParamRoute},
				vals:     []string{"405"},
				handlers: defaultRouter405Func,
			},
		},
	}
	router.RouterMethod = &RouterMethodStd{
		RouterCore: router,
	}
	return router
}

// RegisterMiddleware Register the middleware into the middleware tree and append the handler if it exists.
//
// If the method is not empty, the path is empty and the modified path is '/'.
//
// RegisterMiddleware注册中间件到中间件树中，如果存在则追加处理者。
//
// 如果方法非空，路径为空，修改路径为'/'。
func (r *RouterFull) RegisterMiddleware(method, path string, hs []Middleware) {
	// 移除路径中的参数
	if pos := strings.IndexByte(path, ' '); pos != -1 {
		path = path[:pos]
	}
	if len(method) != 0 && len(path) == 0 {
		path = "/"
	}
	if method == MethodAny {
		if path == "/" {
			r.middtree.Insert("", hs)
			r.node404.handlers = CombineHandler(r.nodefunc404, r.middtree.val)
			r.node405.Wchildren.handlers = CombineHandler(r.nodefunc405, r.middtree.val)
			return
		}
		for _, method = range RouterAllMethod {
			r.middtree.Insert(method+path, hs)
		}
	} else {
		r.middtree.Insert(method+path, hs)
	}
}

// RegisterHandler Register a new method request path to the router
//
// The router matches the handlers available to the current path from the middleware tree and adds them to the front of the handler.
//
// RegisterHandler给路由器注册一个新的方法请求路径
//
// 路由器会从中间件树中匹配当前路径可使用的处理者，并添加到处理者前方。
func (r *RouterFull) RegisterHandler(method string, path string, handler Handler) {
	switch method {
	case "NotFound", "404":
		r.nodefunc404 = handler
		r.node404.handlers = CombineHandler(handler, r.middtree.val)
	case "MethodNotAllowed", "405":
		r.nodefunc405 = handler
		r.node405.Wchildren.handlers = CombineHandler(handler, r.middtree.val)
	case MethodAny:
		for _, method := range RouterAllMethod {
			r.insertRoute(method, path, CombineHandler(handler, r.middtree.Lookup(method+path)))
		}
	default:
		r.insertRoute(method, path, CombineHandler(handler, r.middtree.Lookup(method+path)))
	}
}

// Add a new route Node.
//
// If the method does not support it will not be added, request to change the path will respond 405
//
// 添加一个新的路由Node。
//
// 如果方法不支持则不会添加，请求改路径会响应405
func (r *RouterFull) insertRoute(method, key string, val Handler) {
	var currentNode *fullNode = r.getTree(method)
	if currentNode == &r.node405 {
		return
	}

	// 创建节点
	args := strings.Split(key, " ")
	for _, path := range getSplitPath(args[0]) {
		currentNode = currentNode.InsertNode(path, newFullNode(path))
	}

	currentNode.handlers = val
	currentNode.SetTags(args)
}

// 实现http.Handler接口，进行路由匹配并处理http请求。
func (r *RouterFull) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	p := paramArrayPool.Get().(*ParamsArray)
	p.Reset()
	hs := r.Match(req.Method, req.URL.Path, p)
	hs(w, req, p)
	paramArrayPool.Put(p)
}

// Match a request, if the method does not allow direct return to node405, no match returns node404.
//
// 匹配一个请求，如果方法不不允许直接返回node405，未匹配返回node404。
func (r *RouterFull) Match(method, path string, params Params) Handler {
	if n := r.getTree(method).recursiveLoopup(path, params); n != nil {
		return n
	}

	// 处理404
	r.node404.AddTagsToParams(params)
	return r.node404.handlers
}

// Create a 405 response radixNode.
//
// 创建一个405响应的radixNode。
func newFullNode405(args string, h Handler) *fullNode {
	newNode := &fullNode{
		Wchildren: &fullNode{
			handlers: h,
		},
	}
	newNode.Wchildren.SetTags(strings.Split(args, " "))
	return newNode
}

// 创建一个Radix树Node，会根据当前路由设置不同的节点类型和名称。
//
// '*'前缀为通配符节点，':'前缀为参数节点，其他未常量节点,如果通配符和参数结点后带有符号'|'则为校验结点。
func newFullNode(path string) *fullNode {
	newNode := &fullNode{path: path}
	switch path[0] {
	case '*':
		newNode.kind = fullNodeKindWildcard
		if len(path) == 1 {
			newNode.name = "*"
		} else {
			newNode.name = path[1:]
			// 如果路径后序具有'|'符号，则截取后端名称返回校验函数
			// 并升级成校验通配符Node
			if name, fn := loadCheckFunc(path); len(name) > 0 {
				// 无法获得校验函数抛出错误
				if fn == nil {
					panic("loadCheckFunc path is invalid, load func failure " + path)
				}
				newNode.kind, newNode.name, newNode.check = fullNodeKindValid, name, fn
			}
		}
	case ':':
		newNode.kind = fullNodeKindParam
		newNode.name = path[1:]
		// 如果路径后序具有'|'符号，则截取后端名称返回校验函数
		// 并升级成校验参数Node
		if name, fn := loadCheckFunc(path); len(name) > 0 {
			if fn == nil {
				panic("loadCheckFunc path is invalid, load func failure " + path)
			}
			newNode.kind, newNode.name, newNode.check = fullNodeKindRegex, name, fn
		}
	// 常量Node
	default:
		newNode.kind = fullNodeKindConst
	}
	return newNode
}

// Load the checksum function by name.
//
// 根据名称加载校验函数。
func loadCheckFunc(path string) (string, RouterCheckFunc) {
	// invalid path
	// 无效路径
	if len(path) == 0 || (path[0] != ':' && path[0] != '*') {
		return "", nil
	}
	path = path[1:]
	// Intercept parameter name and check function name
	// 截取参数名称和校验函数名称
	name, fname := split2byte(path, '|')
	if len(name) == 0 {
		return "", nil
	}

	// regular
	// If it is the beginning of a regular expression, add the default regular check function name.
	// 正则
	// 如果是正则表达式开头，添加默认正则校验函数名称。
	if fname[0] == '^' {
		fname = "regexp:" + fname
	}

	// Determine if there is ':'
	// 判断是否有':'
	f2name, arg := split2byte(fname, ':')
	// no ':' is a fixed function, return directly
	// 没有':'为固定函数，直接返回
	if len(arg) == 0 {
		return name, GetRouterCheckFunc(fname)
	}

	// There is a ':' variable function to create a checksum function
	// 有':'为变量函数，创建校验函数
	fn := GetRouterNewCheckFunc(f2name)(arg)
	// save the newly created checksum function
	// 保存新建的校验函数
	SetRouterCheckFunc(fname, fn)
	return name, fn
}

// Add a child node to the node.
//
// 给节点添加一个子节点。
func (r *fullNode) InsertNode(path string, nextNode *fullNode) *fullNode {
	if len(path) == 0 {
		return r
	}
	nextNode.path = path
	switch nextNode.kind {
	case radixNodeKindConst:
		for i := range r.Cchildren {
			subStr, find := getSubsetPrefix(path, r.Cchildren[i].path)
			if find {
				if subStr == r.Cchildren[i].path {
					nextTargetKey := strings.TrimPrefix(path, r.Cchildren[i].path)
					return r.Cchildren[i].InsertNode(nextTargetKey, nextNode)
				}

				newNode := r.SplitNode(subStr, r.Cchildren[i].path)
				if newNode == nil {
					panic("Unexpect error on split node")
				}
				return newNode.InsertNode(strings.TrimPrefix(path, subStr), nextNode)
			}
		}
		r.Cchildren = append(r.Cchildren, nextNode)
	case fullNodeKindParam:
		// parameter node
		// 参数节点

		// The path exists to return the old Node
		// 路径存在返回旧Node
		for _, i := range r.Pchildren {
			if i.path == path {
				return i
			}
		}
		r.pnum++
		r.Pchildren = append(r.Pchildren, nextNode)
	case fullNodeKindRegex:
		// parameter check node
		// 参数校验节点
		for _, i := range r.Rchildren {
			if i.path == path {
				return i
			}
		}
		r.pnum++
		r.Rchildren = append(r.Rchildren, nextNode)
	case fullNodeKindValid:
		// wildcard check Node
		// 通配符校验Node
		for _, i := range r.Vchildren {
			if i.path == path {
				return i
			}
		}
		r.Vchildren = append(r.Vchildren, nextNode)
	case fullNodeKindWildcard:
		// Set the wildcard Node data.
		// 设置通配符Node数据。
		r.Wchildren = nextNode
	default:
		panic("Undefined radix node type from router full.")
	}
	return nextNode
}

// Bifurcate the child node whose path is edgeKey, and the fork common prefix path is pathKey
//
// 对指定路径为edgeKey的子节点分叉，分叉公共前缀路径为pathKey
func (r *fullNode) SplitNode(pathKey, edgeKey string) *fullNode {
	for i := range r.Cchildren {
		if r.Cchildren[i].path == edgeKey {
			newNode := &fullNode{path: pathKey}
			newNode.Cchildren = append(newNode.Cchildren, r.Cchildren[i])

			r.Cchildren[i].path = strings.TrimPrefix(edgeKey, pathKey)
			r.Cchildren[i] = newNode
			return newNode
		}
	}
	return nil
}

// Set the tags for the current Node
//
// 给当前Node设置tags
func (r *fullNode) SetTags(args []string) {
	if len(args) == 0 {
		return
	}
	r.tags = make([]string, len(args))
	r.vals = make([]string, len(args))
	// The first parameter name defaults to route
	// 第一个参数名称默认为route
	r.tags[0] = ParamRoute
	r.vals[0] = args[0]
	for i, str := range args[1:] {
		r.tags[i+1], r.vals[i+1] = split2byte(str, '=')
	}
}

// Give the current Node tag to Params
//
// 将当前Node的tags给予Params
func (r *fullNode) AddTagsToParams(p Params) {
	for i := range r.tags {
		p.AddParam(r.tags[i], r.vals[i])
	}
}

// Get the tree of the corresponding method.
//
// Support eudore.RouterAllMethod these methods, weak support will return 405 processing tree.
//
// 获取对应方法的树。
//
// 支持eudore.RouterAllMethod这些方法,弱不支持会返回405处理树。
func (r *RouterFull) getTree(method string) *fullNode {
	switch method {
	case MethodGet:
		return &r.get
	case MethodPost:
		return &r.post
	case MethodDelete:
		return &r.delete
	case MethodPut:
		return &r.put
	case MethodHead:
		return &r.head
	case MethodOptions:
		return &r.options
	case MethodPatch:
		return &r.patch
	default:
		return &r.node405
	}
}

// Recursively add a constant Node with a path of containKey to the current node
//
// targetKey and targetValue are new Node data.
//
// 给当前节点递归添加一个路径为containKey的常量Node
//
// targetKey和targetValue为新Node数据。
func (r *fullNode) recursiveInsertTree(containKey string, targetNode *fullNode) *fullNode {
	for i := range r.Cchildren {
		subStr, find := getSubsetPrefix(containKey, r.Cchildren[i].path)
		if find {
			if subStr == r.Cchildren[i].path {
				nextTargetKey := strings.TrimPrefix(containKey, r.Cchildren[i].path)
				return r.Cchildren[i].recursiveInsertTree(nextTargetKey, targetNode)
			}

			newNode := r.SplitNode(subStr, r.Cchildren[i].path)
			if newNode == nil {
				panic("Unexpect error on split node")
			}
			return newNode.InsertNode(strings.TrimPrefix(containKey, subStr), targetNode)
		}
	}

	return r.InsertNode(containKey, targetNode)
}

func (r *fullNode) recursiveLoopup(searchKey string, params Params) Handler {

	// constant match, return data
	// 常量匹配，返回数据
	if len(searchKey) == 0 && r.handlers != nil {
		r.AddTagsToParams(params)
		return r.handlers
	}

	// Traverse constant Node match
	// 遍历常量Node匹配
	for _, edgeObj := range r.Cchildren {
		if contrainPrefix(searchKey, edgeObj.path) {
			nextSearchKey := searchKey[len(edgeObj.path):]
			if n := edgeObj.recursiveLoopup(nextSearchKey, params); n != nil {
				return n
			}
			break
		}
	}

	// parameter matching
	// Check if there is a parameter match
	// 参数匹配
	// 检测是否存在参数匹配
	if r.pnum != 0 && len(searchKey) > 0 {
		pos := strings.IndexByte(searchKey, '/')
		if pos == -1 {
			pos = len(searchKey)
		}
		currentKey, nextSearchKey := searchKey[:pos], searchKey[pos:]

		// check parameter matching
		// 校验参数匹配
		for _, edgeObj := range r.Rchildren {
			if edgeObj.check(currentKey) {
				if n := edgeObj.recursiveLoopup(nextSearchKey, params); n != nil {
					params.AddParam(edgeObj.name, currentKey)
					return n
				}
			}
		}

		// 参数匹配
		// 变量Node依次匹配是否满足
		for _, edgeObj := range r.Pchildren {
			if n := edgeObj.recursiveLoopup(nextSearchKey, params); n != nil {
				params.AddParam(edgeObj.name, currentKey)
				return n
			}
		}
	}

	// wildcard verification match
	// If the current Node has a wildcard processing method that directly matches, the result is returned.
	// 通配符校验匹配
	// 若当前Node有通配符处理方法直接匹配，返回结果。
	for _, edgeObj := range r.Vchildren {
		if edgeObj.check(searchKey) {
			edgeObj.AddTagsToParams(params)
			params.AddParam(edgeObj.name, searchKey)
			return edgeObj.handlers
		}
	}

	// If the current Node has a wildcard processing method that directly matches, the result is returned.
	// 若当前Node有通配符处理方法直接匹配，返回结果。
	if r.Wchildren != nil {
		r.Wchildren.AddTagsToParams(params)
		params.AddParam(r.Wchildren.name, searchKey)
		return r.Wchildren.handlers
	}

	// can't match, return nil
	// 无法匹配，返回空
	return nil
}

var (
	globalRouterCheckFunc    = make(map[string]RouterCheckFunc)
	globalRouterNewCheckFunc = make(map[string]RouterNewCheckFunc)
)

func init() {
	// RouterCheckFunc
	globalRouterCheckFunc["isnum"] = routerCheckFuncIsnm
	globalRouterCheckFunc["nozero"] = routerCheckFuncNozero
	// RouterNewCheckFunc
	globalRouterNewCheckFunc["min"] = routerNewCheckFuncMin
	globalRouterNewCheckFunc["max"] = routerNewCheckFuncMax
	globalRouterNewCheckFunc["regexp"] = routerNewCheckFuncRegexp
}

// SetRouterCheckFunc 保存一个RouterCheckFunc函数，用于参数校验使用。
func SetRouterCheckFunc(name string, fn RouterCheckFunc) {
	globalRouterCheckFunc[name] = fn
}

// GetRouterCheckFunc 获得一个RouterCheckFunc函数
func GetRouterCheckFunc(name string) RouterCheckFunc {
	return globalRouterCheckFunc[name]
}

func routerCheckFuncIsnm(arg string) bool {
	_, err := strconv.Atoi(arg)
	return err == nil
}

func routerCheckFuncNozero(arg string) bool {
	return len(arg) != 0
}

// SetRouterNewCheckFunc 保存一个RouterNewCheckFunc函数，用于参数动态校验使用。
func SetRouterNewCheckFunc(name string, fn RouterNewCheckFunc) {
	globalRouterNewCheckFunc[name] = fn
}

// GetRouterNewCheckFunc 获得一个RouterNewCheckFunc函数
func GetRouterNewCheckFunc(name string) RouterNewCheckFunc {
	return globalRouterNewCheckFunc[name]
}

func routerNewCheckFuncMin(str string) RouterCheckFunc {
	n, err := strconv.Atoi(str)
	if err != nil {
		return nil
	}
	return func(arg string) bool {
		num, err := strconv.Atoi(arg)
		if err != nil {
			return false
		}
		return n <= num
	}
}
func routerNewCheckFuncMax(str string) RouterCheckFunc {
	n, err := strconv.Atoi(str)
	if err != nil {
		return nil
	}
	return func(arg string) bool {
		num, err := strconv.Atoi(arg)
		if err != nil {
			return false
		}
		return n >= num
	}
}

func routerNewCheckFuncRegexp(str string) RouterCheckFunc {
	// 创建正则表达式
	re, err := regexp.Compile(str)
	if err != nil {
		return nil
	}
	// 返回正则匹配校验函数
	return func(arg string) bool {
		return re.MatchString(arg)
	}
}
