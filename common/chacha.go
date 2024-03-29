package common

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
	"golang.org/x/crypto/chacha20poly1305"
)

const streamMaxMessage = 256

// CryptoReadWriter implements an encrypter read-writer over an existing stream
type CryptoReadWriter struct {
	Stream  io.ReadWriteCloser
	ChaCha  cipher.AEAD
	OldData []byte
}

// MessageHeader is sent before each message
type MessageHeader struct {
	Nonce [chacha20poly1305.NonceSize]byte
	Len   uint32 // length of encoded message
}

// NewCryptoReadWriter constructor that requires a key and a stream
func NewCryptoReadWriter(conn io.ReadWriteCloser, key []byte) (*CryptoReadWriter, error) {

	chacha, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}

	return &CryptoReadWriter{
		ChaCha: chacha,
		Stream: conn,
	}, nil
}

// Read next portion of data.
// If there is already some buffered data, return it
// If there is none, wait for the next header, decode it, read payload && return it
func (cr *CryptoReadWriter) Read(p []byte) (n int, err error) {

	// flush old buffer if it's present
	if len(cr.OldData) > 0 {
		n = len(cr.OldData)
		if n > len(p) {
			n = len(p)
		}
		copy(p, cr.OldData[:n])
		cr.OldData = cr.OldData[n:]
		return
	}

	// we need to both read the header into buf for verification and into binary decoder
	var (
		headBuf   = &bytes.Buffer{}
		teeReader = io.TeeReader(cr.Stream, headBuf)
	)

	log.Debugf("waiting for header")

	// decode the header
	var header MessageHeader
	err = binary.Read(teeReader, binary.BigEndian, &header)
	if err == io.EOF {
		return 0, err
	} else if err != nil {
		return 0, errors.Wrap(err, "Failed to read next message header")
	}

	log.Debugf("got message header, waiting for %v bytes of sealed payload", int(header.Len)+cr.ChaCha.Overhead())

	// read the encrypted message
	var messageBuf = make([]byte, int(header.Len)+cr.ChaCha.Overhead())
	n, err = cr.Stream.Read(messageBuf)
	if err != nil {
		return 0, err
	}

	log.Debugf("got message body")

	// buffer for the plaintext message
	var buf = p

	// if the buffer is too short, use an external one
	if len(buf) < int(header.Len) {
		cr.OldData = make([]byte, int(header.Len))
		buf = cr.OldData
	}

	// then - read the stream using chacha
	p, err = cr.ChaCha.Open(buf[:0], header.Nonce[:], messageBuf[:n], headBuf.Bytes())
	if err != nil {
		return 0, err
	}

	log.Debugf("unsealed payload")

	// if OldData is used, it will be read next time

	return len(p), nil
}

// Write back to the stream
func (cr *CryptoReadWriter) Write(p []byte) (n int, err error) {

	for len(p) > 0 {

		// create a new message header
		var header MessageHeader
		_, err = rand.Read(header.Nonce[:])
		if err != nil {
			return
		}

		// fill in size info
		var toSend = len(p)
		if toSend > streamMaxMessage {
			toSend = streamMaxMessage
		}
		header.Len = uint32(toSend)

		var (
			headerBuf    = &bytes.Buffer{}
			headerWriter = io.MultiWriter(headerBuf, cr.Stream)
		)

		// send it
		err = binary.Write(headerWriter, binary.BigEndian, &header)
		if err != nil {
			return
		}

		// sign payload && send it
		payload := cr.ChaCha.Seal(nil, header.Nonce[:], p[:toSend], headerBuf.Bytes())
		_, err = cr.Stream.Write(payload)
		if err != nil {
			return
		}

		log.Debugf("wrote %v bytes of sealed payload", len(payload))

		n += toSend
		p = p[toSend:]
	}

	return n, nil
}

func (cr *CryptoReadWriter) Close() error {
	return cr.Stream.Close()
}
