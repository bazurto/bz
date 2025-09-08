package model_test

import (
	"strings"
	"testing"

	"github.com/bazurto/bz/lib/model"
	"github.com/stretchr/testify/assert"
)

func TestNewMap(t *testing.T) {
	m := model.NewLinkedMap[string, string]()
	m.Put("a", "A")
	m.Put("b", "B")
	m.Put("c", "C")
	m.Put("d", "D")

	//
	var keys []string
	var vals []string
	for k, v := range m.All() {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	assert.Equal(t, "a,b,c,d", strings.Join(keys, ","))
	assert.Equal(t, "A,B,C,D", strings.Join(vals, ","))

	//
	m.Remove("b")
	keys = nil
	vals = nil
	for k, v := range m.All() {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	assert.Equal(t, "a,c,d", strings.Join(keys, ","))
	assert.Equal(t, "A,C,D", strings.Join(vals, ","))

	//
	m.Remove("a")
	keys = nil
	vals = nil
	for k, v := range m.All() {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	assert.Equal(t, "c,d", strings.Join(keys, ","))
	assert.Equal(t, "C,D", strings.Join(vals, ","))

	//
	m.Remove("d")
	keys = nil
	vals = nil
	for k, v := range m.All() {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	assert.Equal(t, "c", strings.Join(keys, ","))
	assert.Equal(t, "C", strings.Join(vals, ","))

	//
	m.Remove("a")
	keys = nil
	vals = nil
	for k, v := range m.All() {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	assert.Equal(t, "c", strings.Join(keys, ","))
	assert.Equal(t, "C", strings.Join(vals, ","))

	//
	m.Remove("c")
	keys = nil
	vals = nil
	for k, v := range m.All() {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	assert.Equal(t, "", strings.Join(keys, ","))
	assert.Equal(t, "", strings.Join(vals, ","))

	//
	m.Put("a", "a")
	m.Put("b", "b")
	m.Put("c", "C")
	keys = nil
	vals = nil
	for k, v := range m.All() {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	assert.Equal(t, "a,b,c", strings.Join(keys, ","))
	assert.Equal(t, "A,B,C", strings.Join(vals, ","))

	//
	m.Put("a", "X")
	m.Put("b", "Y")
	m.Put("c", "Z")
	keys = nil
	vals = nil
	for k, v := range m.All() {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	assert.Equal(t, "a,b,c", strings.Join(keys, ","))
	assert.Equal(t, "X,y,Z", strings.Join(vals, ","))

	assert.Equal(t, "X", m.GetVal("a"))
	assert.Equal(t, "Y", m.GetVal("b"))
	assert.Equal(t, "Z", m.GetVal("c"))

	assert.Equal(t, "X", func() string { v, _ := m.Get("a"); return v }())
	assert.Equal(t, "Y", func() string { v, _ := m.Get("b"); return v }())
	assert.Equal(t, "Z", func() string { v, _ := m.Get("c"); return v }())
}
