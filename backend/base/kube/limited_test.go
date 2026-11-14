package kube

import (
	"errors"
	"testing"
)

func TestLimitedBuffer(t *testing.T) {
	b := LimitedBuffer{Max: 5}
	if n, err := b.Write([]byte("abc")); n != 3 || err != nil {
		t.Fatalf("Write = %d, %v", n, err)
	}
	if _, err := b.Write([]byte("def")); !errors.Is(err, ErrOutputTooLarge) || !b.Full {
		t.Fatalf("a write beyond the limit = %v, full %v", err, b.Full)
	}
	if b.String() != "abc" {
		t.Errorf("content = %q", b.String())
	}
}
