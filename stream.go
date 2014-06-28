package stream

import (
	"errors"
	"fmt"
)

// Token is the unit of input being consumed by Iteratees.
type Token []byte

// Iteratee is a state in a (not necessarily finite) state machine.
//
// TODO: some convention about data fields.
type Iteratee interface {
	// Final indicates the end of input and requests the Iteratee to
	// finish any left-over work.
	Final() error
	// Next takes in the next token and take the transition. The token
	// should not be modified. Also the token is only guaranteed to
	// remain unchanged before Next() returns (thus copies should be
	// made if one needs to retain information in the token). The return
	// value is interpreted in the following way:
	//
	// - If any error is encountered, it should return a non-nil error
	// and other return values are ignored.
	//
	// - If there is no error, the returned Iteratee is the next state
	// and the returned bool indicates whether the input token has been
	// consumed. Specifically, the next state may be nil, indicating
	// reaching a final state.
	Next(Token) (Iteratee, bool, error)
}

// Enumerator is the input source. It gathers input tokens and
// executes it on Iteratees.
type Enumerator interface {
	// Step feeds the current token to the given Iteratee. It may assume
	// the input Iteratee is not nil. When the end of input is
	// encountered, it calls the Final() method of the input Iteratee
	// and returns a nil Iteratee. Successive calls of Step on returning
	// non-nill Iteratees should correctly consume the sequence of input
	// tokens (see Run()).
	Step(Iteratee) (Iteratee, error)
}

// Run executes e starting with it by Stepping e until it reaches a
// final state (either by reaching the end of input or an actual nil
// Iteratee). Returns the first error encountered.
func Run(e Enumerator, it Iteratee) (err error) {
	for {
		it, err = e.Step(it)
		if err != nil || it == nil {
			return
		}
	}
}

// Simple utility Iteratees.

// eofI ensures there is no trailing input.
type eofI struct{}

func (_ eofI) Final() error { return nil }
func (_ eofI) Next(token Token) (Iteratee, bool, error) {
	return nil, false, ErrExpect("<eof>")
}

// skipI skips exactly one token.
type skipI struct{}

func (_ skipI) Final() error { return ErrExpect("a token") }
func (_ skipI) Next(token Token) (Iteratee, bool, error) {
	return nil, true, nil
}

var (
	EOF  = eofI{}  // an Iteratee that ensures there is no trailing input or returns error.
	Skip = skipI{} // an Iteratee that skips exactly one token.
)

// Match requires the next token to be exactly the underlying string
// (not EOF, not anything else). When the next token matches, it
// finishes successfully. Otherwise an error is returned.
func Match(s string) Iteratee {
	return matchI(s)
}

// matchI implements Match().
type matchI string

func (it matchI) Final() error { return ErrExpectQ(it) }
func (it matchI) Next(token Token) (Iteratee, bool, error) {
	if string(token) == string(it) {
		return nil, true, nil
	}
	return nil, false, ErrExpectQ(it)
}

// SkipAny skips zero or more repetition of s.
func SkipAny(s string) Iteratee {
	return skipAnyI(s)
}

// skipAnyI implements SkipAny.
type skipAnyI string

func (it skipAnyI) Final() error { return nil }
func (it skipAnyI) Next(token Token) (Iteratee, bool, error) {
	if string(token) == string(it) {
		return it, true, nil
	}
	return nil, false, nil
}

// Seq represents an Iteratee, when run executes each Iteratee to
// final in order.
func Seq(its ...Iteratee) Iteratee {
	return seqI(its)
}

// thenI executes A to final and then B to final.
type thenI struct {
	A, B Iteratee
}

func (it thenI) Final() error {
	if err := it.A.Final(); err != nil {
		return err
	}
	if err := it.B.Final(); err != nil {
		return err
	}
	return nil
}

func (it thenI) Next(token Token) (Iteratee, bool, error) {
	next, read, err := it.A.Next(token)
	if err != nil {
		return nil, false, err
	}
	if next != nil {
		return thenI{next, it.B}, read, nil
	}
	return it.B, read, nil
}

// seqI implements Seq(). Its content must no be modified during
// execution.
type seqI []Iteratee

func (it seqI) Final() error {
	for _, i := range it {
		if err := i.Final(); err != nil {
			return err
		}
	}
	return nil
}

func (it seqI) Next(token Token) (Iteratee, bool, error) {
	if len(it) == 0 {
		return nil, false, nil
	}
	next, read, err := it[0].Next(token)
	if err != nil {
		return nil, false, err
	}
	if next != nil {
		return thenI{next, it[1:]}, read, nil
	}
	return it[1:], read, nil
}

// Star repeats an Iteratee until it can not further proceed.
func Star(it Iteratee) Iteratee {
	return starI{it}
}

// starI implements Star().
type starI struct {
	A Iteratee
}

func (it starI) Final() error { return nil }
func (it starI) Next(token Token) (Iteratee, bool, error) {
	next, read, err := it.A.Next(token)
	if err != nil {
		return nil, false, nil
	}
	if next != nil {
		return thenI{next, it}, read, nil
	}
	return it, read, nil
}

// Useful errors.

// ErrUnexpected reports an unexpected token.
var ErrUnexpected = errors.New("unexpected token")

// ErrExpect reports that something is expected but not given.
type ErrExpect string

func (e ErrExpect) Error() string { return fmt.Sprintf("expect %s", string(e)) }

// ErrExpectQ is like ErrExpect but quotes the string.
type ErrExpectQ string

func (e ErrExpectQ) Error() string { return fmt.Sprintf("expect %q", string(e)) }
