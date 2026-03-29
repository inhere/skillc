# Skillc PRD

> Create At: 2026-03-27, Update At: 2026-03-28, Version: 0.1.0-beta1, Status: Pending

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
- `version`：版本号、Git commit、tag 或内容指纹
- `supported_agents`：支持的 Agent 列表
- `source`：来源信息
- `entry`：安装入口目录或文件

### 6.2 Source

Source 是 Skill 的来源，分为三类：

- 本地路径源：用户本机目录
- Git 仓库源：通过 Git 拉取到本地缓存
- 搜索注册表源：第三方平台提供的搜索接口，返回 Skill 元数据和下载地址

### 6.3 Repository

Repository 是可被 Skillc 扫描、索引和缓存的一组 Skill 集合。Repository 可来源于本地目录，也可来源于 Git 仓库。

### 6.4 Registry

Registry 是可搜索的 Skill 注册平台。它不一定直接承载 Skill 文件，但必须能返回足够的下载信息或跳转信息。

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

Lock File 用于记录本机已经安装的 Skill 清单，包括：

- Skill 标识
- 来源仓库或注册表
- 解析到的具体版本或 commit
- 安装到的 Agent
- 安装范围（global/project）
- 安装路径
- 安装时间
- 最近同步时间

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

## 8. 产品范围

### 8.1 MVP 范围

首期必须交付：

- 本地路径源管理
- Git 仓库源管理
- 第三方 Registry 搜索
- Skill 搜索
- Skill 安装
- Skill 删除
- Skill 更新
- 全局安装与项目安装
- 多 Agent 安装适配
- 本地缓存与锁文件

### 8.2 后续可扩展范围

- Skill 评分、收藏、来源可信度展示
- 批量安装与批量更新
- 交互式终端 UI
- Skill 校验签名或来源安全评分
- 团队级共享配置
- 依赖管理与冲突检测

## 9. 功能需求

### 9.1 配置管理

Skillc 需要支持初始化、读取、校验和展示配置文件。

功能要求：

- 支持默认配置文件路径
- 支持通过命令行指定自定义配置文件
- 支持配置代理地址，用于 HTTP 请求和 Git 操作
- 支持配置 Agent 工具目录规则
- 支持配置缓存目录和锁文件路径
- 支持配置默认 Registry 和默认仓库源

验收标准：

- 未显式指定配置时，Skillc 能按系统标准路径找到配置文件
- 配置缺失时可自动生成最小默认配置
- 配置项格式错误时能给出明确错误提示

### 9.2 Agent 适配

Skillc 需要通过 Agent 适配层屏蔽不同 Agent 的目录结构差异。

首期支持：

- `claude-code`
- `opencode`
- `codex`

适配要求：

- 每个 Agent 都有独立的目录规则
- 支持全局目录和项目目录的路径解析
- 安装前验证目标目录是否存在，不存在时可自动创建
- 安装后产物结构满足对应 Agent 的加载约定

### 9.3 来源管理

Skillc 需要统一管理本地源与 Git 源。

功能要求：

- 添加本地路径源
- 添加 Git 仓库源
- 列出当前所有来源
- 删除来源
- 手动同步来源
- 查看来源最新同步状态

行为约束：

- 添加本地路径源后，Skillc 需要建立缓存索引
- 添加 Git 仓库源后，Skillc 需要克隆到缓存目录
- 对重复来源进行去重
- 对不可访问的来源保留失败状态，但不影响其他来源

### 9.4 Registry 搜索

Skillc 需要支持从第三方站点搜索 Skill。

功能要求：

- 支持配置多个 Registry 地址
- 支持按关键字搜索
- 支持聚合多个 Registry 结果
- 支持展示来源站点、Skill 名称、描述、支持 Agent、下载入口

行为约束：

- Registry 返回结构不一致时，通过适配层统一为内部模型
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
- 返回来源
- 返回版本信息
- 返回支持的 Agent
- 返回是否已安装
- 返回安装建议或冲突提示

### 9.6 Skill 安装

Skillc 需要支持从缓存安装 Skill 到指定 Agent 目录。

功能要求：

- 指定 Skill、Agent、安装范围执行安装
- 支持从仓库源安装
- 支持从 Registry 解析后安装
- 安装完成后写入锁文件
- 如果目标已存在，可提示覆盖、跳过或升级

安装规则：

- 所有安装动作必须从本地缓存读取
- 对于 Git 源，先同步到缓存再安装
- 对于本地路径源，先更新缓存镜像或索引再安装
- 对于 Registry 源，先解析下载地址并缓存，再执行安装
- 默认采用复制安装，不要求依赖软链接

验收标准：

- 安装成功后，目标 Agent 能在其约定目录中看到 Skill
- 锁文件记录完整
- 同一 Skill 可安装到多个 Agent，但每个安装记录独立追踪

### 9.7 Skill 删除

Skillc 需要支持卸载已安装 Skill。

功能要求：

- 按 Skill、Agent、安装范围删除
- 删除目标目录中的 Skill 产物
- 删除锁文件中的对应记录
- 若目标不存在，应给出幂等提示而不是报错中断

### 9.8 Skill 更新

Skillc 需要支持来源同步和已安装 Skill 升级。

更新分为两层：

- Source Update：同步本地源或远程 Git 源、刷新 Registry 缓存
- Skill Update：对比锁文件中的已安装版本与来源最新版本，执行升级

功能要求：

- 检查某个 Skill 是否可升级
- 列出所有可升级 Skill
- 升级单个 Skill
- 升级指定 Agent 下的一组 Skills

版本判断规则：

- Git 源优先基于 commit/tag 判断
- 本地路径源基于内容指纹或修改时间加指纹判断
- Registry 源基于其返回的版本字段或下载对象摘要判断

### 9.9 已安装状态管理

Skillc 需要提供已安装清单和状态查询能力。

功能要求：

- 列出所有已安装 Skills
- 按 Agent 过滤
- 按安装范围过滤
- 显示来源、版本、安装路径、最近更新时间
- 标记“来源已失效”“存在更新”“目标缺失”等异常状态

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
skillc init
skillc config show

skillc source add local <path>
skillc source add git <url>
skillc source list
skillc source remove <source-id>
skillc source sync [source-id]

skillc search <keyword>
skillc show <skill-id>

skillc install <skill-id> --agent <agent> --scope <global|project>
skillc uninstall <skill-id> --agent <agent> --scope <global|project>
skillc list [--agent <agent>] [--scope <global|project>]
skillc outdated
skillc update [<skill-id>] [--agent <agent>]

skillc cache info
skillc cache clean
skillc cache rebuild
```

说明：

- `source` 用于统一管理本地源和 Git 源
- `search` 用于聚合本地仓库和 Registry 结果
- `show` 用于查看 Skill 详情与来源信息
- `outdated` 用于列出可更新项

## 11. 配置模型

### 11.1 配置原则

- 配置文件使用 YAML
- 路径支持 `~`、相对路径和环境变量展开
- 默认路径遵循操作系统规范
- 用户可以覆盖默认 Agent 安装目录

### 11.2 建议配置示例

```yaml
proxy_url: http://localhost:7890

agent_tools:
  claude-code:
    dirname: .claude
    user_dir: ~/.claude
    project_dir: ./.claude
  opencode:
    dirname: .opencode
    user_dir: ~/.config/opencode
    project_dir: ./.opencode
  codex:
    dirname: .codex
    user_dir: ~/.codex
    project_dir: ./.codex

lock_file: ~/.config/skillc/skillc-install.lock
repo_cache_dir: ~/.cache/skillc/repos
skill_cache_dir: ~/.cache/skillc/skills
registry_cache_dir: ~/.cache/skillc/registry

sources:
  local:
    - id: team-skills
      path: ~/workspace/team-skills
  git:
    - id: awesome-claude-skills
      url: https://github.com/ComposioHQ/awesome-claude-skills
    - id: antigravity-awesome-skills
      url: https://github.com/sickn33/antigravity-awesome-skills

registries:
  - id: skills-sh
    url: https://skills.sh/
  - id: skillsmp
    url: https://skillsmp.com/
  - id: skillsllm
    url: https://skillsllm.com/
```

### 11.3 跨平台要求

Skillc 应兼容 Linux、macOS、Windows。

要求：

- 使用 Go 的标准目录能力解析用户配置目录和缓存目录
- 避免在实现中写死 Unix 风格路径
- 在 Windows 上正确处理盘符、分隔符和用户目录展开

## 12. 数据模型要求

### 12.1 Source 模型

至少包含：

- `id`
- `type`：local/git/registry
- `name`
- `path` 或 `url`
- `last_sync_at`
- `status`
- `error_message`

### 12.2 Skill 元数据模型

至少包含：

- `id`
- `name`
- `description`
- `version`
- `tags`
- `supported_agents`
- `source_id`
- `source_type`
- `install_entry`
- `homepage`

### 12.3 Lock 记录模型

至少包含：

- `skill_id`
- `agent`
- `scope`
- `version`
- `source_id`
- `resolved_ref`
- `installed_path`
- `installed_at`
- `updated_at`

## 13. 非功能需求

### 13.1 性能

- 本地搜索应优先命中缓存索引
- 在 1000 个 Skill 规模下，本地搜索应保持可接受响应速度
- 重复安装已缓存 Skill 时，不应再次访问网络

### 13.2 可靠性

- 单个来源失败不影响整体可用性
- 锁文件写入要具备基本原子性，避免部分写入损坏
- 安装失败时要尽量回滚目标目录到一致状态

### 13.3 可维护性

- Agent 适配器与来源适配器解耦
- Registry 适配器支持后续新增站点
- 内部统一使用标准化 Skill 元数据模型

### 13.4 可观测性

- CLI 输出应明确展示当前操作阶段
- 出错时展示来源、目标和失败原因
- 支持 debug 级日志开关

## 14. 错误与边界场景

Skillc 需要明确处理以下情况：

- 来源地址不可访问
- Git 克隆失败或认证失败
- Registry 返回格式不兼容
- Skill 缺少必要元数据
- 指定 Agent 未配置
- 安装目标目录无权限写入
- 同名 Skill 来自多个来源
- 已安装 Skill 被用户手动删除或改坏

建议策略：

- 对同名冲突输出来源候选并要求用户显式选择
- 对缺少元数据的 Skill 标记为不可安装，但允许在搜索结果中展示警告
- 对目标目录异常给出修复建议

## 15. 安全与信任要求

本期不做复杂安全系统，但至少需要：

- 展示 Skill 来源和下载地址
- 安装前明确目标路径
- 默认不执行任何 Skill 内的脚本
- 对来自 Registry 的跳转和下载地址进行基本校验

## 16. 里程碑建议

### 里程碑 1：本地与 Git 源打通

- 完成配置加载
- 完成来源管理
- 完成缓存目录结构
- 完成 Skill 索引和本地搜索

### 里程碑 2：安装链路打通

- 完成 Agent 适配
- 完成安装、卸载、列表、锁文件
- 完成覆盖与冲突处理

### 里程碑 3：更新与 Registry

- 完成 Registry 搜索
- 完成可升级检查
- 完成更新命令
- 完成缓存清理和重建

## 17. 验收标准

满足以下条件可视为 MVP 可用：

- 用户能添加至少一个本地源和一个 Git 源
- 用户能从来源中搜索并看到标准化 Skill 信息
- 用户能把一个 Skill 安装到指定 Agent 的全局目录或项目目录
- 用户能卸载已安装 Skill
- 用户能查看已安装清单
- 用户能检查并升级某个已安装 Skill
- 相同 Skill 重复安装时能命中缓存而非重复下载
- 锁文件能准确记录安装结果
- 单个来源失败不会导致整个命令不可用

## 18. 待确认问题

以下问题当前 PRD 已做默认假设，但建议在设计阶段最终确认：

- Skill 仓库的标准目录结构是什么，是否要求每个 Skill 自带 `skill.yaml`
- 对现有第三方仓库是否需要兼容“无统一元数据”的场景
- 从 Registry 获取到的 Skill 是否总能映射为可下载包，还是有一部分仅用于跳转展示
- 安装冲突时默认策略是覆盖、重命名还是阻止安装
- 后续是否需要支持软链接开发模式

## 19. 附录：原始需求归纳

原始需求可以归纳为四条主线：

- 来源管理：本地路径、Git 仓库、第三方平台
- 生命周期管理：搜索、下载、安装、删除、更新
- 目标适配：多 Agent、全局目录、项目目录
- 本地状态管理：缓存、锁文件、版本追踪

以上四条主线已经在本 PRD 中展开为可落地的产品需求。
