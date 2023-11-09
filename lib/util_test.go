package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonDecode(t *testing.T) {
	mPtr, err := jsonDecode(`{"a":"b"}`, make(map[string]string))
	m := *mPtr
	assert.Nil(t, err)
	assert.Equal(t, 1, len(m))
	assert.Equal(t, "b", m["a"])
}
