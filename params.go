package erouter

import "sync"

// 数组参数的复用池。
var paramArrayPool = sync.Pool{
	New: func() interface{} {
		return &ParamsArray{}
	},
}

// ParamsArray 默认参数实现，使用数组保存键值对。
type ParamsArray struct {
	Keys []string
	Vals []string
}

// Reset 清空数组，配合sync.Pool减少GC。
func (p *ParamsArray) Reset() {
	p.Keys = p.Keys[0:0]
	p.Vals = p.Vals[0:0]
}

// GetParam 读取参数的值，如果不存在返回空字符串。
func (p *ParamsArray) GetParam(key string) string {
	for i, str := range p.Keys {
		if str == key {
			return p.Vals[i]
		}
	}
	return ""
}

// AddParam 追加一个参数的值。
func (p *ParamsArray) AddParam(key string, val string) {
	p.Keys = append(p.Keys, key)
	p.Vals = append(p.Vals, val)
}

// SetParam 设置一个参数的值。
func (p *ParamsArray) SetParam(key string, val string) {
	for i, str := range p.Keys {
		if str == key {
			p.Vals[i] = val
			return
		}
	}
	p.AddParam(key, val)
}
