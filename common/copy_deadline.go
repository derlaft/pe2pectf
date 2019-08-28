package common

import (
	"io"
	"sync"
	"time"
)

const (
	CopyBufSize = 4096
	ReadTimeout = time.Millisecond * 100
)

func connectStream(pipe, stream io.ReadWriter) error {

	// @TODO: use context?

	var (
		errTo   error
		errFrom error
		wg      sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		// @TODO: check error
		_, errTo = io.Copy(pipe, stream)
		wg.Done()
	}()

	go func() {
		_, errFrom = io.Copy(stream, pipe)
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
