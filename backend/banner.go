package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// logo is printed once when the panel starts (FIGlet font "smblock", one column between the letters).
const logo = `▌ ▌ ▌ ▌ ▛▀▖ ▛▀▘ ▛▀▖ ▞▀▖ ▞▀▖ ▀▛▘ ▌ ▌ ▌
▙▞  ▌ ▌ ▙▄▘ ▙▄  ▌ ▌ ▙▄▌ ▌    ▌  ▝▞  ▌
▌▝▖ ▌ ▌ ▌ ▌ ▌   ▌ ▌ ▌ ▌ ▌ ▖  ▌   ▌  ▌
▘ ▘ ▝▀  ▀▀  ▀▀▘ ▀▀  ▘ ▘ ▝▀   ▘   ▘  ▀▀▘`

// Brand gradient of the web interface (emerald-500 to sky-600), from left to right.
var logoFrom, logoTo = [3]int{0x00, 0xbc, 0x7d}, [3]int{0x00, 0x84, 0xd1}

// printBanner writes the logo and the version. Colors only on a terminal: log collectors would show the escape codes.
func printBanner(w io.Writer, version string, color bool) {
	lines := strings.Split(logo, "\n")
	width := 0
	for _, line := range lines {
		width = max(width, utf8.RuneCountInString(line))
	}
	for _, line := range lines {
		if color {
			line = gradient(line, width)
		}
		fmt.Fprintln(w, line)
	}
	if version != "dev" {
		version = "v" + version
	}
	fmt.Fprintf(w, "%s · game servers on Kubernetes\n\n", version)
}

// gradient colors every character of line by its column (24-bit terminal colors).
func gradient(line string, width int) string {
	var b strings.Builder
	for i, r := range []rune(line) {
		mix := func(k int) int { return logoFrom[k] + (logoTo[k]-logoFrom[k])*i/max(width-1, 1) }
		fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm%c", mix(0), mix(1), mix(2), r)
	}
	return b.String() + "\x1b[0m"
}

// colorTerminal reports whether f is a terminal that wants colors (NO_COLOR unset).
func colorTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0 && os.Getenv("NO_COLOR") == ""
}
