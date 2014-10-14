package stream

import (
	"bufio"
	"reflect"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

type OneTwoThree int

func (a OneTwoThree) Next(data []byte, _ bool) (SplitState, int, []byte, error) {
	if int(a) < len(data) {
		return (a + 1) % 4, int(a) + 1, data[a : a+1], nil
	}
	return a - OneTwoThree(len(data)), len(data), nil, nil
}

func TestStatefulSplitFunc(t *testing.T) {
	in := bufio.NewScanner(strings.NewReader("a.b..c...de.f."))
	in.Split(StatefulSplitFunc(OneTwoThree(0)))
	tokens := []string{}
	for in.Scan() {
		tokens = append(tokens, in.Text())
	}
	if err := in.Err(); err != nil {
		t.Fatal("unexpected error: ", err)
	}
	expectedTokens := []string{"a", "b", "c", "d", "e", "f"}
	if !reflect.DeepEqual(tokens, expectedTokens) {
		t.Error("Tokens:\n", pretty.Compare(tokens, expectedTokens))
	}
}
