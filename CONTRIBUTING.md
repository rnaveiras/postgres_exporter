# Contributing

## Setup

Tools are pinned in `mise.toml`. Install [mise](https://mise.jdx.dev), then from the repository root:

```sh
mise install          # Go, golangci-lint, hk, pitchfork, linters, promtool, promu
hk install --mise     # git hooks that resolve tools through mise
```

If you used the old pre-commit hooks, remove them first with `pre-commit uninstall`.
On git 2.54+ you can instead run `hk install --global --mise` once per machine.

## Everyday commands

| Command | What it does |
|---|---|
| `mise run check` | Every linter on all files. CI runs exactly this. |
| `mise run fix` | Every fixer on modified files. |
| `mise run lint` | golangci-lint only. |
| `mise run lint:fix` | golangci-lint fixes and formatters. |
| `mise run modernize` | Apply the `go fix` modernizers. |
| `mise run test` | Unit tests with `-race -shuffle=on`. |
| `mise run test:integration` | Integration tests against one Postgres major (`POSTGRES_MAJOR`, default 18). |
| `mise run test:matrix` | Integration tests against 14 to 18 in turn. |
| `mise run build` | Build the binary into `.build/` with promu. |
| `mise run tarball` | Build the release tarball into `.build/`. |
| `mise run docker` | Build the container image. |

`mise tasks` lists everything.

## Git hooks

`pre-commit` runs the hygiene checks and the Go linters and fixes what it can. `pre-push` adds
`govulncheck`. Run the slow steps locally with `HK_PROFILE=slow mise run check`.

Escape hatches, for when you really need them:

```sh
HK_SKIP_STEPS=renovate-config mise run check   # offline: the validator needs the network on first run
HK_SKIP_HOOK=pre-push git push
HK=0 git commit
```

Personal hook overrides go in `hk.local.pkl` (gitignored), starting with `amends "./hk.pkl"`.

## Local stack

[pitchfork](https://pitchfork.jdx.dev) runs Postgres, the exporter (from source, restarted on every Go
change), Prometheus and Grafana in the background. Docker is required.

```sh
mise run stack:up                 # Postgres 18 + exporter + Prometheus + Grafana
PGEXP_PG=16 mise run stack:up     # same, scraping Postgres 16
mise run stack:matrix             # Postgres 14 to 18 side by side
mise run stack:beta               # Postgres 19 beta
mise run stack:down               # stop everything
pitchfork logs exporter-pg18      # follow one daemon
```

| Service | Address |
|---|---|
| Postgres `<major>` | `127.0.0.1:54<major>` (5414 to 5419), user `postgres`, password `postgres` |
| Exporter | http://127.0.0.1:9187/metrics |
| Prometheus | http://127.0.0.1:9090 |
| Grafana | http://127.0.0.1:3000 (admin / admin) |

The exporter connects as the least-privilege `postgres_exporter` role, created on every start by
`docker/sql/monitoring-role.sql`. Only one exporter runs at a time because they all listen on 9187.

If a supervisor crash leaves a container behind, `pitchfork start` fails with a name clash. Clean up with:

```sh
docker ps -a --filter name=pgexp- -q | xargs -r docker rm -f
```

Personal overrides (ports, image tags) go in `pitchfork.local.toml` (gitignored).

`compose.yaml` describes the same stack for one-shot runs. Each Postgres major is a profile and
listens on its own host port, so profiles can be combined:

```sh
docker compose --profile default --profile postgres18 up -d
```

## Private test dependency

The integration tests use `github.com/rnaveiras/pgfresh`, a private module. `mise.toml` sets
`GOPRIVATE` and `GONOSUMDB` for it. Locally, git needs credentials for that repository, for example:

```sh
gh auth setup-git
# or, with SSH keys:
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

In CI the `PGFRESH_TOKEN` repository secret holds a fine-grained token with read-only Contents access to
that one repository. Secrets are not available to pull requests from forks, so those run the linters,
unit tests and build, and show the integration checks as skipped. A maintainer can push the branch to
this repository to run them.

## Lint rules

The golangci-lint configuration lives in `.golangci.yaml`; every `//nolint` must name the linter and give
a reason. To check that the SQL rule fires, add `fmt.Sprintf("SELECT %s FROM t", col)` to any file in
`collector/` and run `mise run lint`: forbidigo must report it.

After a Go upgrade, run `mise run modernize` and commit the result on its own.
