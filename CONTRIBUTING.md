# Contributing

Thanks for improving NodePulse.

## Before you start

- Open an issue for new features or larger refactors so the scope and product fit can be discussed first.
- Small fixes (typos, docs, tests, obvious bugs) can go straight to a pull request.
- Never commit secrets, `settings.json`, `results.json`, signed download URLs, real subscription links, or screenshots containing personal node/provider names.

## Development

See [docs/development.md](docs/development.md).

```bash
go test ./...
go vet ./...
cd desktop && flutter analyze && flutter test
```

## Pull requests

- Keep changes focused; avoid unrelated formatting or renames.
- Add or update tests for behavior changes.
- Update docs when commands, configuration, APIs or UI flows change.
- Use Conventional Commits: `feat:`, `fix:`, `docs:`, `test:`, `build:`, `refactor:`.
- Describe the user-visible effect and the verification you ran.

## Product boundaries

- NodePulse does not proxy traffic, implement TUN/DNS, or replace a Clash client.
- Automatic selection must never manage `GLOBAL`, and must verify switches with rollback.
- Download mode selects a path before download; it must not switch nodes mid-download.
- Site-specific download-link scraping is out of scope.
- Android is built from the same Flutter project (`make android`); iOS is not provided.

## License

By contributing, you agree that your contribution is licensed under GPL-3.0.
