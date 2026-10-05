// imagebuild assembles the Kubedactyl runtime image without a Docker daemon, mirroring the
// last stage of the Dockerfile: alpine base + /kubedactyl + user 65532 + entrypoint.
//
//	go run . build -bin kubedactyl-linux-amd64 -tag ghcr.io/x/kubedactyl:0.1.0 -out image.tar
//	GHCR_USER=... GHCR_TOKEN=... go run . push -in image.tar -tag ghcr.io/x/kubedactyl:0.1.0
package main

import (
	"archive/tar"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: imagebuild build|push …")
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	base := fs.String("base", "alpine:3.24.2", "base image")
	bin := fs.String("bin", "", "linux/amd64 binary")
	tag := fs.String("tag", "", "image reference")
	out := fs.String("out", "image.tar", "output tarball")
	in := fs.String("in", "image.tar", "input tarball")
	version := fs.String("version", "0.1.0", "version label")
	source := fs.String("source", "", "source repository URL label")
	licenses := fs.String("licenses", "", "third-party-licenses.md, stored as /usr/share/licenses/kubedactyl/THIRD_PARTY_LICENSES.md")
	_ = fs.Parse(os.Args[2:])
	ref, err := name.ParseReference(*tag)
	must(err)
	switch os.Args[1] {
	case "build":
		build(*base, *bin, ref, *out, *version, *source, *licenses)
	case "push":
		img, err := tarball.ImageFromPath(*in, nil)
		must(err)
		auth := &authn.Basic{Username: os.Getenv("GHCR_USER"), Password: os.Getenv("GHCR_TOKEN")}
		must(remote.Write(ref, img, remote.WithAuth(auth)))
		d, _ := img.Digest()
		fmt.Println("pushed", ref, d)
	}
}

func build(base, bin string, ref name.Reference, out, version, source, licenses string) {
	platform := v1.Platform{OS: "linux", Architecture: "amd64"}
	baseImg, err := remote.Image(mustRef(base), remote.WithPlatform(platform))
	must(err)
	passwd, group := readFile(baseImg, "etc/passwd"), readFile(baseImg, "etc/group")
	// Same result as: addgroup -S -g 65532 kubedactyl && adduser -S -D -H -u 65532 -G kubedactyl kubedactyl
	passwd += "kubedactyl:x:65532:65532::/home/kubedactyl:/sbin/nologin\n"
	group += "kubedactyl:x:65532:kubedactyl\n"
	binary, err := os.ReadFile(bin)
	must(err)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	now := time.Unix(0, 0)
	add := func(path string, mode int64, data []byte) {
		must(tw.WriteHeader(&tar.Header{Name: path, Mode: mode, Size: int64(len(data)), ModTime: now, Typeflag: tar.TypeReg, Format: tar.FormatPAX}))
		_, err := tw.Write(data)
		must(err)
	}
	add("etc/passwd", 0o644, []byte(passwd))
	add("etc/group", 0o644, []byte(group))
	add("kubedactyl", 0o755, binary)
	if licenses != "" {
		text, err := os.ReadFile(licenses)
		must(err)
		for _, dir := range []string{"usr/share/licenses/", "usr/share/licenses/kubedactyl/"} {
			must(tw.WriteHeader(&tar.Header{Name: dir, Mode: 0o755, ModTime: now, Typeflag: tar.TypeDir, Format: tar.FormatPAX}))
		}
		add("usr/share/licenses/kubedactyl/THIRD_PARTY_LICENSES.md", 0o644, text)
	}
	must(tw.Close())
	layer, err := tarball.LayerFromReader(bytes.NewReader(buf.Bytes()))
	must(err)

	img, err := mutate.AppendLayers(baseImg, layer)
	must(err)
	cf, err := img.ConfigFile()
	must(err)
	cfg := cf.Config
	cfg.Entrypoint = []string{"/kubedactyl"}
	cfg.Cmd = nil
	cfg.User = "65532:65532"
	cfg.ExposedPorts = map[string]struct{}{"8080/tcp": {}}
	cfg.Labels = map[string]string{
		"org.opencontainers.image.title":       "kubedactyl",
		"org.opencontainers.image.description": "Game server panel for Kubernetes that runs Pterodactyl and Pelican eggs",
		"org.opencontainers.image.version":     version,
		"org.opencontainers.image.source":      source,
		"org.opencontainers.image.licenses":    "MIT",
		"org.opencontainers.image.base.name":   "docker.io/library/" + base,
	}
	img, err = mutate.Config(img, cfg)
	must(err)
	img, err = mutate.CreatedAt(img, v1.Time{Time: time.Now().UTC()})
	must(err)
	must(tarball.WriteToFile(out, ref, img))

	d, _ := img.Digest()
	cf, _ = img.ConfigFile()
	fmt.Printf("built %s\n  digest:     %s\n  platform:   %s/%s\n  entrypoint: %v  user: %s  ports: %v\n  layers:     %d\n", ref, d, cf.OS, cf.Architecture, cf.Config.Entrypoint, cf.Config.User, keys(cf.Config.ExposedPorts), len(mustLayers(img)))
}

func readFile(img v1.Image, path string) string {
	rc := mutate.Extract(img)
	defer rc.Close()
	tr := tar.NewReader(rc)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			log.Fatalf("%s not found in base image", path)
		}
		must(err)
		if strings.TrimPrefix(h.Name, "/") == path {
			b, err := io.ReadAll(tr)
			must(err)
			return string(b)
		}
	}
}

func mustRef(s string) name.Reference { r, err := name.ParseReference(s); must(err); return r }
func mustLayers(img v1.Image) []v1.Layer { l, err := img.Layers(); must(err); return l }
func keys(m map[string]struct{}) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
