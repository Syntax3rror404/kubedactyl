// Command licenses writes web/dist/third-party-licenses.md: the Kubedactyl license, the
// third-party notices, the license files of every Go module listed in THIRD_PARTY_NOTICES.md
// and the npm section the Vite build wrote (web/dist/licenses-npm.md). The panel shows the
// file on its licenses page (/licenses) and the image ships it under /usr/share/licenses/kubedactyl.
//
// The modules are tracked by hand in the "Go modules" table of THIRD_PARTY_NOTICES.md (name,
// version, license). The command fails when the table and the modules compiled into the
// linux/amd64 binary differ, and prints the rows to add or remove.
//
//	go run ./tools/licenses -root .. -dist web/dist
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var licenseFile = regexp.MustCompile(`(?i)^(licen[cs]e|copying|notice|copyright)([.-].*)?$`)

// module is a row of the table (path, version, license) and where its source is.
type module struct{ path, version, license, dir string }

func (m module) row() string { return "| " + m.path + " | " + m.version + " | " + m.license + " |" }

const tableHeading = "### Go modules"

func main() {
	root := flag.String("root", "..", "repository root (LICENSE, THIRD_PARTY_NOTICES.md)")
	dist := flag.String("dist", "web/dist", "frontend build output (embedded into the binary)")
	flag.Parse()

	notices, err := os.ReadFile(filepath.Join(*root, "THIRD_PARTY_NOTICES.md"))
	if err != nil {
		log.Fatal(err)
	}
	listed := listedModules(notices)
	built, err := goModules()
	if err != nil {
		log.Fatal(err)
	}
	if problems := compare(listed, built); len(problems) > 0 {
		log.Fatalf("THIRD_PARTY_NOTICES.md (%s) does not match the panel binary:\n%s",
			tableHeading, strings.Join(problems, "\n"))
	}
	npm, err := os.ReadFile(filepath.Join(*dist, "licenses-npm.md"))
	if err != nil {
		log.Fatalf("%v: run the frontend build first", err)
	}
	out, err := licenses(*root, notices, built, listed, npm)
	if err != nil {
		log.Fatal(err)
	}
	target := filepath.Join(*dist, "third-party-licenses.md")
	if err := os.WriteFile(target, out, 0o644); err != nil {
		log.Fatal(err)
	}
	// The npm part is now in the combined file.
	if err := os.Remove(filepath.Join(*dist, "licenses-npm.md")); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %s (%d Go modules, %d KiB)\n", target, len(listed), len(out)/1024)
}

// licenses joins the license, the notices, the texts of the listed Go modules and the npm part
// into one Markdown document; license texts are code blocks, so they keep their line breaks.
func licenses(root string, notices []byte, built map[string]module, listed []module, npm []byte) ([]byte, error) {
	license, err := os.ReadFile(filepath.Join(root, "LICENSE"))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString(
		"# Kubedactyl licenses\n\n" +
			"Kubedactyl itself is licensed under the MIT License below. It contains or bundles the\n" +
			"third-party material listed after it: notices for copied code and assets, then the license\n" +
			"texts of every Go module in the panel binary and every npm package in the web interface.\n\n" +
			"## License\n\n" + codeBlock(string(license)) + "\n\n" + demote(string(notices)),
	)
	fmt.Fprintf(&out, "\n\n## Go modules in the panel binary (%d)", len(listed))
	for _, m := range listed {
		texts, err := licenseTexts(built[m.path].dir)
		if err != nil {
			return nil, err
		}
		if len(texts) == 0 {
			return nil, fmt.Errorf("%s has no license file", m.path)
		}
		out.WriteString("\n\n### " + m.path + " " + m.version + " (" + m.license + ")\n\n" +
			codeBlock(strings.Join(texts, "\n\n")))
	}
	out.WriteString("\n\n" + strings.TrimSpace(string(npm)) + "\n")
	return out.Bytes(), nil
}

// codeBlock fences text with more backticks than any run inside it.
func codeBlock(text string) string {
	longest, run := 0, 0
	for _, r := range text {
		if r == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	fence := strings.Repeat("`", max(3, longest+1))
	return fence + "\n" + strings.TrimSpace(text) + "\n" + fence
}

// demote moves every heading of a Markdown text one level down (outside code blocks), so the
// notices sit below the document title.
func demote(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	fenced := false
	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
		} else if !fenced && strings.HasPrefix(line, "#") {
			lines[i] = "#" + line
		}
	}
	return strings.Join(lines, "\n")
}

// listedModules reads the rows of the Go modules table.
func listedModules(notices []byte) []module {
	var mods []module
	in := false
	for line := range strings.SplitSeq(string(notices), "\n") {
		if strings.HasPrefix(line, "#") {
			in = strings.HasPrefix(line, tableHeading)
			continue
		}
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if !in || len(cells) != 3 {
			continue
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		m := module{path: cells[0], version: cells[1], license: cells[2]}
		if m.path != "Module" && !strings.HasPrefix(m.path, "---") {
			mods = append(mods, m)
		}
	}
	return mods
}

// compare lists the rows that are missing, outdated or no longer used.
func compare(listed []module, built map[string]module) []string {
	var problems []string
	seen := map[string]bool{}
	for _, m := range listed {
		seen[m.path] = true
		switch b, ok := built[m.path]; {
		case !ok:
			problems = append(problems, "  not in the binary any more, remove: "+m.row())
		case b.version != m.version:
			problems = append(problems, fmt.Sprintf("  %s is now %s (check its license): %s",
				m.path, b.version, m.row()))
		}
	}
	for path, b := range built {
		if !seen[path] {
			problems = append(problems, "  missing (add with its license): "+b.row())
		}
	}
	slices.Sort(problems)
	return problems
}

// goModules lists the modules of the packages the linux/amd64 panel binary is built from.
func goModules() (map[string]module, error) {
	cmd := exec.Command(
		"go", "list", "-deps", "-f", "{{with .Module}}{{if not .Main}}{{.Path}}\t{{.Version}}\t{{.Dir}}{{end}}{{end}}",
		".",
	)
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	cmd.Stderr = os.Stderr
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}
	mods := map[string]module{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) == 3 {
			mods[f[0]] = module{path: f[0], version: f[1], license: "?", dir: f[2]}
		}
	}
	return mods, nil
}

// licenseTexts returns the license, copying and notice files in the root of a module.
func licenseTexts(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var texts []string
	for _, e := range entries {
		if e.IsDir() || !licenseFile.MatchString(e.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		texts = append(texts, strings.TrimSpace(string(data)))
	}
	return texts, nil
}
