package cli

import (
	"fmt"
	"strings"
	"testing"

	"app/app"
)

func TestBanner(t *testing.T) {
	var plain strings.Builder
	PrintBanner(&plain, "1.2.3", false)
	if !strings.HasPrefix(plain.String(), app.Logo+"\n") || !strings.Contains(plain.String(), "v1.2.3 · ") {
		t.Errorf("plain banner:\n%s", plain.String())
	}
	if strings.Contains(plain.String(), "\x1b[") {
		t.Error("no escape codes without a terminal (log collectors would show them)")
	}
	var colored strings.Builder
	PrintBanner(&colored, "dev", true)
	from := app.BrandFrom
	brand := fmt.Sprintf("\x1b[38;2;%d;%d;%dm%c", from[0], from[1], from[2], []rune(app.Logo)[0])
	if !strings.HasPrefix(colored.String(), brand) || !strings.Contains(colored.String(), "\ndev · ") {
		t.Errorf("colored banner must start with the brand color:\n%q", colored.String())
	}
}
