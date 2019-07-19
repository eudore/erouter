package erouter

import (
	"strings"
)

type (
	// 存储中间件信息的基数树。
	//
	// 用于内存存储路由器中间件注册信息，并根据注册路由返回对应的中间件。
	middTree struct {
		root middNode
	}
	middNode struct {
		path     string
		children []*middNode
		key      string
		val      []Middleware
	}
)

func (r *middNode) Insert(key string, val []Middleware) {
	r.recursiveInsertTree(key, key, val)
}

//Lookup: Find if seachKey exist in current radix tree and return its value
func (r *middNode) Lookup(searchKey string) []Middleware {
	searchKey = strings.Split(searchKey, " ")[0]
	if searchKey[len(searchKey)-1] == '*' {
		searchKey = searchKey[:len(searchKey)-1]
	}
	return r.recursiveLoopup(searchKey)
}

// 新增Node
func (r *middNode) InsertNode(path, key string, value []Middleware) {
	if len(path) == 0 {
		// 路径空就设置当前node的值
		r.key = key
		r.val = combineMiddlewares(r.val, value)
	} else {
		// 否则新增node
		r.children = append(r.children, &middNode{path: path, key: key, val: value})
	}
}

// 对指定路径为edgeKey的Node分叉，公共前缀路径为pathKey
func (r *middNode) SplitNode(pathKey, edgeKey string) *middNode {
	for i := range r.children {
		if r.children[i].path == edgeKey {
			newNode := &middNode{path: pathKey}
			newNode.children = append(newNode.children, &middNode{
				path:     strings.TrimPrefix(edgeKey, pathKey),
				key:      r.children[i].key,
				val:      r.children[i].val,
				children: r.children[i].children,
			})
			r.children[i] = newNode
			return newNode
		}
	}
	return nil
}

// 给currentNode递归添加，路径为containKey的Node。
func (r *middNode) recursiveInsertTree(containKey string, targetKey string, targetValue []Middleware) {
	for i := range r.children {
		subStr, find := getSubsetPrefix(containKey, r.children[i].path)
		if find {
			if subStr == r.children[i].path {
				nextTargetKey := strings.TrimPrefix(containKey, r.children[i].path)
				r.children[i].recursiveInsertTree(nextTargetKey, targetKey, targetValue)
			} else {
				newNode := r.SplitNode(subStr, r.children[i].path)
				if newNode == nil {
					panic("Unexpect error on split node")
				}

				newNode.InsertNode(strings.TrimPrefix(containKey, subStr), targetKey, targetValue)
			}
			return
		}
	}
	r.InsertNode(containKey, targetKey, targetValue)
}

// 递归获得r路径为searchKey的Node数据。
func (r *middNode) recursiveLoopup(searchKey string) []Middleware {
	if len(searchKey) == 0 {
		return r.val
	}

	for _, edgeObj := range r.children {
		// 寻找相同前缀node
		if contrainPrefix(searchKey, edgeObj.path) {
			nextSearchKey := strings.TrimPrefix(searchKey, edgeObj.path)
			return append(r.val, edgeObj.recursiveLoopup(nextSearchKey)...)
		}
	}

	if len(r.key) == 0 || r.key[len(r.key)-1] == '/' {
		return r.val
	}

	return nil
}

func combineMiddlewares(hs1, hs2 []Middleware) []Middleware {
	// if nil
	if len(hs1) == 0 {
		return hs2
	}
	if len(hs2) == 0 {
		return hs1
	}
	// combine
	const abortIndex int8 = 63
	finalSize := len(hs1) + len(hs2)
	if finalSize >= int(abortIndex) {
		panic("too many handlers")
	}
	hs := make([]Middleware, finalSize)
	copy(hs, hs1)
	copy(hs[len(hs1):], hs2)
	return hs
}
