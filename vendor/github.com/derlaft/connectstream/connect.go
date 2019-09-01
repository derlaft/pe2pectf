package connectstream

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// ConnectError wrapps all possible errors that may be possible while connecting two streams
type ConnectError struct {
	CloseA error
	CloseB error
	CopyA  error
	CopyB  error
}

func (ce *ConnectError) isError() bool {
	return ce.CloseA != nil || ce.CloseB != nil || ce.CopyA != nil || ce.CopyB != nil
}

func (ce *ConnectError) fixReadClosedErr() {

	// Unfortunately, this error is unexported
	// so there is no other way to check for this error %)
	// https://github.com/golang/go/issues/4373

	var searchSubstr = "use of closed network connection"

	if ce.CopyA != nil && strings.Contains(ce.CopyA.Error(), searchSubstr) || ce.CopyA == io.ErrClosedPipe {
		ce.CopyA = nil
	}
	if ce.CopyB != nil && strings.Contains(ce.CopyB.Error(), searchSubstr) || ce.CloseB == io.ErrClosedPipe {
		ce.CopyB = nil
	}
}

func (ce *ConnectError) Error() string {

	var errorParts []string

	if ce.CopyA != nil {
		errorParts = append(errorParts, fmt.Sprintf("while copy A<-B: %v", ce.CopyA))
	}

	if ce.CopyB != nil {
		errorParts = append(errorParts, fmt.Sprintf("while copy B<-A: %v", ce.CopyB))
	}

	if ce.CloseA != nil {
		errorParts = append(errorParts, fmt.Sprintf("while closing A: %v", ce.CloseA))
	}

	if ce.CloseB != nil {
		errorParts = append(errorParts, fmt.Sprintf("while closing B: %v", ce.CloseB))
	}

	return fmt.Sprintf("Got %v errors while connecting two streams: %v",
		len(errorParts),
		strings.Join(errorParts, ", "))
}

func closeStreams(a, b io.Closer) (error, error) {
	return a.Close(), b.Close()
}

// Connect connects two streams using io.Copy
// whenever any operation returns, both streams are closed
// function returns only both Copy are finished
func Connect(a, b io.ReadWriteCloser) error {

	var (
		errors    ConnectError
		wg        sync.WaitGroup
		closeOnce sync.Once
		doClose   = func() {
			closeOnce.Do(func() {
				errors.CloseA, errors.CloseB = closeStreams(a, b)
			})
		}
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		defer doClose()

		_, errors.CopyA = io.Copy(a, b)
	}()

	go func() {
		defer wg.Done()
		defer doClose()

		_, errors.CopyB = io.Copy(b, a)
	}()

	wg.Wait()

	if errors.fixReadClosedErr(); errors.isError() {
		return &errors
	}

	return nil
}
