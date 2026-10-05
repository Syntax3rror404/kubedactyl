package configfile

import (
	"fmt"
	"strings"
	"testing"
)

var cfg = NewConfig("10.0.0.1")

func apply(t *testing.T, parser, content string, exists bool, reps ...Replacement) string {
	t.Helper()
	out, write, err := Apply(parser, []byte(content), exists, reps, cfg)
	if err != nil {
		t.Fatalf("%s: %v", parser, err)
	}
	if !write {
		return "<skipped>"
	}
	return string(out)
}

func TestProperties(t *testing.T) {
	in := "#Minecraft server properties\n#Mon Sep 28\nmotd=Hello §a\nserver-port=25565\nserver-ip=1.2.3.4\n"
	out := apply(t, "properties", in, true,
		Replacement{Match: "server-ip", Value: ""},
		Replacement{Match: "server-port", Value: "25570"},
		Replacement{Match: "query.port", Value: "25570"},
		Replacement{Match: "motd", IfValue: "nope", Value: "ignored"},
	)
	for _, want := range []string{
		"#Minecraft server properties\n#Mon Sep 28\n", "server-ip=\n", "server-port=25570\n", "query.port=25570\n",
		"motd=Hello \\u00a7a",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// Missing files are created.
	if out := apply(
		t, "properties", "", false, Replacement{Match: "server-port", Value: "1"},
	); out != "server-port=1\n" {
		t.Errorf("new file = %q", out)
	}
}

// TestNestedReferences: references to other keys stay as written: a few hundred bytes of nested
// references must not expand to gigabytes in the panel's memory.
func TestNestedReferences(t *testing.T) {
	for parser, ref := range map[string]string{"properties": "${k%d}", "ini": "%%(k%d)s"} {
		var in strings.Builder
		in.WriteString("k0=" + strings.Repeat("A", 64) + "\n")
		for i := 1; i <= 40; i++ {
			fmt.Fprintf(&in, "k%d="+ref+ref+"\n", i, i-1, i-1)
		}
		out := apply(t, parser, in.String(), true, Replacement{Match: "port", Value: "1"})
		if len(out) > 2*in.Len() || !strings.Contains(out, fmt.Sprintf(ref, 0)) {
			t.Errorf("%s: references were expanded (%d bytes)", parser, len(out))
		}
	}
}

func TestYamlWildcardAndConfigLookup(t *testing.T) {
	in := "listeners:\n- host: 0.0.0.0:25577\n  motd: hi\n" +
		"servers:\n  lobby:\n    address: 127.0.0.1:25565\n  pvp:\n    address: 127.0.0.1\n"
	out := apply(t, "yaml", in, true,
		Replacement{Match: "listeners[0].host", Value: "0.0.0.0:25580"},
		Replacement{Match: "servers.*.address", IfValue: "127.0.0.1", Value: "{{config.docker.interface}}"},
	)
	for _, want := range []string{"host: 0.0.0.0:25580", "address: 127.0.0.1:25565", "address: 10.0.0.1"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestJSONTypes(t *testing.T) {
	out := apply(t, "json", `{"port": 1, "name": "x", "nested": {"public": false}}`, true,
		Replacement{Match: "port", Value: "2302"},
		Replacement{Match: "nested.public", Value: "true", Type: TypeBoolean},
		Replacement{Match: "name", Value: "My Server"},
		Replacement{Match: "new.key", Value: "v"},
	)
	for _, want := range []string{`"port": 2302`, `"public": true`, `"name": "My Server"`, `"key": "v"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestIni(t *testing.T) {
	out := apply(t, "ini", "[server]\nport=1\n", true,
		Replacement{Match: "server.port", Value: "7777"},
		Replacement{Match: "[/Script/Engine.Game].MaxPlayers", Value: "10"},
		Replacement{Match: "rootkey", Value: "x"},
	)
	for _, want := range []string{"port = 7777", "[/Script/Engine.Game]", "MaxPlayers = 10", "rootkey = x"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestXML(t *testing.T) {
	out := apply(t, "xml", "<Settings><Port>1</Port></Settings>", true,
		Replacement{Match: "Settings.Port", Value: "7777"},
		Replacement{Match: "Settings.Name", Value: "[value='test']"},
	)
	for _, want := range []string{"<Port>7777</Port>", `<Name value="test"/>`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if out := apply(
		t, "xml", "", false, Replacement{Match: "Root.Port", Value: "1"},
	); !strings.Contains(out, "<Root>") ||
		!strings.Contains(out, "<Port>1</Port>") {
		t.Errorf("new xml = %s", out)
	}
}

func TestFile(t *testing.T) {
	out := apply(
		t, "file", "port 1\nname x\n", true, Replacement{Match: "port", Value: "port {{config.docker.interface}}"},
	)
	// The file parser writes values literally (no {{config}} lookup).
	if out != "port {{config.docker.interface}}\nname x\n" {
		t.Errorf("file = %q", out)
	}
	if out := apply(t, "file", "", false, Replacement{Match: "a", Value: "b"}); out != "<skipped>" {
		t.Errorf("missing file should be skipped, got %q", out)
	}
}
