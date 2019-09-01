package common

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPipe(t *testing.T) {

	left, right := net.Pipe()

	go func() {

		time.Sleep(time.Second)

		err := left.Close()
		assert.NoError(t, err)
	}()

	var buf = new(bytes.Buffer)

	err := connectStream(buf, right)
	assert.NoError(t, err)
}
