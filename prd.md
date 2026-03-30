# Skillc PRD

> Create At: 2026-03-27, Update At: 2026-03-30, Version: 0.1.2, Status: Draft

## 1. 文档信息

- 产品名称：Skillc
- 产品形态：命令行工具（CLI）
- 技术栈：Go
- 文档状态：Draft v1
- 文档目标：定义 Skillc 的目标用户、产品范围、功能要求、配置模型与验收标准，用于指导后续设计与开发

## 2. 产品概述

Skillc 是一个面向多 Agent 生态的本地 Skill 管理工具，用于统一发现、安装、更新、卸载和同步本机 Skills。它屏蔽不同 Agent 在目录结构、安装位置和元数据上的差异，为用户提供一套一致的 CLI 体验。

Skillc 首期目标支持 `claude-code`、`opencode`、`codex` 等主流 Agent 工具，并支持以下能力：

- 管理多种 Skill 来源，包括本地路径、Git 仓库、第三方搜索平台注册表
- 搜索、下载、安装、删除和更新 Skills
- 安装到全局目录或项目目录
- 通过本地缓存避免重复下载和重复解析
- 维护安装锁文件，记录来源、版本、安装位置和 Agent 绑定关系

## 3. 背景与问题

当前不同 Agent 对 Skills 的目录约定、安装方式和来源组织差异较大，导致用户面临以下问题：

- 同一个 Skill 在不同 Agent 中需要重复安装和重复维护
- 技能来源分散在本地文件夹、GitHub 仓库、第三方平台，发现成本高
- 安装和升级过程依赖手工复制，容易出错且不可追踪
- 缺少统一缓存，重复下载浪费时间与网络资源
- 缺少统一的锁定与版本记录，无法清晰知道“装了什么、从哪里来、当前是不是最新”

Skillc 的价值在于把“Skill 来源管理”和“Agent 安装适配”拆开，用统一的数据模型管理所有来源，再通过 Agent 适配器完成最终落地安装。

## 4. 产品目标

### 4.1 核心目标

- 为多 Agent 提供统一的 Skill 生命周期管理能力
- 支持用户从多来源发现并安装 Skills
- 通过缓存机制降低重复下载和重复解析成本
- 让用户可以清晰追踪 Skill 的来源、版本、安装位置和更新时间
- 将 Skill 管理从“手工复制文件”升级为“可搜索、可安装、可更新、可回滚感知”的标准化流程

### 4.2 非目标

本期不包含以下范围：

- 不做图形界面，仅提供 CLI
- 不做在线账号体系、云端同步或远程托管
- 不做 Skill 发布平台，本期仅消费外部来源
- 不解决 Agent 运行时兼容性问题，只负责安装与元数据管理
- 不做复杂依赖解析，本期不支持 Skill 间依赖树管理

## 5. 目标用户

### 5.1 主要用户

- 同时使用多个 Agent 工具的开发者
- 维护本地 Skill 仓库或团队共享 Skill 仓库的工程师
- 希望从第三方平台快速发现并安装 Skills 的高级用户

### 5.2 典型用户画像

#### 用户 A：个人开发者

希望在 `claude-code` 和 `codex` 中复用一套 Skills，不想手动复制和更新。

#### 用户 B：团队维护者

维护团队 Skill 仓库，希望团队成员通过统一命令完成同步、安装和升级。

#### 用户 C：重度探索用户

希望从多个公开站点搜索 Skills，筛选后安装到本地试用，并能方便清理和升级。

## 6. 核心概念

### 6.1 Skill

Skill 是一份可安装到某个 Agent 工具目录中的能力包，包含至少以下信息：

- `id`：Skill 唯一标识
- `name`：展示名称
- `description`：简述
- `version`：从 Skill 元数据文件（`SKILL.md` YAML front matter）中读取；若无则回退为 Git commit SHA（短格式）或内容指纹
- `supported_agents`：支持的 Agent 列表
- `source`：来源信息
- `install_entry`：安装入口目录或文件（指向 Skill 内容的根路径，相对于 Skill 根目录；可以是单个文件或目录，由 Agent 适配器决定如何处理）

> **Skill 元数据解析规则**：Skillc 不强制要求独立的 `skill.yaml`，而是从 Skill 目录下 `SKILL.md` 的 YAML front matter 中读取元数据（id、name、description、version、supported_agents 等）。一个 Git 仓库或本地目录中可以包含多个 Skill，每个 Skill 对应一个子目录，子目录内含 `SKILL.md`。对于没有 `SKILL.md` 的目录，Skillc 跳过该目录或以目录名作为 id 进行有限索引并标注"元数据不完整"。

### 6.2 Source

Source 是 Skill 的来源，分为三类：

- **本地路径源（local）**：用户本机目录，Skillc 扫描并建立索引
- **Git 仓库源（git）**：通过 Git 拉取到本地缓存，支持指定 branch/tag/ref
- **搜索注册表源（registry）**：第三方平台提供的搜索接口，返回 Skill 元数据和下载地址

> **注意**：本地路径源和 Git 仓库源通过 `skillc source` 子命令管理；Registry 通过 `skillc registry` 子命令管理。两者在数据模型层统一为 Source 类型，`type` 字段枚举值为 `local` / `git` / `registry`。Registry 来源安装的 Skill，Lock 记录中 `source_id` 填写对应 Registry 的 id，`source_type` 填写 `registry`。

### 6.3 Repository

Repository 是可被 Skillc 扫描、索引和缓存的一组 Skill 集合。Repository 可来源于本地目录，也可来源于 Git 仓库。

### 6.4 Registry

Registry 是可搜索的 Skill 注册平台。它不一定直接承载 Skill 文件，但必须能返回足够的下载信息或跳转信息。

Registry 接入优先级：
1. **JSON 索引**：若 Registry 提供静态 JSON 索引文件，优先使用（性能好、可缓存）
2. **HTTP API**：若无 JSON 索引，则使用 HTTP API 查询

Registry 返回的搜索结果统一映射为内部 Skill 元数据模型后，安装时需先将 Skill 内容下载到本地缓存，再执行安装。Lock 记录中 `source_id` 填写对应 Registry 的 id，`source_type` 填写 `registry`。

### 6.5 Cache

Cache 是 Skillc 的本地缓存区，负责保存：

- 仓库克隆结果
- 仓库索引结果
- Skill 包快照
- Registry 查询结果缓存

Skill 安装动作必须优先从 Cache 中读取，只有在缓存缺失或过期时才触发拉取或同步。

### 6.6 Install Target

Install Target 表示 Skill 的安装目标，分为：

- 全局安装：安装到用户级 Agent 目录
- 项目安装：安装到当前项目内 Agent 目录

### 6.7 Lock File

Lock File 以 JSON 格式记录本机已经安装的 Skill 清单，包括：

- Skill 标识
- 来源仓库或注册表（source_id + source_type，保证 source 被删除后仍可追溯）
- 解析到的具体版本或 commit
- 安装到的 Agent
- 安装范围（global/project）
- 安装路径（相对于 Agent 目录的相对路径，跨平台安全）
- 文件完整性指纹（checksum，SHA256）
- 安装时间
- 最近同步时间
- 是否锁定版本（pinned，为 true 时 update 命令跳过此条记录）

## 7. 用户场景

### 7.1 添加来源

用户将一个本地目录或 Git 仓库加入 Skillc，Skillc 扫描并建立索引。

### 7.2 搜索 Skill

用户输入关键字后，Skillc 会联合搜索本地缓存仓库与第三方 Registry，统一输出候选结果。

### 7.3 安装 Skill

用户选择一个 Skill，指定 Agent 和安装范围，Skillc 从缓存中复制到目标目录，并更新锁文件。

### 7.4 升级 Skill

用户执行更新命令，Skillc 会先同步来源，再比较已安装版本与最新版本，提示或执行升级。

### 7.5 删除 Skill

用户按 Agent 和安装范围删除某个已安装 Skill，同时清理锁文件记录。

### 7.6 查看状态

用户查看当前配置、来源列表、缓存状态、已安装 Skill 列表和可升级项。

### 7.7 批量恢复

用户在新机器或新环境中，通过现有 Lock File 执行 `skillc install`（无参数）一键恢复全部已安装 Skill，无需逐条重新安装。

## 8. 产品范围

### 8.1 MVP 范围

首期必须交付：

- 本地路径源管理
- Git 仓库源管理
- 第三方 Registry 搜索（含 `skillc registry` 子命令管理）
- Skill 搜索
- Skill 安装（含从 Lock File 批量恢复）
- Skill 删除
- Skill 更新
- 全局安装与项目安装
- 多 Agent 安装适配
- 本地缓存与锁文件
- 环境诊断（`skillc doctor`）

### 8.2 后续可扩展范围

- Skill 评分、收藏、来源可信度展示
- 批量安装与批量更新
- 交互式终端 UI
- Skill 校验签名或来源安全评分
- 团队级共享配置
- 依赖管理与冲突检测
- 开发者软链接模式（`skillc link <path>`：将本地开发中的 Skill 软链接到 Agent 目录，用于实时调试）

## 9. 功能需求

### 9.1 配置管理

Skillc 需要支持初始化、读取、校验和展示配置文件。

功能要求：

- 支持默认配置文件路径（按平台规范，见 §11.3）
- 支持通过命令行 `--config <path>` 指定自定义配置文件
- 支持配置代理地址，用于 HTTP 请求和 Git 操作
- 支持配置 Agent 工具目录规则
- 支持配置缓存目录和锁文件路径
- 支持配置默认 Registry 和默认仓库源
- 支持通过 CLI 读取和设置配置项（`skillc config get <key>` / `skillc config set <key> <value>`），无需手动编辑 YAML

验收标准：

- 未显式指定配置时，Skillc 能按系统标准路径找到配置文件
- 配置缺失时可自动生成最小默认配置
- 配置项格式错误时能给出明确错误提示

### 9.2 Agent 适配

Skillc 需要通过 Agent 适配层屏蔽不同 Agent 的目录结构差异。

首期支持：

- `claude-code`
- `opencode`
- `codex`（目录规范待最终确认，见 §18 待确认问题）

适配要求：

- 每个 Agent 都有独立的目录规则，包含全局目录和项目目录的路径解析
- **全局目录默认规则**：所有平台统一使用 `~/.{agent-name}/`（如 `~/.claude/`、`~/.opencode/`、`~/.codex/`），方便用户记忆查找；用户可在配置中覆盖
- **项目目录默认规则**：使用当前工作目录下的 `.{agent-name}/`
- **Agent 目录名推断规则**：若配置中未显式指定 `dirname`，默认以 `.{工具名称}` 推断（如 codex → `.codex`）
- 安装前验证目标目录是否存在，不存在时可自动创建
- 安装后产物结构满足对应 Agent 的加载约定

### 9.3 来源管理

Skillc 需要统一管理本地源与 Git 源。

功能要求：

- 添加本地路径源（支持指定 `--id` 和 `--name`）
- 添加 Git 仓库源（支持指定 `--id`、`--name`、`--ref <branch|tag|commit>`）
- 列出当前所有来源（输出包含 id，便于后续操作引用）
- 删除来源
- 手动同步来源
- 查看来源最新同步状态

行为约束：

- 添加本地路径源后，Skillc 需要建立缓存索引
- 添加 Git 仓库源后，Skillc 需要克隆到缓存目录；系统未安装 Git 时应给出明确错误并指引安装
- 对重复来源进行去重
- 对不可访问的来源保留失败状态，但不影响其他来源

### 9.4 Registry 管理

Skillc 需要支持从第三方站点搜索 Skill，并提供 CLI 管理 Registry 列表。

功能要求：

- 支持通过 `skillc registry add/remove/list` 管理 Registry（与 `skillc source` 体验一致）
- 支持配置多个 Registry 地址
- 支持按关键字搜索
- 支持聚合多个 Registry 结果
- 支持展示来源站点、Skill 名称、描述、支持 Agent、下载入口

行为约束：

- Registry 返回结构不一致时，通过适配层统一为内部模型（Registry 接入协议规范见 §18 待确认问题）
- 某个 Registry 超时或失败时，不中断整体搜索
- 支持对 Registry 搜索结果做短时缓存

### 9.5 Skill 索引与搜索

Skillc 需要统一搜索本地缓存仓库和 Registry 结果。

功能要求：

- 按名称、描述、标签、来源搜索
- 可按 Agent 过滤
- 可按来源类型过滤
- 可按是否已安装过滤
- 支持精确匹配与模糊匹配

输出要求：

- 返回 Skill 标识
- 返回来源（source_id + source_type）
- 返回版本信息
- 返回支持的 Agent
- 返回是否已安装及当前状态（installed/outdated/missing）
- 返回安装建议或冲突提示
- 支持 `--json` 输出格式，便于脚本和管道使用

### 9.6 Skill 安装

Skillc 需要支持从缓存安装 Skill 到指定 Agent 目录。

功能要求：

- 指定 Skill、Agent、安装范围执行安装
- 支持从仓库源安装
- 支持从 Registry 解析后安装（先下载内容到缓存，再执行安装）
- 安装完成后写入锁文件（含 checksum、source_type 字段）
- 如果目标已存在，**默认提示用户选择操作**（覆盖/跳过/取消），使用 `--yes` 时默认覆盖
- 支持 `--yes/-y` 跳过交互确认，适合 CI/CD 场景
- 支持 `--dry-run` 预览将要执行的操作，不实际写入
- 支持版本约束：`skillc install <skill-id>[@<version>]`，不指定版本时默认安装 `latest`
- `skillc install`（不带 skill-id 参数）时，从当前 Lock File 读取全部记录批量恢复已安装 Skill

安装规则：

- 所有安装动作必须从本地缓存读取
- 对于 Git 源，先同步到缓存再安装
- 对于本地路径源，先更新缓存镜像或索引再安装
- 对于 Registry 源，先解析下载地址并缓存，再执行安装
- 默认采用复制安装，不依赖软链接
- 安装目标路径下已存在非 Skillc 管理的同名文件/目录时，提示用户选择处理方式（跳过或 `--force` 覆盖）

验收标准：

- 安装成功后，目标 Agent 能在其约定目录中看到 Skill
- 锁文件记录完整（含 checksum、source_type）
- 同一 Skill 可安装到多个 Agent，但每个安装记录独立追踪
- `skillc install`（无参数）能从 Lock File 批量恢复全部 Skill

### 9.7 Skill 删除

Skillc 需要支持卸载已安装 Skill。

功能要求：

- 按 Skill、Agent、安装范围删除
- 删除目标目录中的 Skill 产物
- 删除锁文件中的对应记录
- 若目标不存在，应给出幂等提示而不是报错中断
- 支持 `--yes/-y` 跳过交互确认
- 支持 `--dry-run` 预览将要删除的内容

### 9.8 Skill 更新

Skillc 需要支持来源同步和已安装 Skill 升级。

更新分为两层：

- **Source Update**：通过 `skillc source sync [source-id]` 或 `skillc registry refresh [registry-id]` 手动同步来源
- **Skill Update**：通过 `skillc update [<skill-id>]` 对比锁文件中已安装版本与来源最新版本，执行升级。**执行 Skill Update 时会自动先触发相关来源的 Source Update**，用户无需手动先执行 sync

功能要求：

- 检查某个 Skill 是否可升级
- 列出所有可升级 Skill（`skillc outdated`）
- 升级单个 Skill
- 升级指定 Agent 下的所有 Skills（`--agent <agent> --all`）
- 支持 `--dry-run` 预览将要升级的内容
- `pinned: true` 的 Skill 在 update 时自动跳过

版本判断规则：

- Git 源优先基于 commit/tag 判断
- 本地路径源基于内容指纹（SHA256）或修改时间加指纹判断
- Registry 源基于其返回的版本字段或下载对象摘要判断

### 9.9 已安装状态管理

Skillc 需要提供已安装清单和状态查询能力。

功能要求：

- 列出所有已安装 Skills
- 按 Agent 过滤
- 按安装范围过滤
- 显示来源、版本、安装路径、checksum、最近更新时间
- 标记异常状态：`missing`（目标产物缺失）、`outdated`（存在更新）、`orphan`（来源已失效）
- 支持 `--json` 输出格式，便于脚本使用

### 9.10 缓存管理

Skillc 需要提供缓存清理与重建能力。

功能要求：

- 查看缓存目录占用
- 清理某个来源缓存
- 清理失效缓存
- 全量重建缓存索引

行为约束：

- 清理缓存不应默认删除已安装 Skill
- 缓存损坏时可通过重建恢复

## 10. CLI 信息架构

建议首期命令结构如下：

```bash
# 工具信息
skillc version
skillc help [command]
skillc completion bash|zsh|fish|powershell   # 生成 shell 补全脚本

# 初始化
skillc init [--global|--project]   # 全局或项目级初始化；不带参数时提示选择

# 配置管理
skillc config show
skillc config get <key>
skillc config set <key> <value>

# 来源管理（本地路径源 + Git 源）
skillc source add local <path> [--id <id>] [--name <name>]
skillc source add git <url> [--id <id>] [--name <name>] [--ref <branch|tag|commit>]
skillc source list
skillc source remove <source-id>
skillc source sync [source-id]
skillc source status [source-id]

# Registry 管理（第三方搜索平台）
skillc registry add <url> [--id <id>] [--name <name>]
skillc registry list
skillc registry remove <registry-id>
skillc registry refresh [registry-id]

# 搜索与详情
skillc search <keyword> [--agent <agent>] [--source <source-id>] [--installed] [--json]
skillc show <skill-id>

# 安装管理
skillc install [<skill-id>[@<version>]] [--agent <agent>] [--scope <global|project>] [--yes] [--dry-run]
# 不带 skill-id 时：从 Lock File 批量恢复全部已安装 Skill
# 不带 --agent 时：安装到所有已配置 Agent 的全局目录
# 不带 --scope 时：默认 global
# 不带 @version 时：默认 latest
skillc uninstall <skill-id> [--agent <agent>] [--scope <global|project>] [--yes] [--dry-run]
skillc list [--agent <agent>] [--scope <global|project>] [--status <installed|outdated|missing|orphan>] [--json]
skillc outdated [--agent <agent>] [--json]
skillc update [<skill-id>] [--agent <agent>] [--all] [--dry-run]

# 缓存管理
skillc cache info
skillc cache clean [--source <source-id>]
skillc cache rebuild

# 工具与诊断
skillc doctor
skillc validate <path>
```

说明：

- `source` 管理本地路径源和 Git 源；`registry` 管理第三方搜索平台，两套命令体验对称
- `search` 聚合本地仓库和 Registry 结果
- `show` 展示 Skill 详情（元数据、来源、已安装状态、版本信息）
- `outdated` 列出可更新项
- `install`（不带 skill-id）从 Lock File 批量恢复
- `--yes/-y` 跳过交互确认，适合 CI/CD 场景
- `--dry-run` 预览操作，不实际写入
- `--json` 输出结构化 JSON，便于脚本和管道使用
- `--agent` / `--scope` 缺省行为见 §18 待确认问题

## 11. 配置模型

### 11.1 配置原则

- 配置文件使用 YAML 格式
- 路径支持 `~`、相对路径和环境变量展开
- 默认路径遵循操作系统规范（见 §11.3）
- 用户可以覆盖默认 Agent 安装目录

### 11.2 建议配置示例

```yaml
proxy_url: http://localhost:7890

agent_tools:
  claude-code:
    dirname: .claude
    user_dir: ~/.claude              # 所有平台统一使用 ~/，方便记忆查找
    project_dir: ./.claude
  opencode:
    dirname: .opencode
    user_dir: ~/.opencode            # 所有平台统一使用 ~/，方便记忆查找
    project_dir: ./.opencode
  codex:
    dirname: .codex                  # 按工具名称推断：.{name}
    user_dir: ~/.codex               # 所有平台统一使用 ~/，方便记忆查找
    project_dir: ./.codex

lock_file: ~/.config/skillc/skillc-install.lock   # JSON 格式
repo_cache_dir: ~/.cache/skillc/repos
skill_cache_dir: ~/.cache/skillc/skills
registry_cache_dir: ~/.cache/skillc/registry

sources:
  local:
    - id: team-skills
      name: Team Skills
      path: ~/workspace/team-skills
  git:
    - id: awesome-claude-skills
      name: Awesome Claude Skills
      url: https://github.com/ComposioHQ/awesome-claude-skills
      ref: main
    - id: antigravity-awesome-skills
      name: Antigravity Awesome Skills
      url: https://github.com/sickn33/antigravity-awesome-skills
      ref: main

registries:
  - id: skills-sh
    name: Skills.sh
    url: https://skills.sh/
  - id: skillsmp
    name: SkillsMP
    url: https://skillsmp.com/
  - id: skillsllm
    name: SkillsLLM
    url: https://skillsllm.com/
```

### 11.3 跨平台路径规范

Skillc 兼容 Linux、macOS、Windows，各平台默认路径遵循以下规则：

| 路径类型 | 默认值 | 说明 |
|----------|--------|------|
| Agent 全局目录 | `~/.{agent-name}/` | 所有平台统一，方便用户记忆查找 |
| 配置目录 | `~/.config/skillc/` | 使用 `os.UserConfigDir()` 解析 |
| 缓存目录 | `~/.cache/skillc/` | 使用 `os.UserCacheDir()` 解析 |

实现要求：

- Agent 全局目录所有平台统一使用 `~/.{agent-name}/`，不区分平台
- 配置目录和缓存目录使用 Go 标准库（`os.UserConfigDir()`、`os.UserCacheDir()`）解析
- 避免硬编码路径，路径支持 `~` 和环境变量展开
- 在 Windows 上正确处理盘符、路径分隔符
- 注意 Windows 默认 260 字符路径限制，避免深层嵌套导致路径超限

## 12. 数据模型要求

### 12.1 Source 模型

至少包含：

- `id`
- `type`：local / git / registry
- `name`
- `path`：本地路径（local 类型）
- `url`：远程地址（git / registry 类型）
- `ref`：Git branch/tag/commit（git 类型，可选，缺省跟踪默认分支）
- `last_sync_at`
- `status`：ok / error / syncing
- `error_message`

### 12.2 Skill 元数据模型

至少包含：

- `id`
- `name`
- `description`
- `version`（语义按 source_type 区分，见下方说明）
- `tags`
- `supported_agents`
- `source_id`
- `source_type`：local / git / registry
- `install_entry`（相对于 Skill 根目录的路径，目录或单文件）
- `homepage`（可选）
- `author`（可选）
- `license`（可选）

**version 字段语义：**

version 统一从 Skill 的 `SKILL.md` YAML front matter 中读取。由于一个 Git 仓库或本地目录可包含多个 Skill，version 是每个 Skill 自身的元数据，而非仓库级别的版本。

回退规则（当 `SKILL.md` 中无 version 字段时）：
- `git` 来源：使用当前 commit SHA（短格式 8 位）
- `local` 来源：使用内容 SHA256 指纹前 8 位
- `registry` 来源：使用 Registry 返回的 version 字段；若无则使用下载对象摘要

### 12.3 Lock 记录模型

至少包含：

- `skill_id`
- `agent`
- `scope`：global / project
- `version`
- `source_id`
- `source_type`：local / git / registry（保证 source 被删除后仍可追溯）
- `resolved_ref`：解析到的具体 commit / 指纹
- `installed_path`：相对于 Agent 目录的相对路径（跨平台安全，避免绝对路径）
- `checksum`：安装产物的 SHA256 指纹，用于完整性校验
- `installed_at`
- `updated_at`
- `pinned`：bool，为 true 时 `skillc update` 跳过此条记录

## 13. 非功能需求

### 13.1 性能

- 本地搜索应优先命中缓存索引
- 在 1000 个 Skill 规模下，本地搜索 P95 响应时间应 ≤ 500ms
- 重复安装已缓存 Skill 时，不应再次访问网络

### 13.2 可靠性

- 单个来源失败不影响整体可用性
- 锁文件写入要具备基本原子性（先写临时文件再原子替换），避免部分写入损坏
- 安装失败时要尽量回滚目标目录到一致状态
- 通过文件锁（advisory lock）防止多进程并发写入冲突
- 锁文件损坏时，`skillc doctor` 可检测并提供修复建议

### 13.3 可维护性

- Agent 适配器与来源适配器解耦
- Registry 适配器支持后续新增站点
- 内部统一使用标准化 Skill 元数据模型

### 13.4 可观测性

- CLI 输出应明确展示当前操作阶段
- 出错时展示来源、目标和失败原因
- 支持 `--debug` 标志或环境变量 `SKILLC_DEBUG=1` 开启 debug 级日志
- 日志输出到 stderr，格式为结构化文本（key=value）；日志文件路径可在配置中指定

## 14. 错误与边界场景

Skillc 需要明确处理以下情况：

- 来源地址不可访问
- Git 克隆失败或认证失败
- **Git 未安装**：输出明确错误并指引安装方式
- Registry 返回格式不兼容
- Skill 缺少必要元数据
- 指定 Agent 未配置
- 安装目标目录无权限写入
- 同名 Skill 来自多个来源
- 已安装 Skill 被用户手动删除或改坏（`skillc doctor` 可检测）
- **并发操作冲突**：多个终端同时运行 `skillc install/update` 时，通过文件锁保护，后进入者等待或报错提示
- **锁文件损坏**：解析失败时提示用户，提供 `skillc doctor` 修复选项，不自动覆盖原文件
- **磁盘空间不足**：在 clone/下载前预检可用空间，不足时给出明确错误
- **网络中断（部分完成）**：下载/clone 中途断开时，清理残留临时文件，下次重试时重新开始
- **安装目标路径已存在非 Skillc 管理的同名文件/目录**：提示用户手动处理或使用 `--force` 覆盖
- **Agent 自身升级后目录结构变更**：`skillc doctor` 可检测 orphan 状态并提示重新安装
- **Skillc 自身升级后格式迁移**：启动时检测配置/锁文件版本，自动执行向前兼容迁移；迁移失败时保留原文件并报错
- **符号链接循环**：扫描本地路径源时检测循环引用并跳过，输出警告

建议策略：

- 对同名冲突输出来源候选并要求用户显式选择
- 对缺少元数据的 Skill 标记为不可安装，但允许在搜索结果中展示警告
- 对目标目录异常给出修复建议

## 15. 安全与信任要求

本期不做复杂安全系统，但至少需要：

- 展示 Skill 来源和下载地址
- 安装前明确目标路径
- **默认不执行 Skill 包内的任何安装脚本（如 post-install hook）**；Skill 内容本身（prompt 指令等）由 Agent 在运行时处理，不在 Skillc 职责范围内
- 对来自 Registry 的跳转和下载地址进行基本合法性校验（协议限定为 HTTPS）
- 安装完成后记录 checksum，支持后续通过 `skillc doctor` 进行完整性验证

## 16. 里程碑建议

### 里程碑 1：本地与 Git 源打通

目标：用户能添加来源并在本地搜索到 Skill。

- 完成 `skillc init`（全局与项目级）
- 完成配置加载与 `skillc config show/get/set`
- 完成来源管理（`skillc source add/list/remove/sync/status`）
- 完成缓存目录结构
- 完成 Skill 索引和本地搜索（`skillc search`、`skillc show`）
- 完成环境诊断（`skillc doctor`）

**M1 验收标准：**

- 能添加至少一个本地源和一个 Git 源
- 能从本地来源搜索到 Skill 并查看详情
- `skillc doctor` 能检测 Git 可用性、目录权限等环境问题
- `skillc config get/set` 能读写配置项

### 里程碑 2：安装链路打通

目标：用户能安装、卸载 Skill 并查看已安装清单。

- 完成 Agent 适配（claude-code、opencode、codex）
- 完成安装（`skillc install`）、卸载（`skillc uninstall`）、列表（`skillc list`）
- 完成锁文件读写（含 checksum、source_type、pinned 字段）
- 完成从 Lock File 批量恢复（`skillc install` 无参数）
- 完成覆盖与冲突处理

**M2 验收标准：**

- 能把一个 Skill 安装到指定 Agent 的全局目录或项目目录
- 能卸载已安装 Skill
- 能查看已安装清单（含 missing/outdated/orphan 状态标记）
- 锁文件能准确记录安装结果（含 checksum、source_type）
- 相同 Skill 重复安装时能命中缓存而非重复下载
- `skillc install`（无参数）能从 Lock File 恢复全部 Skill

### 里程碑 3：更新与 Registry

目标：用户能升级 Skill 并从第三方平台搜索安装。

- 完成 Registry 管理（`skillc registry add/remove/list/refresh`）
- 完成 Registry 搜索与安装链路
- 完成可升级检查（`skillc outdated`）与更新命令（`skillc update`）
- 完成缓存清理和重建（`skillc cache info/clean/rebuild`）
- 完成 Skill 校验（`skillc validate`）
- 完成 Shell 补全（`skillc completion`）

**M3 验收标准：**

- 能从至少一个 Registry 搜索并安装 Skill
- 能检查并升级某个已安装 Skill
- 单个来源失败不会导致整个命令不可用

## 17. 验收标准

满足以下条件可视为 MVP 可用：

- 用户能添加至少一个本地源和一个 Git 源
- 用户能从来源中搜索并看到标准化 Skill 信息
- 用户能把一个 Skill 安装到指定 Agent 的全局目录或项目目录
- 用户能卸载已安装 Skill
- 用户能查看已安装清单（含 missing/outdated/orphan 状态标记）
- 用户能检查并升级某个已安装 Skill
- 用户能执行 `skillc install`（无参数）从 Lock File 批量恢复全部已安装 Skill
- 相同 Skill 重复安装时能命中缓存而非重复下载
- 锁文件能准确记录安装结果（含 checksum、source_type）
- 单个来源失败不会导致整个命令不可用
- `skillc doctor` 能检测并报告环境健康问题

## 18. 设计决策记录

以下问题已在设计阶段确认，记录决策结论供实现参考：

1. **Skill 元数据来源**：不强制要求 `skill.yaml`，从 `SKILL.md` 的 YAML front matter 读取元数据。无 `SKILL.md` 的目录跳过或以目录名作为 id 进行有限索引并标注"元数据不完整"。一个仓库/目录可包含多个 Skill，每个 Skill 一个子目录。

2. **version 字段语义**：统一从 `SKILL.md` front matter 读取。回退规则：git 来源用 commit SHA（8位），local 来源用内容指纹（8位），registry 来源用返回的 version 字段或下载摘要。

3. **Registry 接入协议**：优先使用 JSON 索引（若 Registry 提供），无 JSON 索引时使用 HTTP API。

4. **安装冲突默认策略**：提示用户选择（覆盖/跳过/取消）；使用 `--yes` 时默认覆盖。

5. **`--agent` / `--scope` 缺省行为**：不带 `--agent` 时安装到所有已配置 Agent；不带 `--scope` 时默认 `global`。

6. **版本约束安装语法**：支持 `skillc install <skill-id>[@<version>]`，不指定版本时默认 `latest`。

7. **Agent 目录名推断规则**：默认以 `.{工具名称}` 推断（如 codex → `.codex`）；全局目录所有平台统一使用 `~/.{name}/`，方便记忆查找。

8. **开发者软链接模式**：后续版本支持 `skillc link <path>`，首期不实现。

## 19. 附录：原始需求归纳

原始需求可以归纳为四条主线：

- 来源管理：本地路径、Git 仓库、第三方平台
- 生命周期管理：搜索、下载、安装、删除、更新
- 目标适配：多 Agent、全局目录、项目目录
- 本地状态管理：缓存、锁文件、版本追踪

以上四条主线已经在本 PRD 中展开为可落地的产品需求。
