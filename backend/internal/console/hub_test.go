package console

import (
	"bufio"
	"io"
	"strings"
	"testing"
	"time"
)

func TestDeduperReconnect(t *testing.T) {
	t0 := time.Date(2026, 9, 28, 12, 0, 0, 100, time.UTC)
	t1 := t0.Add(time.Millisecond)
	var d deduper
	d.reconnect()
	// First stream: two lines share t1.
	for i, ts := range []time.Time{t0, t1, t1} {
		if !d.accept(ts) {
			t.Fatalf("line %d dropped on first stream", i)
		}
	}
	// Reconnect replays from the start of the second: t0, t1, t1 are old, then a new t1 line and t2.
	d.reconnect()
	want := []bool{false, false, false, true, true}
	for i, ts := range []time.Time{t0, t1, t1, t1, t1.Add(time.Second)} {
		if got := d.accept(ts); got != want[i] {
			t.Errorf("replayed line %d: accept=%v, want %v", i, got, want[i])
		}
	}
}

func TestSplitTimestamp(t *testing.T) {
	ts, text := splitTimestamp(
		"2026-09-28T02:44:15.123456789Z [12:00:00 INFO]: Done (3.2s)! For help, type \"help\"\r\n",
	)
	if ts.IsZero() || text != `[12:00:00 INFO]: Done (3.2s)! For help, type "help"` {
		t.Fatalf("got %v %q", ts, text)
	}
	if _, text := splitTimestamp("2026-09-28T02:44:15Z 10%\r50%\r100%\n"); text != "100%" {
		t.Errorf("carriage return handling: %q", text)
	}
}

func TestMatcher(t *testing.T) {
	m := NewMatcher([]string{")! For help, type ", "regex:^Server started on port \\d+$"}, true)
	if !m.Match("\x1b[32m[12:00 INFO]: Done (1s)! For help, type \"help\"\x1b[0m") {
		t.Error("plain match failed")
	}
	if !m.Match("Server started on port 25565") || m.Match("Server started on port x") {
		t.Error("regex match wrong")
	}
}

// TestReadLine: a line without end is cut at the buffer size, the rest of it is skipped.
func TestReadLine(t *testing.T) {
	long := strings.Repeat("x", 3*maxLineLength)
	r := bufio.NewReaderSize(strings.NewReader("short\n"+long+"\nnext\n"+long), maxLineLength)
	var got []string
	for {
		line, err := readLine(r)
		got = append(got, line)
		if err == io.EOF {
			break
		}
	}
	want := []string{"short\n", long[:maxLineLength] + " …\n", "next\n", long[:maxLineLength] + " …\n"}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %d bytes, want %d", i, len(got[i]), len(want[i]))
		}
	}
}
