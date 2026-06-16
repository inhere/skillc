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

Interactive selection uses `gookit/cliui`: type to filter the candidate list, press Space to multi-select, and press Enter to continue into the normal install confirmation and execution flow.

### `update` — Update Skills

```bash
skillc update                        # update all installed Skills
skillc update --target <skill-id>    # update a specific Skill
skillc update --check                # preview update candidates without installing
skillc update --interactive          # filter and multi-select update candidates
skillc update --all-projects --check # preview registered project updates
skillc update --all-projects --projects my-project,api --target go-pro --yes
```

### Project Registry and Cross-Project Updates

`skillc project` registers local projects that are allowed to be managed by Web and `update --all-projects`. Cross-project updates only operate on registered projects; they do not blindly scan unknown lock entries. Use `skillc project add . --id <id>` or `skillc project import-lock`, inspect with `skillc update --all-projects --check`, then execute with `skillc update --all-projects --yes`.

### `status` — Skill health

```bash
skillc status                        # show current project skill status
skillc status --profile go-dev       # filter by profile
skillc status --agent claude-code    # filter by agent
```

### `web` — Local management UI

```bash
skillc web
skillc web --port 8090
skillc web --host 127.0.0.1 --port 8090
```

The web manager runs on `127.0.0.1` by default and supports source/profile/status/install-map/version-drift views, guarded current-project profile apply and update, source add/sync/remove, profile save/from-installed/from-collection, uninstall, and registered-project cross-project update plan/run. Every Web write action is plan-first, requires `confirm:true` on the run request, appends a local `skillc-web-history.jsonl` record, and only operates on the current project or explicitly selected registered projects.

### `uninstall` — Remove Skills

```bash
skillc uninstall <skill-id> [...]    # uninstall one or more Skills
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
    installed_at: "2026-01-01T00:00:00Z"
```

## Development

```bash
go test ./...    # run all tests
make build       # local build
```

## License

MIT
