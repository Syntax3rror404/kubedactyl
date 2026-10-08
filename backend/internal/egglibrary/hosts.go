package egglibrary

import (
	"net/url"
	"path"
)

// host is how a git host serves a repository as one archive and its files one by one; every
// layout reads the default branch (HEAD). The URLs were checked against each public host.
type host struct {
	archive func(repo *url.URL) string
	raw     func(repo *url.URL, file string) string
}

var (
	// gitHub serves archives from codeload.github.com (no API rate limit).
	gitHub = host{
		archive: func(r *url.URL) string { return "https://codeload.github.com" + r.Path + "/tar.gz/HEAD" },
		raw: func(r *url.URL, file string) string {
			return "https://raw.githubusercontent.com" + r.Path + "/HEAD/" + file
		},
	}
	// gitLab also takes projects in subgroups (https://gitlab.com/<group>/<subgroup>/<project>).
	gitLab = host{
		archive: func(r *url.URL) string {
			return r.String() + "/-/archive/HEAD/" + path.Base(r.Path) + "-HEAD.tar.gz"
		},
		raw: func(r *url.URL, file string) string { return r.String() + "/-/raw/HEAD/" + file },
	}
	// gitea is also the layout of Forgejo (Codeberg).
	gitea = host{
		archive: func(r *url.URL) string { return r.String() + "/archive/HEAD.tar.gz" },
		raw:     func(r *url.URL, file string) string { return r.String() + "/raw/HEAD/" + file },
	}
	bitbucket = host{
		archive: func(r *url.URL) string { return r.String() + "/get/HEAD.tar.gz" },
		raw:     func(r *url.URL, file string) string { return r.String() + "/raw/HEAD/" + file },
	}
)

// knownHosts are the public hosts with their layout.
var knownHosts = map[string]host{
	"github.com":    gitHub,
	"gitlab.com":    gitLab,
	"codeberg.org":  gitea,
	"bitbucket.org": bitbucket,
}

// hostsOf returns the layouts to try for a repository: its own on a known host, else every
// layout a self-hosted server may use.
func hostsOf(repo *url.URL) []host {
	if h, ok := knownHosts[repo.Host]; ok {
		return []host{h}
	}
	return []host{gitLab, gitea, bitbucket}
}
