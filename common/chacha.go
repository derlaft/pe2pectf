package common

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"io"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/chacha20poly1305"
)

const streamMaxMessage = 256

type CryptoReadWriter struct {
	Stream  SimpleConn
	ChaCha  cipher.AEAD
	OldData []byte
}

type MessageHeader struct {
	Nonce [chacha20poly1305.NonceSize]byte
	Len   uint32 // length of encoded message
}

func NewCryptoReadWriter(conn SimpleConn, key []byte) (*CryptoReadWriter, error) {

	chacha, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}

	return &CryptoReadWriter{
		ChaCha: chacha,
		Stream: conn,
	}, nil
}

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

	// decode the header
	var header MessageHeader
	err = binary.Read(teeReader, binary.BigEndian, &header)
	if err == io.EOF {
		return 0, err
	} else if err != nil {
		return 0, errors.Wrap(err, "Failed to read next message header")
	}

	// read the encrypted message
	var messageBuf = make([]byte, int(header.Len)+cr.ChaCha.Overhead())
	n, err = cr.Stream.Read(messageBuf)
	if err != nil {
		return 0, err
	}

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

	// if OldData is used, it will be read next time

	return len(p), nil
}

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

		n += toSend
		p = p[toSend:]
	}

	return n, nil
}

func (cr *CryptoReadWriter) SetReadDeadline(t time.Time) error {
	return cr.Stream.SetReadDeadline(t)
}

func (cr *CryptoReadWriter) SetWriteDeadline(t time.Time) error {
	return cr.Stream.SetWriteDeadline(t)
}

func (cr *CryptoReadWriter) SetDeadline(t time.Time) error {
	return cr.Stream.SetDeadline(t)
}
