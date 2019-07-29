package common

import (
	"io"
	"net"
	"sync"
	"time"
)

const (
	CopyBufSize = 4096
	ReadTimeout = time.Millisecond * 100
)

type ReaderDeadline interface {
	io.Reader
	SetReadDeadline(t time.Time) error
}

type SimpleConn interface {
	io.ReadWriter
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	SetDeadline(t time.Time) error
}

func copyRealtime(dest io.Writer, source ReaderDeadline) error {

	var (
		buf       = make([]byte, 2048)
		toWrite   int
		lastError error
	)

	for {

		// set timeout
		err := source.SetReadDeadline(time.Now().Add(ReadTimeout))
		if err != nil {
			return err
		}

		// do the read
		toWrite, err = source.Read(buf)
		if err == io.EOF {
			break
		} else if err != nil {

			// check for timeout
			if value, ok := err.(net.Error); ok && value.Timeout() {
				// @TODO: write remaining
				continue
			}

			// no timeout!
			lastError = err
			break
		}

		// do the casual write (no retries or timeouts)
		attemptedWrite, err := dest.Write(buf[:toWrite])
		if err != nil {
			lastError = err
			toWrite -= attemptedWrite
			break
		}
		toWrite = 0

	}

	// last attempt to write
	if toWrite > 0 {
		_, err := dest.Write(buf[:toWrite])
		if err != nil {
			return err
		}
	}

	return lastError

}

func connectStream(pipe, stream SimpleConn) error {

	// @TODO: use context?

	var (
		errTo   error
		errFrom error
		wg      sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		// @TODO: check error
		errTo = copyRealtime(pipe, stream)
		wg.Done()
	}()

	go func() {
		errFrom = copyRealtime(pipe, stream)
		wg.Done()
	}()

	wg.Wait()

	if errTo != nil {
		return errTo
	}

	if errFrom != nil {
		return errFrom
	}

	return nil
}
