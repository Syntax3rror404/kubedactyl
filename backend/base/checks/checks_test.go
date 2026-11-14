package checks

import "testing"

func TestWorst(t *testing.T) {
	cases := []struct {
		statuses []Status
		want     Status
	}{
		{nil, OK},
		{[]Status{OK, Skipped}, OK},
		{[]Status{OK, Warning, Skipped}, Warning},
		{[]Status{Warning, Error, OK}, Error},
	}
	for _, tc := range cases {
		var list []Check
		for _, s := range tc.statuses {
			list = append(list, New("x", "x", s, ""))
		}
		if got := Worst(list); got != tc.want {
			t.Errorf("Worst(%v) = %s, want %s", tc.statuses, got, tc.want)
		}
	}
}
