package common

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
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

		var rwc = struct {
			io.ReadWriter
			io.Closer
		}{
			ReadWriter: network,
			Closer:     ioutil.NopCloser(network),
		}

		var key = make([]byte, chacha20poly1305.KeySize)
		_, err := rand.Read(key)
		assert.NoError(t, err)

		readwriter, err := NewCryptoReadWriter(rwc, key)
		assert.NoError(t, err)

		_, err = io.Copy(readwriter, input)
		assert.NoError(t, err)

		_, err = io.Copy(output, readwriter)
		assert.NoError(t, err)

		assert.Equal(t, sampleString, output.String())

	}
}
