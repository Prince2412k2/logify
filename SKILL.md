---
name: logify
description: Inspect Coolify-managed projects via the `logify` CLI — list services with health and latest deploy status, tail runtime or build logs. Triggers when the user asks "why is X down/failing/unhealthy", "check the latest deploy", "tail logs", "watch this deploy", "show me errors in X", or when the working directory contains a `.logify` file.
---

# logify

Read-only CLI for Coolify-managed services. Three commands: **`list`**, **`logs`**, **`bind`**.

**Output is plain text by default for every command.** Dense, line-oriented, the same shape LLMs read fastest in training data. Pass the global `--json` flag (before the subcommand) for structured JSON / NDJSON when you need to pipe into other tooling.

Exit codes carry meaning. Designed for agentic use.

## First action

Run `logify` (no args) to discover the command manifest and current project binding:

```
$ logify
logify 0.1.0

bound:   opex (id=4)
config:  /repo/.logify

commands:
  logify list
      List services in the bound project (or every accessible service with --all).
      flags: --all
  logify logs service
      Runtime or build logs for one service.
      flags: --build --deployment --tail --level --grep --follow --max
  logify bind project?
      Bind the working directory to a Coolify project.
      flags: --remove --list
  logify unbind
      Alias for `bind --remove`.

global flags: --url --token --config --json
add --json (before the subcommand) for structured output.
```

(For programmatic discovery, `logify --json` returns the same data as a JSON manifest.)

Branch on `context.bound`:

| `bound` | What it means | Behavior |
|---|---|---|
| `true` | The cwd is bound to one Coolify project | `list` shows only that project; `logs <svc>` resolves bare service names within that project |
| `false` | No `.logify` found | `list` requires `--all`; `logs` requires a full path or uuid |

Never invent commands or flags. The manifest is the source of truth.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 2 | Bad input (unknown command, bad flag) |
| 3 | Not found (no service or project matches) |
| 4 | Ambiguous identifier — `error.candidates` lists options |
| 5 | Auth failed |
| 6 | Network / gateway error |
| 7 | Not allowed (scope check denied) |
| 1 | Other |

Always check the exit code before parsing. On non-zero exit, stdout contains a text block of the form:

```
error: <CODE> — <message>
hint:  <suggested next step>
candidates:                   # only present on AMBIGUOUS (exit 4)
  Innovation Hub/production/frontend  gow408skskg0sc8o804w0s0k
  opex/production/frontend            g08880cscc0w0wgc8sc4og44
```

With `--json`, the same payload comes as `{"error":{"code","message","hint"?,"candidates"?}}`. The code list (NOT_FOUND, AMBIGUOUS, AUTH, NETWORK, NOT_ALLOWED, NOT_REACHABLE, BAD_INPUT) is stable and grep-friendly.

## The three verbs

### `logify list [--all]`

Shows every service in the bound project, each row enriched with runtime health and the latest deployment status. Without `--all`, an unbound dir gets `NOT_FOUND`.

```
NAME                    TYPE           HEALTH              DEPLOY        AGE        COMMIT     MESSAGE
backend                 application    running             finished      2h ago     141c5c1    GRN status
frontend                application    running             finished      3h ago     3336149    UI changes - scroll - wrap fixes
postgres                service        running             —             —          —
```

**This is the primary investigation snapshot.** A single `logify list` answers "is anything broken right now". For each row inspect:

- `HEALTH` — Docker runtime status; anything other than `running` is a smoking gun (`unreachable` means the gateway can't see the container).
- `DEPLOY` — `failed` or `cancelled-by-user` means the last deploy didn't land.
- `COMMIT` + `MESSAGE` — useful for citing the user back.

Services (databases, redis) show `—` in DEPLOY columns because Coolify doesn't queue deploys for them — that's expected, not an error.

`--json` returns an array of `{name, uuid, project, path, type, container_name, health, latest_deployment:{...}}`.

### `logify logs <service> [flags]`

Both runtime and build logs live here. Text by default; `--json` (global, before the subcommand) opts into structured form.

| Flag | Effect |
|---|---|
| `--build` | Show the latest build (deployment) log instead of runtime |
| `--deployment <uuid>` | (Reserved — currently rejected; gateway returns latest only) |
| `--tail N` | Snapshot line count, runtime only (default 200) |
| `--level err,warn,info,debug` | Runtime level filter |
| `--grep <pattern>` | Substring filter, applied to runtime entries or build lines |
| `--follow` | Stream until `--max` (runtime) or until deploy reaches terminal state (`--build`) |
| `--max <dur>` | Bound `--follow` (e.g. `60s`, `10m`). Always pass this in agentic loops — otherwise the call runs forever. |

**Service identifier rules** (in order):

1. Exact UUID
2. Exact container name
3. Exact `project/stage/service` path
4. Bare service name — unique within the bound project (or globally if `--all` listing showed it as unique)

`AMBIGUOUS` (exit 4) returns `candidates` with `path` and `uuid`; retry with either.

#### Runtime snapshot output (default, plain text)

```
13:42:14 WARN  slow query 1.2s
13:42:15 ERROR connection refused: dial tcp 10.4.1.3:5432
13:42:15 ERROR     at db.go:88 (*Pool).Acquire
```

Format per line: `HH:MM:SS LEVEL message`. Level is fixed-width 5 chars. Pass `--json` for `[{ts, level, msg}, …]`.

#### Runtime follow output (`--follow`, plain text)

Same per-line format streamed as the gateway emits lines. Ends after `--max` elapses or the gateway closes the connection. With `--json --follow`, NDJSON.

#### Build snapshot output (`--build`, plain text)

```
deployment: to44k8o80os4cg4wcgos8gcs
status:     finished
commit:     141c5c1 — GRN status
started:    2026-04-29T13:54:54Z
duration:   87s

Starting deployment of Gateway-Digital/Opex-Backend:main to localhost.
#1 [internal] load build definition from Dockerfile
#1 transferring dockerfile: 1.85kB done
...
```

A small header (deployment metadata) then the build lines raw. With `--json`, returns `{deployment, status, commit, started_at, duration_seconds, lines: [...]}`.

#### Build follow output (`--build --follow`, plain text)

Status transitions are bracketed; build lines stream raw:

```
[deploy.queued] ncw0sgos08wc
[deploy.in_progress] ncw0sgos08wc
#6 [4/6] RUN npm ci
#6 DONE 18.1s
[deploy.finished] ncw0sgos08wc
```

Exits 0 when the deploy reaches a terminal state (`finished`, `failed`, `cancelled-by-user`, `cancelled`) or when `--max` elapses. With `--json --follow`, NDJSON events.

### `logify bind [<project>]` / `logify unbind`

Binds the working directory to one Coolify project. Agents pass `<project>` explicitly; humans omit it for an interactive numbered prompt (only on a TTY).

```
logify bind opex                  # set
logify bind --list                # current binding as JSON
logify unbind                     # remove (alias for bind --remove)
```

Agents rarely set bindings — they read `context.project` from the manifest. The user sets them.

## Decision tree

### "Is anything broken?"

```
logify list
```

Scan `health` for non-`running` rows. Scan `latest_deployment.status` for `failed`. Report.

### "Why is X failing?"

1. `logify list` (JSON) — confirm X is unhealthy or recent deploy failed.
2. If deploy failed: `logify logs X --build` (text) — read the tail of the build for the failure point.
3. If runtime unhealthy: `logify logs X --level err,warn --tail 200` (text) — read recent errors.

### "Watch this deploy until done"

```
logify logs X --build --follow --max 1800s
```

Read the stream line-by-line. Lines look like `[deploy.queued] ncw0…` for status events and raw build text otherwise. Stop on the first `[deploy.finished]`, `[deploy.failed]`, `[deploy.cancelled-by-user]`, or `[deploy.cancelled]` event. Surface that status to the user, plus the last 20 build lines if it failed.

### "Tail errors live"

```
logify logs X --follow --level err,warn --max 600s
```

One log line per output line until the duration elapses.

## Don'ts

- **Don't poll without `--max`.** `--follow` blocks indefinitely otherwise.
- **Don't invent flags.** Read `commands[].flags` from the manifest.
- **Don't expect env-var values.** They are not exposed by the CLI. (Configuration data is not surfaced in this version either; reach for Coolify's UI for that.)
- **Don't assume a service has builds.** Services (databases, redis, etc.) return an empty `lines` array on `logs --build`. That's correct.
- **Don't retry on exit 5 (AUTH) or 7 (NOT_ALLOWED).** Surface to the user.

## A worked example

User, in `/repo/api` (`.logify` binds to project `opex`): "Frontend is broken, what gives?"

```
$ logify list
NAME                    TYPE           HEALTH              DEPLOY        AGE        COMMIT     MESSAGE
backend                 application    running             finished      2h ago     141c5c1    GRN status
frontend                application    running:unhealthy   finished      3h ago     3336149    UI changes - scroll - wrap fixes
```

Frontend is `running:unhealthy` despite a successful deploy. Look at runtime errors next:

```
$ logify logs frontend --level err,warn --tail 200
13:42:15 ERROR connection refused: dial tcp 10.4.1.3:5432: connect: connection refused
13:42:15 ERROR     at db.go:88 (*Pool).Acquire
```

Diagnosis report to user:

> Frontend's latest deploy (commit `3336149`: "UI changes - scroll - wrap fixes") finished cleanly, but the running container is `running:unhealthy` because it can't reach Postgres at `10.4.1.3:5432`. The error fires at `db.go:88` in the pool's `Acquire`. Probably a network or DB-host change since the last deploy — check Postgres reachability from the frontend's network.

Two commands, complete picture.

## Output cheat sheet

All commands emit **plain text by default**. Pass `--json` (global, before the subcommand) for structured output.

| Command | Text shape | `--json` shape |
|---|---|---|
| `logify` | Manifest + binding context, formatted | JSON object |
| `logify list` | Aligned table: NAME · TYPE · HEALTH · DEPLOY · AGE · COMMIT · MESSAGE | JSON array of services |
| `logify bind …` | `bound to <name> (id=<id>)\nsaved to <path>` | JSON `{config_path,project,project_id}` |
| `logify logs X` | `HH:MM:SS LEVEL msg` per line | JSON array |
| `logify logs X --build` | Small header + raw build lines | JSON snapshot object |
| `logify logs X --follow` | Per-line text stream | NDJSON |
| `logify logs X --build --follow` | `[deploy.STATE] dep…` events + raw build lines | NDJSON events |
| Any non-zero exit (text) | `error: CODE — message\nhint: …\ncandidates: …` | `{"error":{...}}` |
