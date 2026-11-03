# Security Policy

## Supported versions

Security fixes are applied to the latest release and `main`.

## Reporting

Please report vulnerabilities privately through GitHub Security Advisories:

<https://github.com/HanphoneJan/nodepulse/security/advisories/new>

Do not open a public issue for a vulnerability that could expose secrets, local files, or the management API.

Include:

- affected version/commit;
- steps to reproduce;
- impact;
- any suggested fix.

## Scope

NodePulse is a local desktop tool. The web UI binds to `127.0.0.1` by default and has no remote authentication. Do not expose it to a LAN or the public internet without adding authentication and TLS.

Relevant areas:

- `settings.json`, controller secret, subscription URLs and signed download URLs;
- local Web API (`/api/*`) and same-origin protection;
- import/download request handling (`internal/netfetch`);
- result/history/probe cache files;
- daemon lock and file permissions;
- package scripts and CI secrets.

## Practices

- Secrets are stored with `0600`; backup/cache directories use `0700`.
- API responses set `Cache-Control: no-store`.
- Subscription downloads allow only HTTP/HTTPS and reject link-local/cloud-metadata targets.
- Signed download URLs and secrets are not written to history or logs.
- Do not include real subscription data in bug reports; redact node names and URLs.
