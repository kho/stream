package stream

import (
	"bufio"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type CopyIteratee []string

func (i *CopyIteratee) Final() error { return nil }
func (i *CopyIteratee) Next(token []byte) (Iteratee, bool, error) {
	*i = append(*i, string(token))
	return i, true, nil
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v' || b == '\xA0' || b == '\x85'
}

func LispTokenizer(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// skip white spaces
	var (
		i int
		b byte
	)
	for i, b = range data {
		if !isSpace(b) {
			break
		}
	}
	if isSpace(b) {
		return len(data), nil, nil
	}
	if b == '(' || b == ')' {
		return i + 1, data[i : i+1], nil
	}
	for j, b := range data[i+1:] {
		if isSpace(b) {
			return j + i + 2, data[i : j+i+1], nil
		} else if b == '(' || b == ')' {
			return j + i + 1, data[i : j+i+1], nil
		}
	}
	if atEOF {
		return len(data), data[i:], nil
	}
	return i, nil, nil
}

var (
	ErrLeftParen   = errors.New("expected (")
	ErrRightParent = errors.New("expected )")
	ErrAtom        = errors.New("expected atom")
)

type Balance int

func (i Balance) Final() error {
	if i == 0 {
		return nil
	}
	return ErrRightParent
}

func (i Balance) Next(token []byte) (Iteratee, bool, error) {
	switch string(token) {
	case "(":
		return i + 1, true, nil
	case ")":
		switch i {
		case 0:
			return nil, false, ErrLeftParen
		case 1:
			return nil, true, nil
		default:
			return i - 1, true, nil
		}
	default:
		return nil, false, ErrUnexpected
	}
}

func TestFoo(t *testing.T) {
	for _, i := range []struct {
		Input  string
		Tokens []string
	}{
		{"", nil},
		{"(())", []string{"(", "(", ")", ")"}},
		{"(a)", []string{"(", "a", ")"}},
		{"( a )", []string{"(", "a", ")"}},
		{"(ab cd)", []string{"(", "ab", "cd", ")"}},
		{"ab(cd e ) ", []string{"ab", "(", "cd", "e", ")"}},
		{"ab(\tcd\n e ) ", []string{"ab", "(", "cd", "e", ")"}},
	} {
		var tok CopyIteratee
		enum := NewScanEnumeratorWith(strings.NewReader(i.Input), LispTokenizer)
		if err := Run(enum, &tok); err != nil {
			t.Errorf("unexpected error: input %q; error %q", i, err)
		} else if !reflect.DeepEqual([]string(tok), i.Tokens) {
			t.Errorf("case %q; got %q", i, tok)
		}
	}

	abcN := Seq(Star(Seq(Match("a"), Match("b"), Star(Match("c")))), Seq(Match("x"), EOF))
	enumGood := NewScanEnumeratorWith(strings.NewReader("ababcabccx"), bufio.ScanBytes)
	if err := Run(enumGood, abcN); err != nil {
		t.Errorf("unexpected error: ", err)
	}
	enumBad := NewScanEnumeratorWith(strings.NewReader("abxy"), bufio.ScanBytes)
	if err := Run(enumBad, abcN); err == nil {
		t.Errorf("expect error")
	} else {
		t.Log("enumBad gives:", err)
	}

	var bal Balance
	if err := Run(NewScanEnumeratorWith(strings.NewReader("(()(()))"), bufio.ScanBytes), bal); err != nil {
		t.Errorf("got error %v", err)
	}
	if err := Run(NewScanEnumeratorWith(strings.NewReader("(()(()))("), bufio.ScanBytes), bal); err != nil {
		t.Errorf("got error %v", err)
	}
	if err := Run(NewScanEnumeratorWith(strings.NewReader("(()(())"), bufio.ScanBytes), bal); err == nil {
		t.Errorf("expect error")
	}
	if err := Run(NewScanEnumeratorWith(strings.NewReader("(()(()))("), bufio.ScanBytes), Seq(bal, EOF)); err == nil {
		t.Errorf("expect error")
	}
}
