package common

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/chacha20poly1305"
)

func TestReaderWriter(t *testing.T) {

	for _, sampleString := range []string{
		"",
		"sample test string",
		"1",
		"💩",
		"sample test stringAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAheh",
	} {

		var (
			input   = bytes.NewBuffer([]byte(sampleString))
			network = &bytes.Buffer{}
			output  = &bytes.Buffer{}
		)

		var key = make([]byte, chacha20poly1305.KeySize)
		_, err := rand.Read(key)
		assert.NoError(t, err)

		writer, err := NewCryptoWriter(network, key)
		assert.NoError(t, err)

		reader, err := NewCryptoReader(network, key)
		assert.NoError(t, err)

		_, err = io.Copy(writer, input)
		assert.NoError(t, err)

		_, err = io.Copy(output, reader)
		assert.NoError(t, err)

		assert.Equal(t, sampleString, output.String())

	}
}
