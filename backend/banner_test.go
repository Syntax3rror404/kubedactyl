package main

import (
	"strings"
	"testing"
)

func TestBanner(t *testing.T) {
	var plain strings.Builder
	printBanner(&plain, "1.2.3", false)
	if !strings.HasPrefix(plain.String(), logo+"\n") || !strings.Contains(plain.String(), "v1.2.3 · ") {
		t.Errorf("plain banner:\n%s", plain.String())
	}
	if strings.Contains(plain.String(), "\x1b[") {
		t.Error("no escape codes without a terminal (log collectors would show them)")
	}
	var colored strings.Builder
	printBanner(&colored, "dev", true)
	if !strings.Contains(colored.String(), "\x1b[38;2;0;188;125m▌") || !strings.Contains(colored.String(), "\ndev · ") {
		t.Errorf("colored banner must start with the brand color:\n%q", colored.String())
	}
}
