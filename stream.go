package stream

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
)

type Iteratee interface {
	Final() error
	Next([]byte) (Iteratee, bool, error)
}

type Enumerator interface {
	Step(Iteratee) (Iteratee, error)
}

type EnumScanner struct {
	in   *bufio.Scanner
	scan bool // true iff we must call scan before getting next token.
}

func (e *EnumScanner) Step(it Iteratee) (Iteratee, error) {
	log.Printf("enter %#v", it)
	if e.scan && !e.in.Scan() {
		err := e.in.Err()
		if err == nil {
			err = it.Final()
		}
		log.Printf("error %v", err)
		return nil, err
	}
	token := e.in.Bytes()
	next, read, err := it.Next(token)
	log.Printf("token %q; read %v", token, read)
	e.scan = read
	log.Printf("leave %#v : %v", next, err)
	return next, err
}

func EnumScan(in *bufio.Scanner) *EnumScanner {
	return &EnumScanner{in, true}
}

func EnumRead(in io.Reader, split bufio.SplitFunc) *EnumScanner {
	enum := EnumScan(bufio.NewScanner(in))
	enum.in.Split(split)
	return enum
}

func Run(e Enumerator, it Iteratee) (err error) {
	for {
		it, err = e.Step(it)
		if err != nil || it == nil {
			return
		}
	}
}

// eofI ensures there is no trailing input.
type eofI struct{}

func (_ eofI) Final() error { return nil }
func (_ eofI) Next(token []byte) (Iteratee, bool, error) {
	return nil, false, ErrTrailingInput(token)
}

type ErrTrailingInput string

func (e ErrTrailingInput) Error() string { return fmt.Sprintf("trailing input token: %q", string(e)) }

var (
	EOF = eofI{} // an iteratee that ensures there is no trailing input or returns ErrTrailingInput.
)

var ErrUnexpectedEnd = errors.New("unexpected end of input")

type ErrMistmatch struct {
	Expect, Got string
}

func (e ErrMistmatch) Error() string { return fmt.Sprintf("expect %q: got %q", e.Expect, e.Got) }

type ErrUnexpected string

func (e ErrUnexpected) Error() string { return fmt.Sprintf("unexpected token: %q", string(e)) }

type Match string

func (it Match) Final() error {
	return ErrUnexpectedEnd
}

func (it Match) Next(token []byte) (Iteratee, bool, error) {
	if string(token) == string(it) {
		return nil, true, nil
	}
	return nil, false, ErrMistmatch{string(it), string(token)}
}

type Then struct {
	A, B Iteratee
}

func (it Then) Final() error {
	if err := it.A.Final(); err != nil {
		return err
	}
	if err := it.B.Final(); err != nil {
		return err
	}
	return nil
}

func (it Then) Next(token []byte) (Iteratee, bool, error) {
	next, read, err := it.A.Next(token)
	if err != nil {
		return nil, false, err
	}
	if next != nil {
		return Then{next, it.B}, read, nil
	}
	return it.B, read, nil
}

type Star struct {
	A Iteratee
}

func (it Star) Final() error { return nil }
func (it Star) Next(token []byte) (Iteratee, bool, error) {
	next, read, err := it.A.Next(token)
	if err != nil {
		return nil, false, nil
	}
	if next != nil {
		return Then{next, it}, read, nil
	}
	return it, read, nil
}
