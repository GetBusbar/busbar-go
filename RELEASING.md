# Releasing busbar-go

This repo has two separate, deliberately-decoupled mechanisms: keeping `dev`
in sync with busbarAI, and cutting a version tag on `main`. Don't confuse them.

`main` is a protected, PR-only branch — nothing pushes to it directly, ever,
including the sync bot. It only advances via a reviewed pull request from
`dev`, opened at an actual release cut (see §2).

## 1. `dev` is auto-synced with busbarAI's `dev` branch

`.github/workflows/regen-from-upstream.yml` polls
`GetBusbar/busbar`'s `dev` branch every 15 minutes for changes to
[`crates/busbar/src/admin/v1/json/openapi.json`](https://github.com/GetBusbar/busbar/blob/dev/crates/busbar/src/admin/v1/json/openapi.json).
When that file changes, the workflow:

1. Overwrites this repo's committed `openapi.json` with the new spec.
2. Runs `make generate` to re-derive `client.gen.go`.
3. Runs `make verify` (`go build ./... && go vet ./...`).
4. If (and only if) that's all green, commits both files and pushes straight
   to `dev` — never `main`.

If regeneration or verification fails (e.g. the new spec breaks something
that needs a manual fix, like a renamed operation used in `examples/`), the
workflow run fails loudly and does **not** push a broken commit — `dev`
stays on the last known-good sync until someone fixes it by hand.

This means **`dev` tracks busbarAI's `dev`, which is unstable by definition.**
A commit landing on this repo's `dev` via this workflow is not a release and
does not mean the SDK is ready to depend on at a pinned point — it means the
generated code is consistent with *some* snapshot of busbarAI's in-progress
spec. `main` is unaffected until someone deliberately promotes it (§2).

**Why polling, not a webhook:** `busbar-go` doesn't control `GetBusbar/busbar`'s
CI, so a true push-triggered `repository_dispatch` would require adding a step
to `busbar's` own workflows plus a cross-repo PAT stored as a secret in that
repo. No such token exists yet in either repo (checked: `busbar-go` has no
secrets at all; `busbar`'s only repo-level secrets are Docker Hub credentials
and an unrelated, currently-unused `PULLS_WRITE_SECRET` not wired into any
workflow). Both repos are public, so this workflow reads the upstream spec
anonymously via `raw.githubusercontent.com` — no token is needed for that
half either. Given no existing cross-repo credential to build on, a tight
polling interval (15 min, vs. headroom-hook's daily cadence) is the
self-contained fallback: no new secret, no changes required in the `busbar`
repo. If a cross-repo dispatch token is added later, this workflow's `on:`
block is the only thing that needs to change.

## 2. Version tags are cut manually, not automatically

Tagging `v0.2.0`, `v0.3.0`, etc. is **not** wired into the sync workflow on
purpose. `dev` moves continuously; tagging every auto-synced commit would
mean cutting a new `go get`-able SDK version many times a day, which is bad
semver practice and would make the tag list meaningless.

**Cut a release by hand when busbarAI itself ships a real, tagged release** (i.e.
a `main`/release build of busbarAI at a specific `info.version`, not just a
`dev` snapshot):

1. Confirm `busbar-go`'s `dev` is already synced to the spec matching that
   busbarAI release (check the latest `sync:` commit's `openapi.json`
   `info.version`, or just wait for the next poll — at most 15 minutes).
2. Open a PR from `dev` -> `main` (`gh pr create --base main --head dev`).
   Review and merge it — this is the one point a human (or an explicitly
   authorized agent) actually looks at what's being promoted.
3. Update the "Versioning" section of `README.md` if the reported
   `info.version` changed.
4. On `main`: `git tag vX.Y.Z -m "busbar-go vX.Y.Z — generated from busbar <version> spec"`
5. `git push origin vX.Y.Z` — this triggers `.github/workflows/release.yml`,
   which re-validates that the tagged commit regenerates cleanly and builds.

There is no registry token for Go modules — "publishing" a tag is exactly
`go get github.com/GetBusbar/busbar-go@vX.Y.Z` becoming resolvable, which
happens the moment the tag exists on GitHub.
