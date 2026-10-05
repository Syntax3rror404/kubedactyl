# Security

What the panel does to be safe on the internet. To report a vulnerability, see [SECURITY.md](../SECURITY.md).

Measures for exposing the panel to the internet:

- **Setup token** (see [Users and permissions](administration.md#users-and-permissions)), Argon2id passwords, HMAC session cookie (HttpOnly, SameSite=Strict,
  `Secure` over HTTPS), API tokens stored as SHA-256, login throttling per client IP + username, per client
  IP across usernames and per username across clients (the throttle forgets expired entries and remembers at
  most 10 000 keys; usernames over 64 and passwords over 1024 characters are refused before they reach it).
  The password of a disabled user is checked like any other, so the answer does not tell
  disabled accounts apart. At most two
  password checks run at once (19 MiB each, OWASP's first Argon2id configuration, 2 iterations; older hashes with
  64 MiB are replaced at the next sign-in), so a burst of logins cannot exhaust the panel's memory. The last
  active administrator cannot be demoted, deactivated or deleted, also not by parallel requests.
- **Public routes** (`/branding`, `/legal`, `/setup`) do not call the Kubernetes API once an administrator exists
  (the settings are kept in memory and read again only after they changed), so anonymous requests cannot use up the panel's
  Kubernetes API limit. Every console message counts against the user's request limit like an HTTP request.
  Values every viewer polls are read once and shared: server CPU/memory (10 s), disk usage (30 s), cluster health
  (30 s), the panel's permission checks (10 min) and the node list with its metrics (10 s). More viewers do not
  mean more Kubernetes API calls.
- **Memory:** the panel parses egg config files of at most 1 MiB (larger ones are skipped with a warning) and
  never expands `${…}` references in `.properties` files; console lines are cut at 4 KiB.
- **Lifetimes:** sign-ins and API tokens expire (default 12 hours / 90 days, adjustable under *Settings*); the
  limits are checked on every request, also for sessions and tokens created before a change.
- **Sessions:** logging out ends the session at once (in memory until it expires; "log out everywhere" also
  survives restarts); open consoles check their credentials every 30 seconds and before every command and close
  after a logout, a disabled account, a password change or a lost admin role.
- **Client IP:** gin trusts only the proxies in `--trusted-proxies` (chart default `10.0.0.0/8`, the usual pod
  networks of an in-cluster gateway), so `X-Forwarded-For` cannot be spoofed to evade the throttle. Without it,
  all clients behind a proxy share the proxy IP and its login limit.
- **CSRF:** SameSite=Strict plus a `Sec-Fetch-Site` check: state changing requests with the session
  cookie from another site (also another subdomain) are rejected; API token requests are not affected.
  Sign-in and setup only take JSON and refuse other sites, so no page can sign a visitor into another account.
- **Headers:** a strict Content-Security-Policy (`script-src 'self'`, no inline scripts; the theme
  script is `/theme-init.js`; inline styles are needed by xterm.js/CodeMirror), `frame-ancestors
  'none'`/`X-Frame-Options: DENY`, `nosniff`, `Referrer-Policy`, `Permissions-Policy`, HSTS over HTTPS.
  The Swagger UI has no CSP (inline scripts), all API calls still need a token. API answers carry
  `Cache-Control: no-store` (tokens, settings, file contents).
- **Branding and legal texts** carry an ETag: a page load gets 304 while they are unchanged, so logo and favicon
  (up to 128 KiB each) are not sent again.
- **Web UI files:** hashed files under `assets/` are cached for a year (`immutable`), `index.html` and the other
  files are revalidated (`no-cache` + ETag: 304 while unchanged); text files are sent gzip compressed (compressed
  once at start).
- **Requests** are limited to 8 MiB (uploads stream and are not limited); websocket connections only
  from the same origin (localhost ports only in dev mode).
- **No information leaks:** internal errors show details only to administrators; `/api/info` only
  returns name, version and mode; servers of other users answer 404.
- **Network isolation** (panel setting, on by default): every user namespace gets the NetworkPolicy
  `kubedactyl-isolation`: game servers (which run user uploaded plugins/mods) only reach the
  internet, the cluster DNS and the user's own servers, not other namespaces, nodes, the Kubernetes
  API, cloud metadata or private networks. Verified live with and without the policy.
- **Pods:** no pod gets a service account token. Game pods run as UID 988 with a read-only root filesystem
  and no capabilities. The files helper runs file operations the same way; only its init container hands files
  back to UID 988 as root, without following links. The panel uses a files pod only when its server controls it
  (owner reference; otherwise 409).
- **File manager:** symbolic links (made by the game, a plugin or an archive) never lead out of the volume:
  every read, download, listing, write, extraction and download target is resolved first (404 outside), and the
  egg's `file_denylist` applies to where a link really points. The editor reads at most its limit plus one
  byte, listings at most 16 MiB, error output 64 KiB. Archives are extracted into a temporary folder and merged
  from there: `../` entries and links inside the archive cannot write outside its folder.
- **Audit log:** sign-ins (also failed ones, with the client IP), file reads, downloads and changes, power
  actions, console commands, user changes (the changed fields, never a password) and every other change are
  logged with the user and, for API tokens, the token ID (`token=`). Install pods run as root with a small set of capabilities. User namespaces enforce
  the Pod Security `baseline` level.
- **Kubernetes permissions:** see [Running in the cluster](installation.md#running-in-the-cluster): writes only in the panel's own namespaces, checked by an
  admission policy.

Run the panel behind TLS (reverse proxy / Gateway) when it is reachable from outside.
