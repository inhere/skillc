# Skillc

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/inhere/skillc?style=flat-square)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/inhere/skillc)](https://github.com/inhere/skillc)
[![Unit-Tests](https://github.com/inhere/skillc/actions/workflows/go.yml/badge.svg)](https://github.com/inhere/skillc)

---

[English](./README.md) | [简体中文](./README.zh-CN.md)

`skillc` Go编写的单二进制文件，面向多 Agent 生态的本地 skills 管理工具

Skillc 统一管理 `claude-code`、`opencode`、`codex` 等 AI Agent 的 Skills（提示词、规则文件等），屏蔽各 Agent 的目录差异，提供一套一致的 CLI 体验。

## 特性

- 📦 **多来源管理** — 支持本地路径与 Git 仓库作为 Skill 来源
- 🔍 **索引与搜索** — 自动扫描来源并建立索引，支持关键词搜索
- ⚡ **一键安装** — 支持 `--source` 选项同时完成来源注册、同步和安装
- 🧩 **Profile 技能组合** — 保存一组可复用 Skills，并在任意项目预览后应用
- 🔒 **锁文件追踪** — 记录每个 Skill 的来源、版本和安装位置，支持 `restore`
- 🤖 **多 Agent 适配** — 自动适配不同 Agent 的安装目录规范
- 🔄 **批量更新** — 一条命令更新所有已安装的 skills
- ⌨️ **交互式选择** — 安装、更新、创建 Profile 时支持过滤和多选 Skills

## 安装

**使用 Eget 快速安装**

Can quickly install by [inherelab/eget](https://github.com/inherelab/eget)

```bash
eget install inhere/skillc
```

**使用 Go install 安装**

```bash
go install github.com/inhere/skillc/cmd/skillc@latest
```

**从源码构建**

```bash
git clone https://github.com/inhere/skillc
cd skillc
make build          # 编译到当前目录
make install        # 安装到 $GOPATH/bin
```

## 快速开始

```bash
# 1. 初始化配置
skillc config init

# 2. 添加 Skill 来源（Git 仓库或本地路径）
skillc source add https://github.com/org/skills.git --id org-skills --name "Org Skills"
skillc source add /path/to/my-skills --id my-skills

# 3. 同步来源（拉取并建立索引）
skillc source sync --all

# 4. 搜索 Skill
skillc search typescript

# 5. 安装 Skill
skillc install my-skill

# 6. 查看已安装
skillc list
```

## 命令参考

### `config` — 配置管理

```bash
skillc config init          # 初始化配置文件
skillc config show          # 显示当前配置
skillc config set <key> <value>  # 修改配置项
```

### `source` — 来源管理

```bash
skillc source list                              # 列出所有来源
skillc source add <path-or-git-url> [--id <id>] [--name <name>] [--ref <ref>] [--sync]
skillc source add git <url> [ref] [--id <id>] [--name <name>] [--sync]
skillc source add local <path> [--id <id>] [--name <name>] [--sync]
skillc source info <id>                        # 查看来源详情（支持部分 ID 匹配）
skillc source sync <id>                        # 同步指定来源（支持部分 ID 匹配）
skillc source sync --all                       # 同步所有来源
skillc source status                           # 查看来源状态
skillc source collections [source]             # 查看来源下的 Collection
skillc source skills <source>                  # 查看来源下的 Skills
skillc source skills <source> --collection <name>
skillc source remove <id>                      # 删除来源
```

> 新生成的 source ID 不再强制添加 `local-` / `git-` 前缀；已有配置中的旧 ID 不会被自动改写。
> `source sync` 和 `source info` 支持 **部分 ID 匹配**，例如 `skillc source sync edge` 可匹配 `golang-edge-skills`。

### `registry` — 发现并安装 Registry Skills

```bash
skillc registry list
skillc registry add <path-or-url> --id official --name "Official Registry"
skillc registry add https://skillsmp.com --id skillsmp --name SkillsMP --provider skillsmp
skillc registry sync <id>
skillc registry sync --all
skillc registry search go                         # 默认搜索 Skill 结果
skillc registry search go --registry skillsmp     # 通过 SkillsMP provider 远程搜索
skillc registry info official/go-pro              # 查看 Registry Skill 详情
skillc registry install official/go-pro --agent codex --scope project --yes
skillc registry search gstack --kind source       # 查看 source catalog 结果
skillc registry add-source official/gstack [--id <id>] [--name <name>] [--sync]
```

Registry 用于从 generic JSON catalog 或内置 provider adapter 发现 Skill 级结果。`registry install` 会先把单个 Skill materialize 到本地 registry cache，再复用普通 install/lock 流程，并在 lock 中记录 `source_type=registry` provenance。`registry add-source` 是可选入口，只用于把 source 结果转成长期订阅的 source。

SkillsMP 是第一个内置 provider adapter。它不同于 generic JSON registry：搜索时会按关键词请求远程站点，并把返回的 Skill 结果缓存到本地，让 `registry info` 和 `registry install` 继续复用普通 Registry 安装链路。

Registry Skill entry 可以使用 Git/本地 `source_url`，也可以使用 archive `download_url`。Archive 下载支持 `zip`、`tar.gz`、`tgz`；`checksum` 用 SHA-256 校验 archive 原始字节后再解压：

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

### `profile` — Skill 组合

```bash
skillc profile list                                      # 列出已保存的 Profiles
skillc profile show <name>                               # 查看 Profile 详情
skillc profile create <name> --from-installed            # 从当前已安装 Skills 创建 Profile
skillc profile create <name> --from-collection <source>/<collection>
skillc profile create go-dev --interactive               # 交互式选择 Skills 创建 Profile
skillc profile diff <name>                               # 预览应用计划
skillc profile apply <name> --dry-run                    # 只输出计划，不安装
skillc profile apply <name> --yes                        # 跳过确认并应用 Profile
```

### `project` — 已登记项目

```bash
skillc project list                                      # 列出已登记项目
skillc project add . --id my-project --name "My Project"
skillc project remove my-project
skillc project import-lock                               # 从已有 lock 记录导入项目路径
```

已登记项目是 Web 跨项目更新和 `update --all-projects` 使用的显式 allowlist。

### `install` — 安装 Skills

```bash
skillc install <skill-id>                      # 安装指定 Skill
skillc install <id1>,<id2>                     # 批量安装（逗号分隔）
skillc install                                 # 从锁文件恢复所有 Skills
skillc install --interactive [keyword]         # 交互式过滤并多选 Skills

# 一次性：新增来源 → 同步 → 安装（支持 Git URL 或本地路径）
skillc install --source https://github.com/org/skills.git my-skill
skillc install --source /path/to/local-skills my-skill

# 选项
-s, --scope   安装范围（project / global）默认: project
-a, --agent   目标 Agent（默认: claude-code）
-y, --yes     跳过确认提示
-S, --source  Git URL 或本地路径，安装前自动注册并同步
-i, --interactive  打开交互式 Skill 选择器
    --install-mode <mode> 安装方式：symlink / junction / copy
    --copy    等同于 --install-mode copy
```

交互式选择基于 `gookit/cliui`：输入关键词过滤候选项，按空格多选，按回车确认后继续走原有安装计划、确认和执行流程。

> 默认安装方式按平台选择：Windows 使用 **junction**，其他系统使用 **symlink**。
> `symlink` 便于多个项目共享同一份 skill 源；Windows 上若没有创建符号链接的权限，会自动回退到 copy 模式并打印提示。
> 如果 Codex 或其他工具没有加载 Windows 目录 symlink，可使用 `--install-mode junction` 或配置 `install_mode: junction` 改用目录联接点。
> 可通过配置 `install_mode: copy` 永久切换到拷贝模式，或使用 `--copy` / `--install-mode <mode>` 临时覆盖。

### `update` — 更新 Skills

```bash
skillc update                           # 更新所有已安装的 Skills
skillc update --target <skill-id>       # 更新指定 Skill
skillc update --check                   # 只预览更新候选，不安装
skillc update --interactive             # 交互式过滤并多选可更新项
skillc update --all-projects --check    # 预览已登记项目的更新候选
skillc update --all-projects --projects my-project,api --target go-pro --yes
```

`update --check` 和 `status` 会报告精确 drift：版本相同但来源元数据变化时，Git source 比较 resolved ref，本地 source 比较目录级 checksum。

### 项目登记与跨项目更新

`skillc project` 用于登记允许被 Web 和 `update --all-projects` 管理的本机项目。跨项目更新只作用于这些 registered projects，不会直接扫描 lock 中的未知项目。推荐先运行 `skillc project add . --id <id>` 或 `skillc project import-lock`，再使用 `skillc update --all-projects --check` 查看计划，确认后用 `skillc update --all-projects --yes` 执行。

### `status` — Skill 状态

```bash
skillc status                           # 查看当前项目 Skill 状态
skillc status --profile go-dev          # 按 Profile 过滤
skillc status --agent claude-code       # 按 Agent 过滤
```

### `web` — 本地管理界面

```bash
skillc web
skillc web --port 8090
skillc web --host 127.0.0.1 --port 8090
```

Web 管理界面默认监听 `127.0.0.1`，当前支持查看 source/profile/status/install-map/version-drift、Registry 搜索/同步/安装/add-source，计划后确认执行当前项目 profile apply/update、source add/sync/remove、profile save/from-installed/from-collection、uninstall，以及已登记项目的跨项目 update plan/run。所有 Web 写操作都会先展示 plan，执行请求必须包含 `confirm:true`，写入本地 `skillc-web-history.jsonl` 历史记录，并且只操作当前项目或显式选择的 registered projects。

在 `skillc web` 的 Registry 页面可以搜索已同步的 registry Skills，预览安装计划，把 registry Skill 安装到当前项目，同步 registry catalog，或把 registry source 结果转换为已配置 source。

Version Drift 视图也会展示 checksum / Git ref 信号，因此版本号相同但内容或 commit 已变化的情况也能被看到。

### `uninstall` — 卸载 Skills

```bash
skillc uninstall <skill-id> [...]       # 卸载一个或多个 Skill
```

### `list` — 已安装列表

```bash
skillc list                             # 列出当前 Agent 已安装的 Skills
skillc list --scope global              # 列出全局 Skills
```

### `search` / `show` — 索引搜索

```bash
skillc search <keyword>                 # 关键词搜索
skillc search <keyword> --agent claude  # 过滤指定 Agent
skillc show <skill-id>                  # 查看 Skill 详情
```

Collection 通过 `source collections` 和 `source skills --collection` 浏览；如果要把某个 Collection 作为项目可复用技能组合，请先 `profile create --from-collection`，再 `profile apply`。

### `doctor` — 环境诊断

```bash
skillc doctor                           # 检查 git、配置文件、索引等是否就绪
```

## 配置文件

默认配置文件查找顺序：

1. `./skillc.yaml`（当前目录）
2. `~/.config/skillc/config.yaml`

主要配置项：

```yaml
lock_file: skillc.lock.yaml       # 锁文件路径
index_file: skillc-index.json     # 索引文件路径
repo_cache_dir: ~/.cache/skillc   # Git 仓库缓存目录
proxy_url: ""                     # HTTP 代理（可选）
sources: []                       # 管理的来源列表
projects: []                      # 跨项目更新允许管理的本机项目列表
agent_tools:                      # Agent 工具配置 agent_name: config
  claude-code:
    dirname: .claude
    aliases:
    - claude
  codex:
    dirname: .codex
  opencode:
    # dirname: .opencode # 默认是 .{agent_name}
    user_dir: ~/.config/opencode
  universal: # 通用 agent 配置, 大部分 agent tool 都支持
    dirname: .agents
    aliases:
    - agents
    user_dir: ~/.agents
    project_dir: .agents
```

## 锁文件

`skillc.lock.yaml` 记录每个已安装 Skill 的元数据，用于 `skillc install`（无参数）时恢复所有 Skills：

```yaml
records:
  - skill_id: my-skill
    source_id: git-org-skills
    agent: claude-code
    scope: project
    installed_path: .claude/skills/my-skill
    installed_at: "2026-01-01T00:00:00Z"
```

## 开发

```bash
go test ./...          # 运行所有测试
make build             # 本地构建
```

## License

MIT
