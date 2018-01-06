package stack

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sniperkit/xtask/pkg"
)

func testPop(t *testing.T, s *string_stack, right_e string, right_err error) {
	e, err := s.pop()

	// Wrong element returned.
	if e != right_e {
		t.Errorf("%#v != %#v", e, right_e)
	}

	// Error returned when it shouldn't.
	if err == nil && right_err != nil {
		t.Errorf("%#v != %#v", err, right_err)
	}

	// Error not returned when it should or wrong error returned.
	if err != nil && (right_err == nil || err.Error() != right_err.Error()) {
		t.Errorf("%#v != %#v", err, right_err)
	}
}

func TestSinglePopError(t *testing.T) {
	s := NewStringStack()
	testPop(t, s, "", errors.New("Stack is empty"))
}

func TestErrorSinglePopPushPopToError(t *testing.T) {
	s := NewStringStack()
	testPop(t, s, "", errors.New("Stack is empty"))
	s.push("foo")
	testPop(t, s, "foo", nil)
	testPop(t, s, "", errors.New("Stack is empty"))
	testPop(t, s, "", errors.New("Stack is empty"))
}

func TestSinglePushPopToError(t *testing.T) {
	s := NewStringStack()
	s.push("foo")
	testPop(t, s, "foo", nil)
	testPop(t, s, "", errors.New("Stack is empty"))
	testPop(t, s, "", errors.New("Stack is empty"))
}

func TestMultiPushPopToError(t *testing.T) {
	s := NewStringStack()
	s.push("foo")
	s.push("bar")
	s.push("baz")
	testPop(t, s, "baz", nil)
	testPop(t, s, "bar", nil)
	testPop(t, s, "foo", nil)
	testPop(t, s, "", errors.New("Stack is empty"))
	testPop(t, s, "", errors.New("Stack is empty"))
}

func TestMultiPushPopToErrorSequence(t *testing.T) {
	s := NewStringStack()
	s.push("foo")
	s.push("bar")
	s.push("baz")
	testPop(t, s, "baz", nil)
	testPop(t, s, "bar", nil)
	testPop(t, s, "foo", nil)
	testPop(t, s, "", errors.New("Stack is empty"))
	testPop(t, s, "", errors.New("Stack is empty"))
	s.push("quid")
	s.push("pro")
	s.push("quo")
	testPop(t, s, "quo", nil)
	testPop(t, s, "pro", nil)
	testPop(t, s, "quid", nil)
	testPop(t, s, "", errors.New("Stack is empty"))
	testPop(t, s, "", errors.New("Stack is empty"))
	s.push("kn")
	s.push("o")
	s.push("ck")
	testPop(t, s, "ck", nil)
	testPop(t, s, "o", nil)
	testPop(t, s, "kn", nil)
	testPop(t, s, "", errors.New("Stack is empty"))
	testPop(t, s, "", errors.New("Stack is empty"))
}
