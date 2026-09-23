# Skillc

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/inhere/skillc?style=flat-square)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/inhere/skillc)](https://github.com/inhere/skillc)
[![Unit-Tests](https://github.com/inhere/skillc/actions/workflows/go.yml/badge.svg)](https://github.com/inhere/skillc)

---

[English](./README.md) | [简体中文](./README.zh-CN.md)

`skillc` Single binary file, a local skills management tool for the multi-Agent ecosystem.

## Features

- 📦 **Multi-source management** — Local paths and Git repositories as Skill sources
- 🔍 **Index & search** — Auto-scans sources and builds a searchable index
- ⚡ **One-shot install** — `--source` flag registers, syncs, and installs in a single command
- 🧩 **Profiles** — Save a reusable Skill set and apply it to any project with a dry-run plan
- 🔒 **Lock file tracking** — Records origin, version, and install path; supports `restore`
- 🤖 **Multi-agent adapters** — Automatically targets each agent's install directory
- 🔄 **Batch update** — Update all installed Skills with one command
- ⌨️ **Interactive selection** — Filter and multi-select Skills for install, update, and profile creation

## Installation

**Install by eget**

Can quickly install by [inherelab/eget](https://github.com/inherelab/eget)

```bash
eget install inhere/skillc
```

**Install by Go**

```bash
go install github.com/inhere/skillc/cmd/skillc@latest
```

**Build from source**

```bash
git clone https://github.com/inhere/skillc
cd skillc
make build          # compile to current directory
make install        # install to $GOPATH/bin
```

## Quick Start

```bash
# 1. Initialize config
skillc config init

# 2. Add a Skill source (Git repo or local path)
skillc source add https://github.com/org/skills.git --id org-skills --name "Org Skills"
skillc source add /path/to/my-skills --id my-skills

# 3. Sync sources (clone/pull and rebuild index)
skillc source sync --all

# 4. Search for a Skill
skillc search typescript

# 5. Install a Skill
skillc install my-skill

# 6. List installed Skills
skillc list
```

## Command Reference

### `config` — Configuration

```bash
skillc config init               # initialize config file
skillc config show               # display current config
skillc config set <key> <value>  # update a config value
```

### `source` — Source management

```bash
skillc source list                    # list all sources
skillc source add <path-or-git-url> [--id <id>] [--name <name>] [--ref <ref>] [--sync]
skillc source add git <url> [ref] [--id <id>] [--name <name>] [--sync]
skillc source add local <path> [--id <id>] [--name <name>] [--sync]
skillc source info <id>               # show source details (partial ID match supported)
skillc source sync <id>               # sync a source (partial ID match supported)
skillc source sync --all              # sync all sources
skillc source status                  # show source status
skillc source collections [source]    # list collections under sources
skillc source skills <source>         # list skills under a source
skillc source skills <source> --collection <name>
skillc source remove <id>             # remove a source
```

> New generated source IDs no longer receive forced `local-` / `git-` prefixes. Existing configured IDs are left unchanged.
> `source sync` and `source info` support **partial ID matching** — e.g. `skillc source sync edge` matches `golang-edge-skills`.

### `registry` — Discover and install registry Skills

```bash
skillc registry list
skillc registry add <path-or-url> --id official --name "Official Registry"
skillc registry add https://skillsmp.com --id skillsmp --name SkillsMP --provider skillsmp
skillc registry sync <id>
skillc registry sync --all
skillc registry search go                         # search Skill results by default
skillc registry search go --registry skillsmp     # remote search via SkillsMP provider
skillc registry info official/go-pro              # show a registry Skill result
skillc registry install official/go-pro --agent codex --scope project --yes
skillc registry search gstack --kind source       # inspect source catalog entries
skillc registry add-source official/gstack [--id <id>] [--name <name>] [--sync]
```

Registry providers discover Skill-level results from a generic JSON catalog or a built-in provider adapter. `registry install` materializes one Skill into the local registry cache and then uses the normal install/lock flow with `source_type=registry` provenance. `registry add-source` is optional and only turns a source result into a long-lived source subscription.

SkillsMP is the first built-in provider adapter. Unlike generic JSON registries, it performs remote keyword search and caches returned Skill results locally so `registry info` and `registry install` can reuse the normal Registry install flow.

Registry Skill entries can point at a Git/local `source_url` or an archive `download_url`. Archive downloads support `zip`, `tar.gz`, and `tgz`; `checksum` verifies the original archive bytes with SHA-256 before extraction:

```json
{
  "skills": [
    {
      "id": "go-pro",
      "name": "Go Pro",
      "version": "1.0.0",
      "download_url": "https://example.com/skills/go-pro.zip",
      "checksum": "sha256:<archive-sha256>",
      "install_entry": "skills/go-pro"
    }
  ]
}
```

### `profile` — Saved Skill sets

```bash
skillc profile list                              # list saved profiles
skillc profile show <name>                       # show profile details
skillc profile create <name> --from-installed    # create from current installed Skills
skillc profile create <name> --from-collection <source>/<collection>
skillc profile create go-dev --interactive       # pick Skills interactively
skillc profile diff <name>                       # preview profile apply plan
skillc profile apply <name> --dry-run            # print plan without installing
skillc profile apply <name> --yes                # apply profile without confirmation
```

### `project` — Registered projects

```bash
skillc project list                              # list registered projects
skillc project add . --id my-project --name "My Project"
skillc project remove my-project
skillc project import-lock                       # import existing project paths from lock records
```

Registered projects are the explicit allowlist used by Web cross-project updates and `update --all-projects`.

### `install` — Install Skills

```bash
skillc install <skill-id>             # install a Skill
skillc install <id1>,<id2>            # install multiple (comma-separated)
skillc install                        # restore all Skills from lock file
skillc install --interactive [keyword] # filter and multi-select Skills

# One-shot: register source → sync → install
skillc install --source https://github.com/org/skills.git my-skill
skillc install --source /path/to/local-skills my-skill
```

**Options:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--scope` | `-s` | `project` | Install scope (`project` / `global`) |
| `--agent` | `-a` | `claude-code` | Target agent name or directory |
| `--yes` | `-y` | `false` | Skip confirmation prompt |
| `--source` | `-S` | | Git URL or local path — auto-register & sync before installing |
| `--interactive` | `-i` | `false` | Open an interactive Skill selector |
| `--install-mode` | | | Install mode (`symlink` / `junction` / `copy`) |
| `--copy` | | `false` | Install by copying files |
| `--force` | `-f` | `false` | Overwrite an existing install (another source, or local changes); the replaced directory is backed up first |

Interactive selection uses `gookit/cliui`: type to filter the candidate list, press Space to multi-select, and press Enter to continue into the normal install confirmation and execution flow.

### `update` — Update Skills

```bash
skillc update                        # update all installed Skills
skillc update --target <skill-id>    # update a specific Skill
skillc update --check                # preview update candidates without installing
skillc update --interactive          # filter and multi-select update candidates
skillc update --all-projects --check # preview registered project updates
skillc update --all-projects --projects my-project,api --target go-pro --yes
skillc update --force                # overwrite the whole installed directory (backup first)
skillc update --no-merge             # skip locally modified skills instead of merging them
```

`update --check` and `status` report precise drift when the version is unchanged but the source metadata changed: Git sources compare resolved refs, and local sources compare directory-level checksums.

### Per-file merge

Copy installs record a file manifest (`installed_files`) next to the deployed fingerprint, so `update` plans a three-way merge by default between the source content at install time, the installed directory, and the current source content:

| Local | Upstream | Result |
|-------|----------|--------|
| unchanged | changed | upstream content is written |
| changed | unchanged | local content is kept |
| changed | changed | conflict: local is kept and the upstream version is written as `<file>.incoming` |
| file added locally | - | kept |
| file added upstream | - | written |
| file deleted locally | - | stays deleted |
| unchanged | deleted upstream | deleted |

`update --merge --force` resolves conflicts in favour of upstream after backing up the whole installed directory. `--no-merge` disables merging (locally modified skills are skipped, `--force` overwrites them), and a bare `--force` keeps the whole-directory overwrite semantics. Skills without a manifest (installed by older versions, or link installs) fall back to the overwrite/backup behaviour.

### Local change protection

Copy installs (`install_mode: copy`) keep a deployed fingerprint (`installed_checksum`) in the lock file. `install`, `update` and `uninstall` compare the installed directory against it before touching it:

- content differs from the fingerprint → the command skips the skill and reports `locally modified`; pass `--force` to overwrite (or remove) it;
- no fingerprint yet (records created by older versions, or directories skillc did not install) → the directory is snapshotted into `backup_dir` before it is replaced;
- `--force` always snapshots the replaced directory first, and `update` prints the backup path;
- link installs (`symlink` / `junction`) are never overwritten or backed up, because the project directory *is* the source directory.

`skillc status` and `skillc update --check` mark such skills in their `Reason` column, and the summary prints a `modified` count.

Git sources are synced by `git fetch` + `reset --hard` + `clean -fd` in the repository cache. `source sync` (and therefore `update`) now refuses to sync when that cache has local changes, instead of silently discarding them; inspect it with `git -C <repo_cache_dir>/<source-id> status` and commit or discard the changes before syncing.

### Project Registry and Cross-Project Updates

`skillc project` registers local projects that are allowed to be managed by Web and `update --all-projects`. Cross-project updates only operate on registered projects; they do not blindly scan unknown lock entries. Use `skillc project add . --id <id>` or `skillc project import-lock`, inspect with `skillc update --all-projects --check`, then execute with `skillc update --all-projects --yes`.

### `diff` — Installed vs source

```bash
skillc diff <skill-id>               # per-file table plus unified diffs
skillc diff <skill-id> --no-patch    # file table only
```

`diff` compares the installed directory with the source skill directory using the recorded manifest and reports, per file, whether the local copy and the upstream copy changed relative to the install baseline (`same` / `modified` / `added` / `deleted` / `absent`). Conflicts — what `update --merge` would resolve by keeping local and writing `<file>.incoming` — are marked, and a `git diff --no-index` patch is printed for files present on both sides.

### `adopt` — Write local changes back to the source

```bash
skillc adopt <skill-id>              # write the installed skill's local changes into its source
skillc adopt <skill-id> --dry-run    # print the plan only
skillc adopt <skill-id> --yes        # skip the confirmation prompt
```

`adopt` compares the installed directory with the recorded file manifest and copies the local versions back into the source skill directory, so the change can be committed in the skills repository. The source directory is backed up first, the source index is rebuilt, and the lock baseline is refreshed (the skill stops being reported as `modified`). Files deleted locally are reported for manual deletion instead of deleting source files, and Git sources (cache clone) and registry skills are refused.

### `status` — Skill health

```bash
skillc status                        # show current project skill status
skillc status --profile go-dev       # filter by profile
skillc status --agent claude-code    # filter by agent
```

`status` also reports skills whose installed directory no longer matches the recorded deployed fingerprint (`modified` in the summary).

### `web` — Local management UI

```bash
skillc web
skillc web --port 8090
skillc web --host 127.0.0.1 --port 8090
```

The web manager runs on `127.0.0.1` by default and supports source/profile/status/install-map/version-drift views, Registry search/sync/install/add-source, guarded current-project profile apply and update, source add/sync/remove, profile save/from-installed/from-collection, uninstall, and registered-project cross-project update plan/run. Every Web write action is plan-first, requires `confirm:true` on the run request, appends a local `skillc-web-history.jsonl` record, and only operates on the current project or explicitly selected registered projects.

The action bar has `force` (overwrite locally modified skills after backing them up) and `merge` (per-file merge) checkboxes for update and uninstall, and the status view marks locally modified skills.

Open the Registry view in `skillc web` to search synced registry Skills, preview install plans, install a registry Skill into the current project, sync registry catalogs, or convert a registry source result into a configured source.

Version Drift also exposes checksum and Git ref signals, so same-version content changes are visible in Web alongside ordinary version differences.

### `uninstall` — Remove Skills

```bash
skillc uninstall <skill-id> [...]    # uninstall one or more Skills
skillc uninstall --force <skill-id>  # also remove a skill that has local changes
```

### `list` — Installed Skills

```bash
skillc list                          # list installed Skills (current agent)
skillc list --scope global           # list globally installed Skills
```

### `search` / `show` — Index search

```bash
skillc search <keyword>              # keyword search
skillc search <keyword> --agent claude  # filter by agent
skillc show <skill-id>               # show Skill details
```

Collections are browsed through `source collections` and `source skills --collection`; to reuse a collection as a project Skill set, create a profile from it and apply the profile.

### `doctor` — Environment check

```bash
skillc doctor                        # verify git, config, index, and cache
```

## Configuration

Config file lookup order:

1. `./skillc.yaml` (current directory)
2. `~/.config/skillc/config.yaml`

Key fields:

```yaml
lock_file: skillc.lock.yaml        # lock file path
index_file: skillc-index.json      # index file path
repo_cache_dir: ~/.cache/skillc    # Git repo cache directory
backup_dir: ~/.cache/skillc/backups # snapshots taken before replacing an installed directory
proxy_url: ""                      # HTTP proxy (optional)
sources: []                        # registered sources
projects: []                       # registered local projects for cross-project updates
agent_tools:                      # agent tools config agent_name: config
  claude-code:
    dirname: .claude
    aliases:
    - claude
  codex:
    dirname: .codex
  opencode:
    # dirname: .opencode # default is .{agent_name}
    user_dir: ~/.config/opencode
  universal: # universal agent config, most agent tool support this
    dirname: .agents
    aliases:
    - agents
    user_dir: ~/.agents
    project_dir: .agents
```

## Lock File

`skillc.lock.yaml` records every installed Skill and is used by `skillc install` (no args) to restore all Skills:

```yaml
records:
  - skill_id: my-skill
    source_id: git-org-skills
    agent: claude-code
    scope: project
    installed_path: .claude/skills/my-skill
    install_mode: copy
    installed_checksum: 9f2c...   # deployed directory fingerprint, detects local changes
    installed_at: "2026-01-01T00:00:00Z"
```

## Development

```bash
go test ./...    # run all tests
make build       # local build
```

## License

MIT
