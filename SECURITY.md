# Security policy

Kubedactyl is meant to be reachable from the internet, so security reports are welcome.

## Reporting a vulnerability

Please do not open a public issue. Use **Report a vulnerability** on the repository's *Security* tab (GitHub private
vulnerability reporting) and include the affected version (footer of the panel or `GET /api/info`), the steps to
reproduce and what an attacker gains.

You will get an answer within a few days. Fixes are released as a new chart version, which panels with self-upgrades
enabled offer on their *Settings* page.

## Supported versions

Only the latest release gets fixes. Upgrade before reporting if you can.

## Scope

The measures the panel takes are described in [docs/security.md](docs/security.md). Game servers run code their owners
upload (plugins, mods); the panel isolates them with a NetworkPolicy, Pod Security `baseline` and pods without
service account tokens, but it does not inspect what they run.
