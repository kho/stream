package stream

import (
	"bufio"
	"fmt"
	"io"
)

// ScanEnumerator is an Enumerator with a backing bufio.Scanner.
type ScanEnumerator struct {
	in   *bufio.Scanner
	scan bool // true iff we must call scan before getting next token.
}

func (e *ScanEnumerator) Step(it Iteratee) (Iteratee, error) {
	if e.scan && !e.in.Scan() {
		err := e.in.Err()
		if err == nil {
			err = it.Final()
		}
		return nil, err
	}
	token := e.in.Bytes()
	next, read, err := it.Next(token)
	e.scan = read
	return next, WrapTokenError(token, err)
}

func NewScanEnumerator(in *bufio.Scanner) *ScanEnumerator {
	return &ScanEnumerator{in, true}
}

func NewScanEnumeratorWith(in io.Reader, split bufio.SplitFunc) *ScanEnumerator {
	enum := NewScanEnumerator(bufio.NewScanner(in))
	enum.in.Split(split)
	return enum
}

// TokenErr wraps an error with the input token.
type TokenErr struct {
	Token string
	Err   error
}

func (e TokenErr) Error() string {
	return fmt.Sprintf("token %q: %v", e.Token, e.Err)
}

// WrapTokenError creates an appropriate error when err is not nil.
func WrapTokenError(token Token, err error) error {
	if err == nil {
		return nil
	}
	return TokenErr{string(token), err}
}
