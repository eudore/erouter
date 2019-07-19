# RouterRadix 实现

RouterRadix由RouterCore和RouterMethod组合而成，RouterCore是路由器的核心，需要实现路由注册、中间件添加和请求匹配并处理，而RouterMethod只是一层保证，用来方便使用。

## Radix基础

RouterRadix是基于基数树(Radix)实现，压缩前缀树，是一种更节省空间的Trie（前缀树）。对于基数树的每个节点，如果该节点是唯一的子树的话，就和父节点合并。

如果依次向树添加test、team、api，那么过程应该如下，test和team具有公共前缀te，te是st和am公共前缀。

添加test，只有唯一子节点。

```
test
```

添加team，team和test具有相同前缀te，那么提取te为公共前缀，然后子节点有两个，分叉成st和am。

```
te
--st
--am
```

添加api，api和te没有相同前缀（首字母不同），给根节点添加一个新的子节点api。

```
te
--st
--am
api
```

如果需要查找应该字符串，匹配字符串的和节点字符串是否为查找的字符串前缀，是才表示匹配，然后截取剩余字符串，进行下一步匹配。

如果查找append，根节点只有te、app、interface三个子节点，匹配命中app，剩余未匹配的是le。

然后使用a的子节点le、end，两个子节点匹配恰好le。

```
te
--st
----22
--am
app
---le
---end
interface

```

插入和查找的radix实现：

```golang
package main

import (
	"strings"
	"testing"
	"fmt"
	"github.com/kr/pretty"
)

func main() {
	tree := NewRadixTree()
	tree.Insert("test", 1)
	// fmt.Printf("%# v\n", pretty.Formatter(tree))
	tree.Insert("test22", 1)
	// fmt.Printf("%# v\n", pretty.Formatter(tree))
	tree.Insert("team", 3)
	tree.Insert("apple", 4)
	tree.Insert("append", 12)
	tree.Insert("app", 5)
	tree.Insert("append", 6)
	tree.Insert("interface", 7)
	fmt.Printf("%# v\n", pretty.Formatter(tree))
	t.Log(tree.Lookup("append"))
}

type (
	radixTree struct {
		root radixNode
	}
	radixNode struct {
		path     string
		children []*radixNode
		key      string
		val      interface{}
	}
)

func NewRadixTree() *radixTree {
	return &radixTree{radixNode{}}
}

// 新增Node
func (r *radixNode) InsertNode(path, key string, value interface{}) {
	if len(path) == 0 {
		// 路径空就设置当前node的值
		r.key = key
		r.val = value
	} else {
		// 否则新增子node
		r.children = append(r.children, &radixNode{path: path, key: key, val: value})
	}
}

// 对指定路径为edgeKey的Node分叉，公共前缀路径为pathKey
func (r *radixNode) SplitNode(pathKey, edgeKey string) *radixNode {
	for i, _ := range r.children {
		// 找到路径为edgeKey路径的Node，然后分叉
		if r.children[i].path == edgeKey {
			// 创建新的分叉Node，路径为公共前缀路径pathKey
			newNode := &radixNode{path: pathKey}
			// 将原来edgeKey的数据移动到新的分叉Node之下
			// 直接新增Node，原Node数据仅改变路径为截取后的后段路径
			newNode.children = append(newNode.children, &radixNode{
				// 截取路径
				path: strings.TrimPrefix(edgeKey, pathKey),
				// 复制数据
				key:      r.children[i].key,
				val:      r.children[i].val,
				children: r.children[i].children,
			})
			// 设置radixNode的child[i]的Node为分叉Node
			// 原理路径Node的数据移到到了分叉Node的child里面，原Node对象GC释放。
			r.children[i] = newNode
			// 返回分叉新创建的Node
			return newNode
		}
	}
	return nil
}

func (t *radixTree) Insert(key string, val interface{}) {
	t.recursiveInsertTree(&t.root, key, key, val)
}

// 给currentNode递归添加，路径为containKey的Node
//
// targetKey和targetValue为新Node数据。
func (t *radixTree) recursiveInsertTree(currentNode *radixNode, containKey string, targetKey string, targetValue interface{}) {
	for i, _ := range currentNode.children {
		// 检查当前遍历的Node和插入路径是否有公共路径
		// subStr是两者的公共路径，find表示是否有
		subStr, find := getSubsetPrefix(containKey, currentNode.children[i].path)
		if find {
			// 如果child路径等于公共最大路径，则该node添加child
			// child的路径为插入路径先过滤公共路径的后面部分。
			if subStr == currentNode.children[i].path {
				nextTargetKey := strings.TrimPrefix(containKey, currentNode.children[i].path)
				// 当前node新增子Node可能原本有多个child，所以需要递归添加
				t.recursiveInsertTree(currentNode.children[i], nextTargetKey, targetKey, targetValue)
			} else {
				// 如果公共路径不等于当前node的路径
				// 则将currentNode.children[i]路径分叉
				// 分叉后的就拥有了公共路径，然后添加新Node
				newNode := currentNode.SplitNode(subStr, currentNode.children[i].path)
				if newNode == nil {
					panic("Unexpect error on split node")
				}
				// 添加新的node
				// 分叉后树一定只有一个没有相同路径的child，所以直接添加node
				newNode.InsertNode(strings.TrimPrefix(containKey, subStr), targetKey, targetValue)
			}
			return
		}
	}
	// 没有相同前缀路径存在，直接添加为child
	currentNode.InsertNode(containKey, targetKey, targetValue)
}

//Lookup: Find if seachKey exist in current radix tree and return its value
func (t *radixTree) Lookup(searchKey string) (interface{}, bool) {
	return t.recursiveLoopup(&t.root, searchKey)
}

// 递归获得searchNode路径为searchKey的Node数据。
func (t *radixTree) recursiveLoopup(searchNode *radixNode, searchKey string) (interface{}, bool) {
	// 匹配node，返回数据
	if len(searchKey) == 0 {
		// 如果没有添加节点本身，那么key就是空字符串，表示节点数据不存在。
		return searchNode.val, searchNode.key != ""
	}

	for _, edgeObj := range searchNode.children {
		// 寻找相同前缀node
		if contrainPrefix(searchKey, edgeObj.path) {
			// 截取为匹配的路径
			nextSearchKey := strings.TrimPrefix(searchKey, edgeObj.path)
			// 然后当前Node递归判断
			return t.recursiveLoopup(edgeObj, nextSearchKey)
		}
	}

	return nil, false
}

// 判断字符串str1的前缀是否是str2
func contrainPrefix(str1, str2 string) bool {
	if sub, find := getSubsetPrefix(str1, str2); find {
		return sub == str2
	}
	return false
}

// 获取两个字符串的最大公共前缀，返回最大公共前缀和是否拥有最大公共前缀
func getSubsetPrefix(str1, str2 string) (string, bool) {
	findSubset := false
	for i := 0; i < len(str1) && i < len(str2); i++ {
		if str1[i] != str2[i] {
			retStr := str1[:i]
			return retStr, findSubset
		}
		findSubset = true
	}

	if len(str1) > len(str2) {
		return str2, findSubset
	} else if len(str1) == len(str2) {
		//fix "" not a subset of ""
		return str1, str1 == str2
	}

	return str1, findSubset
}
```

## RadixRouter

RadixRouter基于基数树实现，使用节点按类型分类处理，**实现匹配优先顺序、易扩展、低代码复杂度的特点**。

RouterRadix代码复杂度均低于15，而erouter库中只存在两处代码复杂度大于15(17,18)，由于RouterFull处理节点类型增加两种导致的，[代码复杂度](https://goreportcard.com/report/github.com/eudore/erouter#gocyclo)

Radix树只是基本的字符串匹配，但是Radix路由具有常量、变量、通配符三种匹配节点,因此将三种分开处理。

```golang
// radix节点的定义
type radixNode struct {
	// 基本信息
	kind uint8
	path string
	name string
	// 每次类型子节点
	Cchildren []*radixNode
	Pchildren []*radixNode
	Wchildren *radixNode
	// 当前节点的数据
	tags     []string
	vals     []string
	handlers Handler
}
```

在查找时先匹配全部常量子节点，没有就使用变量子节点，uri本段就是变量内容，剩余进行递归匹配，如果变量子节点不匹配，就检查通配符节点，如果存在就是直接匹配通配符。

因此具有路由具有严格的匹配优先顺序，一定是先常量再变量最后通配符，由匹配函数里面代码段的位置决定了顺序。

如果六条路由是`/*`，最先注册的，但是api是常量更有效，就会先检查是否是api，不是才会使用通配符，

而`/api/:user`和`/api/:user/info`两条，会进一步检查是否是info，如果是`/api/eudore/list`只会匹配到`/api/*`。

```
/*
/api/v1
/api/*
/api/user
/api/:user
/api/:user/info
```

`func getSpiltPath(key string) []string`将字符串按Node类型切割。

例如`/api/:get/*`中`:get`和`*`明显是变量和通配符节点。所以两种前后需要切分开来，结果为`[/api/ :get / *]`,`/api/`增加变量子节点`:get`，依次添加完成树。

字符串路径切割例子：

```
/				[/]
/api/note/		[/api/note/]
//api/*			[/api/ *]
//api/*name		[/api/ *name]
/api/get/		[/api/get/]
/api/get		[/api/get]
/api/:get		[/api/ :get]
/api/:get/*		[/api/ :get / *]
/api/:name/info/*		[/api/ :name /info/ *]
/api/:name|^\\d+$/info	[/api/ :name|^\d+$ /info]
/api/*|^0/api\\S+$		[/api/ *|^0 /api\S+$]
/api/*|^\\$\\d+$		[/api/ *|^\$\d+$]
```

### Radix路由添加

insertRoute先根据方法选择对应的树，然后依次向树下节点的路径，最后的节点就是树末，设置路由的处理函数和属性。这里的字符串切分正则时，如果正则规则有空格和斜杠会导致错误。

InsertNode中常量节点添加就使用了基数树的添加分叉，其他类型直接添加，其中处理了相同路由重复注册。

```golang
// 添加一个新的路由节点。
//
// 如果方法不支持则不会添加，请求该路径会响应405.
//
// 将路径按节点类型切割，每段路径即为一种类型的节点，然后依次向树追加，然后给最后的节点设置数据。
//
// 路径切割见getSpiltPath函数，当前未完善，处理正则可能异常。
func (r *RouterRadix) insertRoute(method, key string, val Handler) {
	var currentNode *radixNode = r.getTree(method)
	if currentNode == &r.node405 {
		return
	}

	// 创建节点
	args := strings.Split(key, " ")
	for _, path := range getSpiltPath(args[0]) {
		currentNode = currentNode.InsertNode(path, newRadixNode(path))
	}

	currentNode.handlers = val
	currentNode.SetTags(args)
}

// 给当前节点路径下添加一个子节点。
//
// 如果新节点类型是常量节点，寻找是否存在相同前缀路径的结点，
// 如果存在路径为公共前缀的结点，直接添加新结点为匹配前缀结点的子节点；
// 如果只是两结点只是拥有公共前缀，则先分叉然后添加子节点。
//
// 如果新节点类型是参数结点，会检测当前参数是否存在，存在返回已处在的节点。
//
// 如果新节点类型是通配符结点，直接设置为当前节点的通配符处理节点。
func (r *radixNode) InsertNode(path string, nextNode *radixNode) *radixNode {
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
	case radixNodeKindParam:
		for _, i := range r.Pchildren {
			if i.path == path {
				return i
			}
		}
		r.Pchildren = append(r.Pchildren, nextNode)
	case radixNodeKindWildcard:
		r.Wchildren = nextNode
	default:
		panic("Undefined radix node type")
	}
	return nextNode
}
```

### Radix路由查找

Match是匹配方法，使用方法对应的树进行查找，如果405就方法的405树，只有405的结果，如果其他方法树查找的结果是空，就使用404处理。

recursiveLoopup是具体查找的过程，整个函数分为相同逻辑的五段。

1、检查是否的当前节点
2、匹配全部常量子节点
3、匹配全部变量子节点
4、检查是否存在通配符节点
5、返回空

```golang
// 匹配一个请求，如果方法不不允许直接返回node405，未匹配返回node404。
func (r *RouterRadix) Match(method, path string, params Params) Handler {
	if n := r.getTree(method).recursiveLoopup(path, params); n != nil {
		return n
	}

	// 处理404
	r.node404.AddTagsToParams(params)
	return r.node404.handlers
}

// 按照顺序匹配一个路径。
//
// 依次检查常量节点、参数节点、通配符节点，如果有一个匹配就直接返回。
func (r *radixNode) recursiveLoopup(searchKey string, params Params) Handler {
	// 如果路径为空，当前节点就是需要匹配的节点，直接返回。
	if len(searchKey) == 0 && r.handlers != nil {
		r.AddTagsToParams(params)
		return r.handlers
	}

	// 遍历常量Node匹配，寻找具有相同前缀的那个节点
	for _, edgeObj := range r.Cchildren {
		if contrainPrefix(searchKey, edgeObj.path) {
			nextSearchKey := searchKey[len(edgeObj.path):]
			if n := edgeObj.recursiveLoopup(nextSearchKey, params); n != nil {
				return n
			}
			// TODO: 待优化测试，只有一个相同前缀，当前应该直接退出遍历
			break
		}
	}

	if len(r.Pchildren) > 0 && len(searchKey) > 0 {
		pos := strings.IndexByte(searchKey, '/')
		if pos == -1 {
			pos = len(searchKey)
		}
		nextSearchKey := searchKey[pos:]

		// Whether the variable Node matches in sequence is satisfied
		// 遍历参数节点是否后续匹配
		for _, edgeObj := range r.Pchildren {
			if n := edgeObj.recursiveLoopup(nextSearchKey, params); n != nil {
				params.AddParam(edgeObj.name, searchKey[:pos])
				return n
			}
		}
	}

	// If the current Node has a wildcard processing method that directly matches, the result is returned.
	// 若当前节点有通配符处理方法直接匹配，返回结果。
	if r.Wchildren != nil {
		r.Wchildren.AddTagsToParams(params)
		params.AddParam(r.Wchildren.name, searchKey)
		return r.Wchildren.handlers
	}

	// can't match, return nil
	// 无法匹配，返回空
	return nil
}
```

## RouterMethod

RouterMethodStd是默认的路由方法处理，实现Group的组路由特性，

Group会保存当前组的路径和参数，然后再AddHandler时添加进去。

```golang
// Group 返回一个组路由方法。
//
// 如果路径是'/*'或'/'结尾，则移除后缀。
func (m *RouterMethodStd) Group(path string) RouterMethod {
	// 将路径前缀和路径参数分割出来
	args := strings.Split(path, " ")
	prefix := args[0]
	tags := path[len(prefix):]

	// 如果路径是'/*'或'/'结尾，则移除后缀。
	// '/*'为路由结尾，不可为路由前缀
	// '/'不可为路由前缀，会导致出现'//'
	if len(prefix) > 0 && prefix[len(prefix)-1] == '*' {
		prefix = prefix[:len(prefix)-1]
	}
	if len(prefix) > 0 && prefix[len(prefix)-1] == '/' {
		prefix = prefix[:len(prefix)-1]
	}

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
```

AddMiddleware、NotFound和Any就是封装了一层RouterCore，使用RESTful风格和组路由。

```golang
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
```

# RouterFull

参考RouterRadix，额外添加了两种Node类型，从三种扩展到五种类型。

# RouterHost

使用路由注册的host参数和请求时Host来匹配对应的子路由器处理。

# Middleware

Middleware是一个装饰器模式的处理函数，`type Middleware func(Handler) Handler`，使用的时传入的Handler为下一个Handler，最后返回一个新的Handler，通过一层层的闭包返回一个新的Handler，最后注册到路由器中使用。

例如一个简单的日志中间件，先输出请求信息，然后调用下一个处理者，也可以再日志输出前方调用下一个处理。

```golang
router.AddMiddleware("ANY", "", func(h erouter.Handler) erouter.Handler {
	return func(w http.ResponseWriter, r *http.Request, p erouter.Params) {
		log.Printf("%s %s route: %s", r.Method, r.URL.Path, p.GetParam("route"))
		h(w, r, p)
	}
})
```