//go:build js && wasm
// +build js,wasm

// Package jscache caches expensive JavaScript results, like string encoding
package jscache

import (
	"github.com/hack-pad/safejs"
)

type cacher struct {
	cache map[string]safejs.Value
}

func (c *cacher) value(key string, valueFn func() safejs.Value) safejs.Value {
	if val, ok := c.cache[key]; ok {
		return val
	}
	if c.cache == nil {
		c.cache = make(map[string]safejs.Value)
	}
	val := valueFn()
	c.cache[key] = val
	return val
}
