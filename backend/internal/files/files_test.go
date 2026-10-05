package files

import "testing"

func TestResolve(t *testing.T) {
	cases := map[string]string{
		"":                  "/home/container",
		"/":                 "/home/container",
		"server.properties": "/home/container/server.properties",
		"../../etc/passwd":  "/home/container/etc/passwd",
		"a/../../b":         "/home/container/b",
		"plugins\\x.jar":    "/home/container/plugins/x.jar",
	}
	for in, want := range cases {
		got, err := Resolve(in, nil)
		if err != nil || got != want {
			t.Errorf("Resolve(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestDenylist(t *testing.T) {
	deny := []string{"*.jar", "config/secret.yml", "private/"}
	for path, want := range map[string]bool{
		"/server.jar":        true,
		"/plugins/x.jar":     true,
		"/config/secret.yml": true,
		"/config/other.yml":  false,
		"/private/key":       true,
		"/server.properties": false,
		"/":                  false,
	} {
		if got := Denied(path, deny); got != want {
			t.Errorf("Denied(%q) = %v, want %v", path, got, want)
		}
	}
	if _, err := Resolve("server.jar", deny); err != ErrDenied {
		t.Errorf("Resolve on denied file = %v", err)
	}
}

func TestCheckReal(t *testing.T) {
	deny := []string{"server.properties"}
	for _, c := range []struct {
		reals []string
		want  error
	}{
		{[]string{"/home/container", "/home/container/world/level.dat"}, nil},
		{[]string{"/home/container/server.properties"}, ErrDenied},
		{[]string{"/etc/passwd"}, ErrNotFound},
		{[]string{"/home/containers/x"}, ErrNotFound},
		{[]string{"/proc/12/environ"}, ErrNotFound},
	} {
		if _, err := checkReal(c.reals, deny); err != c.want {
			t.Errorf("checkReal(%q) = %v, want %v", c.reals, err, c.want)
		}
	}
}

// TestRealDir: the entries of a folder behind a link are checked where they really are.
func TestRealDir(t *testing.T) {
	rels, err := checkReal([]string{"/home/container/secret", "/home/container"}, []string{"secret/"})
	if err != nil || len(rels) != 2 || rels[1] != "/" {
		t.Fatalf("checkReal = %q, %v", rels, err)
	}
	if got := realDir("/sl", rels); got != "/secret" || !Denied(got+"/key", []string{"secret/"}) {
		t.Errorf("realDir = %q", got)
	}
	if got := realDir("sl/", nil); got != "/sl" {
		t.Errorf("realDir without a denylist = %q", got)
	}
}
