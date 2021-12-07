package prometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImportantFunction(t *testing.T) {
	//assert = assert.New(t)
	v := ImportantFunction()
	assert.Equal(t, v, 3)
}
