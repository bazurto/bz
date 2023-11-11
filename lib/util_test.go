package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonDecode(t *testing.T) {
	m := make(map[string]string)
	err := jsonDecode(`{"a":"b"}`, m)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(m))
	assert.Equal(t, "b", m["a"])
}
