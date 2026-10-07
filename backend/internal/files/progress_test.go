package files

import "testing"

func TestProgressWriter(t *testing.T) {
	var got []Progress
	w := &progressWriter{report: func(p Progress) { got = append(got, p) }}
	// A backup: lines arrive in pieces, folders do not count, the root "./" is skipped.
	for _, chunk := range []string{"total 2\n./\n./wor", "ld/\n./world/level.dat\n", "./server.properties\n"} {
		if _, err := w.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	want := []Progress{
		{Unit: ProgressFiles, Total: 2},
		{Unit: ProgressFiles, Total: 2, File: "world/"},
		{Unit: ProgressFiles, Total: 2, Done: 1, File: "world/level.dat"},
		{Unit: ProgressFiles, Total: 2, Done: 2, File: "server.properties"},
	}
	if len(got) != len(want) {
		t.Fatalf("reports = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("report %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	// A restore: byte positions, names only change the file; an empty position is skipped.
	got = nil
	w = &progressWriter{report: func(p Progress) { got = append(got, p) }}
	_, _ = w.Write([]byte("pos 0 100\npos  100\n./a.txt\npos 60 100\n"))
	last := Progress{Unit: ProgressBytes, Done: 60, Total: 100, File: "a.txt"}
	if len(got) != 3 || got[2] != last {
		t.Errorf("reports = %+v", got)
	}
}
