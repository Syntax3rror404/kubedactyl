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

// TestDenylist: patterns are read like a .gitignore file, a malformed one denies everything.
func TestDenylist(t *testing.T) {
	cases := []struct {
		deny []string
		path string
		want bool
	}{
		{[]string{"*.jar"}, "/server.jar", true},
		{[]string{"*.jar"}, "/plugins/x.jar", true},
		{[]string{"*.jar"}, "/server.properties", false},
		{[]string{"config/secret.yml"}, "/config/secret.yml", true},
		{[]string{"config/secret.yml"}, "/config/other.yml", false},
		{[]string{"config/secret.yml"}, "/x/config/secret.yml", false},
		{[]string{"/server.jar"}, "/server.jar", true},
		{[]string{"private/"}, "/private/key", true},
		{[]string{"private/"}, "/sub/private/key", true},
		{[]string{"private/"}, "/private", false},
		{[]string{"private"}, "/private", true},
		{[]string{"*.d"}, "/conf.d/x.conf", true},
		{[]string{"**/secret.yml"}, "/a/b/secret.yml", true},
		{[]string{"**/secret.yml"}, "/secret.yml", true},
		{[]string{"config/**"}, "/config/a/b.yml", true},
		{[]string{"config/**"}, "/config", false},
		{[]string{"config/**/key"}, "/config/key", true},
		{[]string{"config/**/key"}, "/config/a/b/key", true},
		{[]string{"config/**/key"}, "/other/a/key", false},
		{[]string{"config/*/key"}, "/config/a/b/key", false},
		{[]string{"!server.jar", "", " "}, "/server.jar", false},
		{[]string{"["}, "/server.properties", true},
		{[]string{"*.jar", "a\\"}, "/world/level.dat", true},
		{[]string{"["}, "/", false},
		{nil, "/server.jar", false},
	}
	for _, c := range cases {
		if got := denied(c.path, c.deny); got != c.want {
			t.Errorf("denied(%q, %q) = %v, want %v", c.path, c.deny, got, c.want)
		}
	}
	if _, err := Resolve("server.jar", []string{"*.jar"}); err != ErrDenied {
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
	if got := realDir("/sl", rels); got != "/secret" || !denied(got+"/key", []string{"secret/"}) {
		t.Errorf("realDir = %q", got)
	}
	if got := realDir("sl/", nil); got != "/sl" {
		t.Errorf("realDir without a denylist = %q", got)
	}
}
