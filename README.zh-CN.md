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
- 🔒 **锁文件追踪** — 记录每个 Skill 的来源、版本和安装位置，支持 `restore`
- 🤖 **多 Agent 适配** — 自动适配不同 Agent 的安装目录规范
- 🔄 **批量更新** — 一条命令更新所有已安装的 skills

## 安装

### 从源码构建

```bash
git clone https://github.com/inhere/skillc
cd skillc
make build          # 编译到当前目录
make install        # 安装到 $GOPATH/bin
```

### 交叉编译

```bash
make build-all      # 全平台
make build-linux    # Linux amd64
make build-darwin   # macOS Intel
make build-darwin-arm64  # macOS Apple Silicon
make build-windows  # Windows amd64
```

## 快速开始

```bash
# 1. 初始化配置
skillc config init

# 2. 添加 Skill 来源（Git 仓库或本地路径）
skillc source add git https://github.com/org/skills.git
skillc source add local /path/to/my-skills

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
skillc source add git <url> [ref]              # 添加 Git 来源
skillc source add local <path>                 # 添加本地来源
skillc source sync <id>                        # 同步指定来源（支持部分 ID 匹配）
skillc source sync --all                       # 同步所有来源
skillc source status                           # 查看来源状态
skillc source remove <id>                      # 删除来源
```

> `source sync` 支持 **部分 ID 匹配**，例如 `skillc source sync edge` 可匹配 `local-golang-edge-skills`。

### `install` — 安装 Skills

```bash
skillc install <skill-id>                      # 安装指定 Skill
skillc install <id1>,<id2>                     # 批量安装（逗号分隔）
skillc install --collection <collection>       # 安装整个 Collection
skillc install                                 # 从锁文件恢复所有 Skills

# 一次性：新增来源 → 同步 → 安装（支持 Git URL 或本地路径）
skillc install --source https://github.com/org/skills.git my-skill
skillc install --source /path/to/local-skills my-skill

# 选项
-s, --scope   安装范围（project / global）默认: project
-a, --agent   目标 Agent（默认: claude-code）
-y, --yes     跳过确认提示
-c, --collection  将目标视为 Collection 选择器
-S, --source  Git URL 或本地路径，安装前自动注册并同步
```

### `update` — 更新 Skills

```bash
skillc update                           # 更新所有已安装的 Skills
skillc update --target <skill-id>       # 更新指定 Skill
```

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

### `collection` — Collection 浏览

```bash
skillc collection list                  # 列出所有 Collection
skillc collection skills <name>         # 列出 Collection 内的 Skills
```

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

