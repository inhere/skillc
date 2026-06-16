# Skillc v0 增强重构设计

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-12 | v0.1 | Codex | 基于 `docs/TODO.md`、现有 PRD/架构文档和代码结构整理 v0 可重塑方案 |
| 2026-06-12 | v0.2 | Codex | 补充 Web 定位、Registry 后期规划，以及 collection/profile 概念取舍 |
| 2026-06-13 | v0.3 | Codex | 补充当前 CLI 命令全景图和调整后的 CLI 命令组织图 |
| 2026-06-13 | v0.4 | Codex | 调整 v0 最优命令设计：移除顶级 collection 命令，收敛到 source 下 |
| 2026-06-13 | v0.5 | Codex | 复审参考项目，补充参考分析文档链接和 Phase 1 边界校验 |
| 2026-06-14 | v0.6 | Codex | 记录 Phase 1 已落地 profile CLI，并明确 `install --collection` 已从 CLI 移除 |
| 2026-06-14 | v0.7 | Codex | 增加 Phase 2 status/update-check 实施计划链接，并收窄本期实现边界 |
| 2026-06-14 | v0.8 | Codex | 增加 Phase 3 interactive selection 实施计划链接，并明确交互式 TUI 基于 `gookit/cliui` |
| 2026-06-15 | v0.9 | Codex | 记录 Phase 3 已落地 install/update/profile create 交互式选择入口 |
| 2026-06-15 | v0.10 | Codex | 增加 Phase 4 Web 管理实施计划链接，并将下一步建议更新为 Web 管理 MVP |
| 2026-06-15 | v0.11 | Codex | 记录 Phase 4 第一轮 `skillc web` 管理入口已落地，后续继续补 Web 执行能力 |
| 2026-06-15 | v0.12 | Codex | 增加 Phase 5 Web 执行闭环实施计划链接，并明确执行范围只覆盖当前项目 profile apply / update |
| 2026-06-15 | v0.13 | Codex | 记录 Phase 5 已落地 Web 当前项目 profile apply / update 确认执行闭环 |
| 2026-06-15 | v0.14 | Codex | 增加 Phase 6 当前项目 Web 管理补齐实施计划链接和范围 |
| 2026-06-15 | v0.15 | Codex | 记录 Phase 6 已落地当前项目 Web source/profile/uninstall/history 管理闭环 |
| 2026-06-16 | v0.16 | Codex | 增加 Phase 7 跨项目 project registry / update --all-projects 实施计划链接和范围 |
| 2026-06-16 | v0.17 | Codex | 记录 Phase 7 已落地 project registry、`update --all-projects` 和 Web 跨项目更新闭环 |
| 2026-06-16 | v0.18 | Codex | 增加 Phase 8 Registry 发现、source UX cleanup 和精确 drift 实施计划链接 |
| 2026-06-16 | v0.19 | Codex | 记录 Phase 8 已落地 Registry MVP、source UX cleanup 和精确 drift metadata |
| 2026-06-16 | v0.20 | Codex | 复核 PRD 后修正 Registry 定位：P8 为 JSON source catalog 子集，Phase 9 回到 Skill registry 搜索/安装 |
| 2026-06-16 | v0.21 | Codex | 记录 Phase 9 已落地 generic JSON Registry skill search/install、lock provenance 和 registry record restore/status/update |

状态：Draft

相关文档：

- `docs/TODO.md`
- `docs/design/skillc-reference-projects-analysis.md`
- `docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md`
- `docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`
- `docs/superpowers/plans/2026-06-14-skillc-v0-phase3-interactive-selection.md`
- `docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md`
- `docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`
- `docs/superpowers/plans/2026-06-15-skillc-v0-phase6-web-current-project-management.md`
- `docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`
- `docs/superpowers/plans/2026-06-16-skillc-v0-phase8-registry-source-drift.md`
- `docs/superpowers/plans/2026-06-16-skillc-v0-phase9-registry-skill-search-install.md`
- `docs/prd.md`
- `docs/mvp-arch.md`
- `docs/mvp-plan.md`
- `docs/superpowers/specs/2026-04-03-skill-update-design.md`
- `docs/superpowers/specs/2026-04-01-collection-command-design.md`

一期开发计划：`docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md`

二期开发计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`

三期开发计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase3-interactive-selection.md`

三期状态：`install --interactive [keyword]`、`update --interactive`、`profile create <name> --interactive` 已落地。交互入口通过 `internal/infra/termselect` 作为 `gookit/cliui` 薄 adapter，CLI 层使用 fake selector 注入完成测试，业务执行仍复用现有 app service。

四期开发计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md`

四期目标：新增 `skillc web` 本地管理入口，第一轮重点是 source/profile/status/install-map/version-drift 的关系查看和 plan-first 预览；Web 直接执行安装、更新、删除等写操作留到后续小阶段。

四期第一轮状态：已新增 Web 管理查询层、HTTP JSON API、自包含本地管理页面和 `skillc web` CLI 入口。Phase 4 完成时 Web 写入口仍保持 plan-first，只返回 profile apply / update plan 预览，不直接修改安装目录或 lock；Phase 5 已在此基础上补齐当前项目 profile apply / update 的确认执行。

五期开发计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`

五期状态：已在 Phase 4 的 plan-first Web 管理基础上，补齐当前项目范围内 profile apply / update 的计划、确认、执行和刷新闭环。第一轮仍不做卸载、source 删除、跨项目批量更新或远程 Web 权限模型。

六期开发计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase6-web-current-project-management.md`

六期状态：已继续限定在当前项目范围，补齐 Web source add/sync/remove、profile save/from-installed/from-collection、uninstall plan/run 和最小 JSONL 操作历史。跨项目批量更新、Registry、source ID 命名清理、精确 checksum/commit drift 仍后置。

七期开发计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`

七期目标：新增 project registry / project selection / per-project confirmation，并接入 `update --all-projects` 与 Web 跨项目 update plan/run。P7 只处理已登记项目，不直接扫描未知 lock key；Registry 发现、source ID 清理、checksum/Git commit 精确 drift、project manifest 和远程 Web 权限继续后置。

七期状态：已落地 project registry 配置持久化、`skillc project list/add/remove/import-lock`、普通 project scope update 边界修正、`skillc update --all-projects` plan/run，以及 Web registered projects 跨项目 update plan/run。Web 执行仍要求 `confirm:true` 并记录 `update.all_projects` history。

八期开发计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase8-registry-source-drift.md`

八期目标：将 Registry 发现能力、source UX cleanup 和精确 drift 判断拆成三个可独立提交的 slice。P8 先补 `source add <path-or-url> --id/--name`、去掉新 source ID 的 `local-` / `git-` 强前缀并新增 `source info <id>`；再落地本机 registry catalog 的 list/add/remove/sync/search/info/add-source；最后将 Git resolved ref 和 local checksum 写入 index/lock/status/Web，用于同版本内容漂移判断。P8 不做 Registry 信任模型、Web Registry 页面、profile 推荐、旧 source ID 自动迁移或 project manifest。

八期状态：已完成 source UX cleanup、本机/HTTP JSON source catalog 子集，以及基于 Git resolved ref / local directory checksum 的精确 drift metadata。复核 `docs/prd.md` 后确认：P8 Registry 实现只覆盖“内部分享 source catalog”的便利场景，尚未覆盖原 PRD 中“从 skills.sh / skillsmp / skillsllm 等公开站点搜索并安装单个 Skill”的 Registry 主场景。

九期开发计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase9-registry-skill-search-install.md`

九期状态：已修正 Registry 回到 PRD 定位的第一条可执行路径。Generic JSON catalog 支持 `skills` + `sources`；`registry search` 默认返回 Skill 级结果，`registry search --kind source` 保留 P8 source catalog；`registry install <registry>/<skill>` 可直接把单个 registry skill materialize 到本地 cache 并安装，lock 记录 `source_type=registry` 和 registry provenance。`install` restore、`status`、`update --check`、`update` 已能处理 registry-installed skills；skills.sh / SkillsMP / SkillsLLM 等真实站点 adapter 仍后置。

## 1. 设计结论

`skillc` 不应继续只做“skill 包的安装器”。v0 阶段最值得重构的方向是把它升级为：

> 面向多 Agent 的本地技能环境管理器：用 source/registry 发现技能，用 collection 表达来源内的上游分组，用 profile 表达用户在某个项目中想启用的一组技能，用 lock 记录实际落地状态。

当前已有的 `source -> index -> install -> lock -> update` 主链路是对的，不需要推倒重写。真正缺的抽象是 **profile**，也就是 `docs/TODO.md` 中提到的“技能场景”。建议不要直接命名为 `scene`，而是命名为 `profile`，原因是：

- profile 更像开发工具配置语义，例如 `go-dev`、`flutter-dev`、`security-review`。
- scene 容易被理解成 UI 场景或运行时场景，不如 profile 明确。
- profile 可以自然支持 `apply`、`diff`、`export`、`import`、`activate` 等命令。

因此 v0 重构的主线应是：

1. CLI 优先，把“任意项目一键启用技能组合”做顺。
2. 在已有 source/install/update 之上新增 profile 层，collection 收敛为 source 内部的分组视图。
3. 先做可检查、可预览、可回滚的命令模型，再扩展交互式选择。
4. Web 主要用于查看和管理 CLI 不方便表达的关系：source、collection、profile、项目安装分布、版本差异和一键更新；写操作仍复用 CLI 背后的 app service。

## 2. 当前现状

已经具备的能力：

- `source`：支持 local/git 来源管理、部分 ID 匹配、同步、索引重建。
- `search/show`：支持索引搜索和单个 skill 详情查看。
- `collection`：支持集合浏览。
- `install/uninstall/list/update`：已经打通安装、锁文件、恢复、更新主链路。
- `install --source`：支持一条命令注册 source、同步并安装。
- 安装方式：已支持 copy/symlink/junction，并在 Windows 做回退。
- Web：目前 `show --web` 是单个 skill 文件查看器，不是管理后台。
- Registry：P8 已落地本机/HTTP JSON source catalog 子集；P9 已补回 generic JSON Registry skill search/install 主链路。公开 Registry 站点 adapter、信任模型和 Web Registry 页面后置。

当前不顺的地方：

- source、collection、skill、lock 这些概念偏底层，用户想表达的是“这个项目启用一套 Go 开发技能”。
- `collection` 是上游组织维度，不等于用户自己的技能组合。
- `install` 支持批量目标，但缺少持久化组合，不能一键在新项目复用。
- `update` 已经能刷新已安装项，但还没有 `outdated/check` 这种低风险预览。
- Web 当前缺少管理视图，无法方便查看 source、collection、profile 之间的关系，也无法看到 skill 被安装到了哪些项目、哪些版本存在差异。
- source ID 仍带 `local-` / `git-` 前缀，不便于用户查看和引用。

## 3. 核心概念重塑

### 3.1 Source

Source 表示技能来源，继续负责“从哪里来”。

建议调整：

- 用户可见 ID 不再强制带 `local-` / `git-` 前缀。
- `type` 字段继续保留 local/git。
- 自动生成 ID 时若冲突，再加短后缀，而不是通过前缀表达类型。
- `source info <id>` 补齐单个 source 的详细信息。

示例：

```yaml
sources:
  - id: gstack
    type: local
    name: GStack Skills
    path: ~/.agents/skills
  - id: awesome-claude-skills
    type: git
    name: Awesome Claude Skills
    url: https://github.com/example/awesome-claude-skills.git
    ref: main
```

### 3.2 Collection

Collection 表示来源内部已有的组织方式，建议继续保留，但从主用户概念降级为“来源内分组/索引命名空间”。

边界：

- collection 不承担用户自己的技能组合，用户自己的组合统一用 profile。
- collection 浏览可以跨 source 聚合，但安装或导入时必须能回到明确 source。
- collection 不建议作为长期配置目标直接被项目依赖；更好的路径是“从 collection 创建/导入 profile”，创建时展开为明确 skill 列表。
- CLI 中不再保留顶级 `collection` 命令，collection 只作为 `source` 下的分组视图出现。
- Web 中 collection 应作为 source 详情下的分组展示，而不是和 profile 平级强调。

是否还需要 collection：

- 需要，但主要作为 **上游目录结构和索引分组** 存在。
- 不建议把 collection 直接更新为 profile，因为两者生命周期不同：collection 属于 source 作者，profile 属于本机用户或项目。
- 建议提供转换动作：`profile create <name> --from-collection <source>/<collection>`，或在 Web 上提供“保存为 profile”按钮。转换后 profile 默认保存为明确的 skill targets，而不是保存动态 collection 引用。

如果未来发现 collection 仍造成强混淆，可以在用户界面里改名为 `group` 或 `source group`，内部数据字段可继续兼容 `collection`。CLI 层优先通过 `source collections` / `source skills --collection` 暴露，不再给它顶级入口。

### 3.3 Profile

Profile 是新增核心概念，表示用户保存的一组安装目标和默认落点。

它回答的是：

- 当前项目需要哪些 skills？
- 安装到哪个 agent？
- 安装到 project 还是 global？
- 使用 copy、symlink 还是 junction？
- 是否来自多个 source？
- 是否由某个 source collection 导入生成？

建议配置结构：

```yaml
profiles:
  go-dev:
    description: Go project development skills
    default_agent: universal
    default_scope: project
    install_mode: symlink
    targets:
      - source: gstack
        skill: go-pro
      - source: gstack
        skill: test-driven-development
      - source: gstack
        skill: investigate
      - source: gstack
        skill: review
      - source: team-skills
        skill: code-review
  flutter-dev:
    description: Flutter app development skills
    default_agent: universal
    default_scope: project
    targets:
      - source: gstack
        skill: flutter-dev
      - source: gstack
        skill: frontend-design
```

Profile target 应优先支持两种稳定粒度：

- `skill`：安装单个 skill。
- `source + skill`：处理同名歧义，也是从 collection 生成 profile 后的默认持久化形态。

Collection 只作为 profile 创建输入，不作为 profile 默认持久化目标。这样 `source sync` 后上游 collection 增删 skill，不会悄悄改变项目期望安装集。若未来确实需要动态跟随 collection，可单独设计 `include_collection`，并要求用户显式 opt-in。

推荐用户路径是：

```bash
skillc profile create go-dev --from-collection gstack/go
skillc profile edit go-dev
skillc profile apply go-dev
```

而不是历史上的 collection 直装路径：

```bash
skillc install --collection go
```

在 v0 最优设计里，后者不再作为推荐命令；Phase 1 已从 CLI 中移除该 flag，新文档和新用户路径不应强调它。

### 3.4 Registry

Registry 是旧 PRD 中已经出现的第三方 Skill 搜索来源，目标是从 skills.sh、skillsmp、skillsllm 这类公开收集站点或团队内部 catalog 搜索 Skill，筛选后安装到本地试用。Phase 8 已落地的是一个更窄的 JSON source catalog 子集，用于发现可复用的 source entry；它有价值，但不等于完整 Registry。

定位：

- Source 管理“我已经知道的本地路径或 Git 仓库”。
- Registry 管理“我还不知道具体仓库或 Skill 时，去哪里搜索发现 skill/source/profile”。
- Registry 的 Skill 结果可以直接下载到本地 registry skill cache 后安装，lock 记录 registry provenance。
- Registry 的 Source 结果可以通过 `registry add-source` 转成长期 source 管理；这是可选入口，不是安装单个 Skill 的必经步骤。

P8 已落地命令：

```bash
skillc registry list
skillc registry add <path-or-url> --id <id> --name <name>
skillc registry sync <id>
skillc registry sync --all
skillc registry search <keyword>
skillc registry info <entry-id>
skillc registry add-source <entry-id> [--id <id>] [--name <name>] [--sync]
```

P8 限制：只支持本机/HTTP JSON catalog 的 `sources` 部分，不支持 `skills` 部分，不做 registry skill install。`registry search` 当前只读 source catalog cache，`registry add-source` 只写 source 配置；安装仍由 source sync/search/install/update 主链路处理。

P9 已落地命令：

```bash
skillc registry search <keyword> [--registry <id>]
skillc registry search <keyword> --kind source
skillc registry info <registry>/<skill>
skillc registry info <registry>/<source> --kind source
skillc registry install <registry>/<skill> --agent codex --scope project
skillc registry install <registry>/<skill> --yes
```

P9 继续保留 `registry add-source`，但只用于“把搜索结果背后的 Git/source 加入长期 source 管理”。公开 registry 搜到单个 Skill 时，用户不再需要先 add-source 再 source sync 再 install。当前 P9 实现 generic JSON catalog；真实 skills.sh / SkillsMP / SkillsLLM adapter、archive download/extract、签名和信任策略后置。

### 3.5 Lock

Lock 继续记录“实际装了什么”。

建议增强：

- 每条记录增加 `profile` 字段，记录来源于哪个 profile apply。
- 增加 `desired` 语义可以延后，不建议 v0 一开始做双文件状态机。
- `profile apply` 成功后仍写现有 lock，避免拆出一套新的安装事实。

示例字段：

```json
{
  "skill_id": "go-pro",
  "profile": "go-dev",
  "source_id": "team-skills",
  "agents": ["universal"],
  "pinned": false
}
```

## 4. CLI 命令组织

### 4.1 当前 CLI 命令全景图

基于当前 `internal/cli/*.go` 的注册代码，现有命令树是：

```text
skillc
├─ config (cfg)
│  ├─ init
│  ├─ show
│  ├─ get <key>
│  └─ set <key> <value>
│
├─ source (src)
│  ├─ add
│  │  ├─ local <path> [--sync]
│  │  └─ git <url> [ref] [--sync]
│  ├─ list (ls)
│  ├─ sync (up) <id>|--all
│  ├─ status (st)
│  └─ remove (rm) <id>
│
├─ collection (coll)
│  ├─ list (ls)
│  └─ skills <collection>
│
├─ search (find) <keyword>
│  ├─ --agent <agent>
│  └─ --source-type <git|local>
│
├─ show <skill>
│  ├─ --web
│  └─ --port <port>
│
├─ install (ins) [skill]
│  ├─ --source <path-or-git-url>
│  ├─ --collection
│  ├─ --scope <project|global>
│  ├─ --agent <agent>
│  ├─ --copy
│  ├─ --install-mode <symlink|junction|copy>
│  └─ --yes
│
├─ update (up) [skill]
│  ├─ --target <skill>
│  ├─ --scope <project|global>
│  └─ --agent <agent>
│
├─ uninstall (uni|remove|rm) <skill...>
│  ├─ --scope <project|global>
│  └─ --agent <agent>
│
├─ list (ls)
│  ├─ --scope <project|global>
│  └─ --agent <agent>
│
└─ doctor
```

现状判断：

- 当前命令是“来源管理 + skill 生命周期管理”的组织方式，已经能覆盖 source、search、install、list、update、uninstall。
- `collection` 作为顶级命令存在，容易让用户误以为它和未来的 `profile` 是同一层概念。
- `show --web` 是单个 skill 查看器，不是管理入口。
- `list` 只能看当前 agent/scope 的安装状态，不能回答“这个 skill 被装到了哪些项目”。
- `update` 偏执行动作，还缺少 `--check` / `outdated` 这种不改文件的预览入口。
- `install --collection` 曾是快捷能力，但如果继续作为主路径，会加剧 collection/profile 混淆；Phase 1 已移除该 CLI flag。

### 4.2 调整后的 CLI 命令组织图

v0 仍在开发期，可以做破坏性调整。推荐新增 `profile`、`status`、`web`，并移除顶级 `collection` 命令，把 collection 收敛为 `source` 下的分组视图。调整后的命令树建议为：

```text
skillc
├─ config (cfg)
│  ├─ init
│  ├─ show
│  ├─ get <key>
│  └─ set <key> <value>
│
├─ source (src)
│  ├─ add <path-or-git-url> [--id <id>] [--name <name>] [--ref <ref>] [--sync]
│  ├─ add local <path> [--id <id>] [--sync]
│  ├─ add git <url> [ref] [--id <id>] [--sync]
│  ├─ list (ls)
│  ├─ info <id>
│  ├─ sync (up) [id|--all]
│  ├─ status (st) [id]
│  ├─ collections [source-id] [--json]
│  ├─ skills <source-id> [--collection <name>] [--json]
│  └─ remove (rm) <id>
│
├─ profile
│  ├─ list (ls)
│  ├─ show <name>
│  ├─ create <name> [--from-installed] [--agent <agent>] [--scope <scope>]
│  ├─ create <name> --from-collection <source>/<collection>
│  ├─ add <name> <target...>
│  ├─ remove <name> <target...>
│  ├─ diff <name> [--agent <agent>] [--scope <scope>]
│  ├─ apply <name> [--agent <agent>] [--scope <scope>] [--dry-run] [--yes]
│  ├─ export <name> [--output <file>]
│  └─ import <file>
│
├─ status
│  ├─ --profile <name>
│  ├─ --agent <agent>
│  ├─ --scope <project|global>
│  ├─ --project <path>
│  └─ --json
│
├─ search (find) <keyword>
│  ├─ --agent <agent>
│  ├─ --source <source-id>
│  ├─ --source-type <git|local|registry>
│  ├─ --installed
│  └─ --json
│
├─ show <skill>
│  ├─ --web
│  └─ --port <port>
│
├─ install (ins) <target...>
│  ├─ --source <path-or-git-url>
│  ├─ --interactive
│  ├─ --scope <project|global>
│  ├─ --agent <agent>
│  ├─ --copy
│  ├─ --install-mode <symlink|junction|copy>
│  ├─ --dry-run
│  └─ --yes
│
├─ update (up) [target]
│  ├─ --check
│  ├─ --interactive
│  ├─ --profile <name>
│  ├─ --all-projects
│  ├─ --scope <project|global>
│  ├─ --agent <agent>
│  ├─ --dry-run
│  └─ --yes
│
├─ uninstall (uni|remove|rm) <skill...>
│  ├─ --scope <project|global>
│  ├─ --agent <agent>
│  ├─ --dry-run
│  └─ --yes
│
├─ list (ls)
│  ├─ --scope <project|global>
│  ├─ --agent <agent>
│  ├─ --profile <name>
│  └─ --json
│
├─ web
│  ├─ --host 127.0.0.1
│  └─ --port 8080
│
├─ registry
│  ├─ add <url> [--id <id>] [--name <name>]
│  ├─ list (ls)
│  ├─ search <keyword>
│  └─ remove (rm) <id>
│
└─ doctor
```

组织原则：

- `source` 负责“技能从哪里来”，补齐 `source add <path-or-url>` 和 `source info <id>`，解决当前 source 查看和命名不顺的问题。
- `source collections` 和 `source skills --collection` 负责浏览来源内部的上游分组，替代当前顶级 `collection` 命令。
- 不再新增 `source collection ...` 三级子树，优先使用较短的 `source collections` / `source skills`，避免 source 管理命令过深。
- `profile` 负责“项目想启用什么组合”，是新增主概念。
- `status` 负责“当前项目或指定项目的状态”，比 `list` 更适合做健康总览和差异预览。
- `collection` 不再作为顶级命令；从 collection 到项目配置的主路径是 `profile create --from-collection`。
- `install` 不再推荐 `--collection`，collection 安装先转 profile，再 apply。
- `web` 是 source/collection/profile/install-map/version-drift 的关系管理入口，不只是 `show --web` 的升级版。
- `registry` 放在后期实现，但命令位置先预留，避免未来混入 `source` 造成概念膨胀。
- `install/update/uninstall/list/search/show` 继续保留顶层命令，保证当前用户习惯和脚本兼容。

### 4.3 常用路径优先

推荐把用户日常路径收敛成四条：

```bash
skillc source add <path-or-git-url> [--id <id>] [--sync]
skillc profile apply go-dev
skillc status
skillc update --check
```

这比要求用户理解 `source sync`、`source collections`、`list`、`update` 的组合更顺。collection 仍可在 source 下面浏览，但不应作为新用户的主入口；collection 到安装的路径应先生成 profile。

### 4.4 Source 命令

保留现有命令，同时增加更短入口：

```bash
skillc source add <path-or-git-url> [--id <id>] [--name <name>] [--ref <ref>] [--sync]
skillc source add local <path> [--id <id>] [--sync]
skillc source add git <url> [ref] [--id <id>] [--sync]
skillc source list
skillc source info <id>
skillc source sync [id|--all]
skillc source collections [source-id]
skillc source skills <source-id> [--collection <name>]
skillc source remove <id>
```

设计要点：

- `source add <path-or-url>` 是推荐入口，自动判断 local/git。
- `source add local/git` 保留给明确用户。
- `source info` 解决 TODO 中“无法方便查看一个 source 信息”的问题。
- `source collections` 替代原顶级 `collection list`，强调 collection 只是 source 内部分组。
- `source skills <source-id> --collection <name>` 替代原顶级 `collection skills <collection>`，避免跨 source 歧义。
- source ID 展示不加 `local-` / `git-` 前缀，类型通过列展示。

### 4.5 Profile 命令

新增：

```bash
skillc profile list
skillc profile show <name>
skillc profile create <name> [--from-installed] [--agent <agent>] [--scope <scope>]
skillc profile create <name> --from-collection <source>/<collection>
skillc profile add <name> <target...>
skillc profile remove <name> <target...>
skillc profile apply <name> [--agent <agent>] [--scope <scope>] [--dry-run] [--yes]
skillc profile diff <name>
skillc profile export <name> [--output skillc.profile.yaml]
skillc profile import <file>
```

最小闭环：

- `profile create go-dev --from-installed`：从当前项目已安装 skills 生成 profile。
- `profile create go-dev --from-collection gstack/go`：把上游 collection 转成本机可编辑 profile。
- `profile apply go-dev`：在任意项目安装该组合。
- `profile diff go-dev`：展示 profile 期望与当前 lock/目录的差异。

`profile apply` 的输出应像部署计划：

```text
Profile: go-dev
Scope: project
Agent: universal

Plan:
  install  go-pro                  source=team-skills
  install  test-driven-development source=gstack
  skip     review                  already installed
  warn     investigate             source not synced

Continue? [y/N]
```

### 4.6 Status / Outdated

把更新拆成“检查”和“执行”：

```bash
skillc status [--profile <name>] [--agent <agent>] [--scope <scope>]
skillc update --check [--agent <agent>] [--scope <scope>]
skillc update [target] [--agent <agent>] [--scope <scope>] [--yes]
```

`status` 负责用户态总览：

- installed
- missing
- outdated
- orphan
- unmanaged
- source-error

`update --check` 只同步/读取元数据并比较，不改安装目录。

版本判断优先级：

1. Git source：比较 `resolved_ref` 或 commit。
2. Local source：比较内容 checksum。
3. Skill metadata：若 `version` 存在，作为展示和辅助判断。

### 4.7 交互式选择

交互式选择应作为 CLI 的增强，而不是默认唯一入口：

```bash
skillc install --interactive
skillc update --interactive
skillc profile create go-dev --interactive
```

本期交互式 TUI 复用已引入的 `gookit/cliui`，优先使用 `interact` / `interact/ui` / `interact/backend` 已支持的过滤、多选和 fake backend 测试能力；`skillc` 只封装薄 adapter 和 CLI 注入边界。

交互功能建议：

- 支持关键字过滤。
- 支持多选。
- 展示 source、collection、version、installed/outdated 状态。
- 安装前统一生成 plan，再确认执行。

非交互命令必须保持完整，方便脚本化和测试。

### 4.8 Web 管理

建议新增独立命令：

```bash
skillc web [--port 8080]
```

Web 的主要价值不是替代 CLI，而是补齐 CLI 不擅长的“查看关系和批量管理”。因此 Web v0 范围建议调整为：

- Dashboard：当前项目、agent、scope、profile 状态，以及安装健康摘要。
- Sources：查看、同步、删除 source；查看每个 source 下的 collections 和 skills。
- Collections：作为 source 详情中的分组视图展示，支持“保存为 profile”，不作为长期主配置概念强调。
- Profiles：创建、编辑、复制、导入、应用 profile；查看 profile 在当前项目的 diff。
- Install Map：查看每个 skill/profile 被安装到了哪些项目、哪些 agent、哪些 scope。
- Version Drift：查看同一 skill 在不同项目中的来源版本、checksum、commit 差异，并支持一键更新。
- Skills：搜索、查看详情、查看安装分布；安装/卸载作为次级操作。
- Logs/Plan：展示每次操作的执行计划和结果。

关键约束：

- Web 只调用 app service，不直接写文件、不复制业务规则。
- 所有写操作必须先返回 plan，再由用户确认。
- Web 操作结果应与 CLI 输出语义一致。
- 现有 `show --web` 可以保留为 skill viewer，但内部可复用到新 Web 的详情页。
- Web 默认监听 `127.0.0.1`，不做远程管理和账号体系。

## 5. 三个可选方案

### 方案 A：最小可用 profile 层

摘要：

在现有架构上新增 `profile` 配置模型和 CLI 命令。先支持 profile 的 create/show/apply/diff，并支持从 installed 或 collection 生成 profile，复用现有 install/update/source/search 服务。

Effort：M
Risk：Low

优点：

- 改动贴近现有架构，不破坏 install/update 主链路。
- 最快解决“任意项目一键启用技能组合”的核心痛点。
- 能把 collection 移出顶层命令，降级为 source 内部分组，同时通过“保存为 profile”解决概念混淆。
- 后续 Web 可以自然复用 profile 服务。

缺点：

- 交互式选择、Web 管理、outdated 精确判断仍需后续阶段补齐。
- Web 中的安装分布和版本差异视图仍需 Phase 2/4 才能完整。
- 如果 profile 模型设计太窄，后续可能需要迁移配置字段。

复用：

- `installapp.Service`
- `searchapp.ResolveInstallTargets`
- `lockstore`
- `agent.ResolveInstallPath`
- `sourceapp.Sync`

### 方案 B：完整项目环境管理器

摘要：

一次性引入 profile、status、outdated、安装分布、版本差异、交互式选择、Web 管理，并把 CLI/Web 都统一到 plan executor。

Effort：L/XL
Risk：Med

优点：

- 用户体验最完整，工具定位一次成型。
- CLI 与 Web 都围绕 plan/apply，可维护性最好。
- 能系统性解决 TODO 中所有体验问题，包括跨项目版本差异和一键更新。

缺点：

- 单次改动涉及配置、锁文件、CLI、Web、update、测试，范围较大。
- v0 仍在快速变化，过早抽象 plan executor 可能有设计浪费。

复用：

- 所有现有 service。
- 需要新增 `planapp` 或 `operationapp` 作为统一执行计划层。

### 方案 C：Web-first 管理器

摘要：

优先扩展 Web 管理界面，把 source、collection、profile、安装分布、版本差异、update 都做成可视化操作，CLI 只保留底层命令。

Effort：L
Risk：High

优点：

- 对新用户更友好。
- 适合浏览、比较、批量选择 skills。
- 视觉上更接近 `skills-manager` 参考项目。

缺点：

- 与 TODO 中“CLI 优先”的方向相反。
- Web 写操作会迫使后端 API 和权限边界提前成型。
- 对 Go CLI 项目来说验证成本更高，容易把核心模型问题藏在 UI 里。

复用：

- 现有 `webapp` 的文件查看能力可保留，但管理功能基本需要新建 API 和页面结构。

## 6. 推荐方案

推荐采用 **方案 A：最小可用 profile 层**。

理由：

- 当前项目的主链路已经可用，缺的是一个用户意图层，不是底层重写。
- profile 是 TODO 中“技能场景”“不同组合”“任意项目一键安装启用”的共同抽象。
- 方案 A 可在较小范围内验证命令语义，避免一次把 Web、交互式、多状态 diff 全部压进来。
- v0 可以任意调整，但仍应保护已经跑通的 source/index/install/lock/update 分层。

建议把 v0.2 的定义设为：

> 用户可以把多个 skill 或上游 collection 保存为 profile，并在任何项目目录用一条命令预览和应用该 profile。collection 只作为来源内分组和 profile 创建输入，不作为用户长期维护项目技能组合的主概念。

## 7. 目标用户路径

### 7.1 首次配置

```bash
skillc config init
skillc source add ~/.agents/skills --id gstack --sync
skillc search go
```

### 7.2 从已安装内容生成 profile

```bash
skillc install go-pro,test-driven-development --yes
skillc profile create go-dev --from-installed
skillc profile show go-dev
```

### 7.3 在新项目启用 profile

```bash
cd another-go-project
skillc profile diff go-dev
skillc profile apply go-dev
```

### 7.4 检查和更新

```bash
skillc status
skillc update --check
skillc update --yes
```

## 8. 推荐数据模型

### 8.1 Config 增加 Profiles

```go
type Config struct {
    // existing fields...
    Profiles map[string]Profile `yaml:"profiles,omitempty"`
}

type Profile struct {
    Description  string          `yaml:"description,omitempty"`
    DefaultAgent string          `yaml:"default_agent,omitempty"`
    DefaultScope string          `yaml:"default_scope,omitempty"`
    InstallMode  string          `yaml:"install_mode,omitempty"`
    Targets      []ProfileTarget `yaml:"targets"`
}

type ProfileTarget struct {
    Source string `yaml:"source,omitempty"`
    Skill  string `yaml:"skill"`
    Pinned bool   `yaml:"pinned,omitempty"`
}
```

`profile create --from-collection <source>/<collection>` 负责把 collection 内的 skills 展开为多条 `ProfileTarget`。profile 文件不默认保存 collection 引用。

### 8.2 Profile Apply Plan

```go
type ProfilePlan struct {
    Profile string
    Agent   string
    Scope   string
    Items   []ProfilePlanItem
}

type ProfilePlanItem struct {
    Action string // install | skip | update | error
    Target string
    Skill  skill.Skill
    Reason string
}
```

Profile apply 不应直接边解析边安装，应先生成 plan：

1. 读取 profile。
2. 解析 target 到 skill 列表。
3. 对照 lock/list 判断当前状态。
4. 输出 plan。
5. 用户确认后调用 install 服务执行。

### 8.3 Registry 扩展模型

Registry 作为独立配置块加入。P8 已实现 `registries` 配置和 JSON source catalog cache；P9 需要扩展 provider 类型和 skill 级搜索结果。

```go
type Config struct {
    // existing fields...
    Registries []Registry `yaml:"registries,omitempty"`
}

type Registry struct {
    ID      string `yaml:"id"`
    Name    string `yaml:"name,omitempty"`
    Type    string `yaml:"type,omitempty"`    // json | api | site
    Adapter string `yaml:"adapter,omitempty"` // generic-json | skills-sh | skillsmp | skillsllm
    URL     string `yaml:"url"`
    Status  string `yaml:"status,omitempty"`
}

type RegistrySkill struct {
    RegistryID      string   `json:"registry_id"`
    ID              string   `json:"id"`
    Name            string   `json:"name,omitempty"`
    Description     string   `json:"description,omitempty"`
    Version         string   `json:"version,omitempty"`
    SupportedAgents []string `json:"supported_agents,omitempty"`
    DownloadURL     string   `json:"download_url,omitempty"`
    SourceURL       string   `json:"source_url,omitempty"`
    SourceRef       string   `json:"source_ref,omitempty"`
    InstallEntry    string   `json:"install_entry,omitempty"`
    Checksum         string   `json:"checksum,omitempty"`
    Tags            []string `json:"tags,omitempty"`
}
```

Registry search 的结果不直接写 lock。安装前应解析为可缓存的 skill snapshot 或 source snapshot，再进入统一 install/profile/lock 流程。单个 registry skill 不需要先注册为 source；只有用户想长期订阅结果背后的仓库时，才使用 `registry add-source`。

## 9. 分层落点

建议新增：

```text
internal/app/profileapp/service.go
internal/domain/profile/model.go
internal/cli/profile_cmd.go
```

可选后续：

```text
internal/app/statusapp/service.go
internal/app/planapp/service.go
internal/app/registryapp/service.go
internal/infra/termselect/
```

职责边界：

- `domain/profile`：只定义 profile 和 target 校验规则。
- `profileapp`：负责 profile CRUD、从 lock 生成 profile、生成 apply plan、执行 apply。
- `cli/profile_cmd.go`：只负责参数解析和输出。
- `installapp`：继续只负责安装。
- `searchapp`：继续负责 target 解析。

## 10. 分阶段实施建议

### Phase 1：Profile 最小闭环

目标：任意项目一键启用技能组合。

任务：

- 增加 profile 配置模型。
- 增加 `profile list/show/create --from-installed`。
- 增加 `profile create --from-collection <source>/<collection>`，并将 collection 展开为明确 skill targets。
- 增加 `profile apply --dry-run/--yes`。
- 增加 profile apply 的计划输出。
- lock record 增加可选 `profile` 字段。
- 补单测和 CLI 测试。

验收：

- 可从当前项目已安装 skills 生成 profile。
- 可在另一个临时项目 apply profile。
- 重复 apply 能正确 skip 已安装项。
- `go test ./...` 通过。

### Phase 2：Status 和 update check

目标：用户先能看到当前项目的技能健康状态和更新候选；安装分布作为后续扩展进入 Web/version-drift 能力。

实施计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`

本期先落地当前项目维度的 `status` 和 `update --check`，以非破坏性预览为主。跨项目安装分布、Git `resolved_ref` 漂移、Local checksum 漂移继续保留为 Phase 2 后续扩展或 Phase 4 Web 的输入能力，不压入第一轮实现。

任务：

- 增加 `status` 命令。
- 增加 `update --check` 或 `outdated`。
- 当前项目输出 installed/missing/outdated/orphan/unmanaged/source-error。
- v0 第一轮 outdated 只比较非空 metadata version，checksum/git commit drift 后置。
- 后续增加安装分布查询模型：skill/profile -> projects/agents/scopes。

验收：

- 已安装 skill 的 metadata version 落后于 index version 时能显示 outdated。
- lock 中记录但 index 找不到时能显示 orphan。
- 安装目录存在但 lock 无记录时能显示 unmanaged。
- check 不修改安装目录和 lock。

### Phase 3：交互式选择

目标：安装、更新、profile 创建支持多选和过滤。

实施计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase3-interactive-selection.md`

当前状态：已完成第一轮实现，`install --interactive [keyword]` 从索引候选中多选安装，`update --interactive` 只展示 `outdated` / `missing` 候选并逐项更新，`profile create <name> --interactive` 将选中 Skills 保存为明确 profile targets。

任务：

- 基于已引入的 `gookit/cliui` 封装最小交互接口。
- `install --interactive`。
- `update --interactive`。
- `profile create --interactive`。
- 所有交互最终仍生成 plan。

验收：

- 非交互测试不受影响。
- 交互逻辑可通过注入输入输出测试。

### Phase 4：Web 管理

目标：提供本地管理 UI，重点解决 CLI 不方便查看和配置的关系型信息。

实施计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md`

任务：

- 新增 `skillc web`。
- 提供 source/collection/skill/profile/status/install-map/version-drift 查询 API。
- 从现有 lock 记录构建轻量 project install map，不在 v0 第一轮引入数据库。
- 复用 profile apply plan。
- 第一轮 Web 写入口只返回 plan，不直接执行安装、更新或删除；执行能力放到后续小阶段。
- 将现有 skill viewer 纳入 skill detail 页面。

验收：

- Web 可查看 source、collection、profile 和 skill 安装分布。
- Web 可查看基于 lock/index 的跨项目版本差异，并生成一键更新计划预览。
- Web 可 dry-run apply profile。
- Web 不绕过 CLI/app service 的业务规则；后续执行能力必须复用同一套 app service。

### Phase 5：Web 执行闭环

目标：在 Phase 4 的 Web 查看和 plan-first 基础上，补齐当前项目范围内 profile apply / update 的确认执行闭环。

实施计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`

已落地：

- Web Profiles 页可以先生成 profile apply plan，再由用户确认执行。
- Web Dashboard / Version Drift 可以先生成 update plan，再由用户确认执行当前项目更新。
- 后端执行端点要求请求体包含 `confirm:true`，不能只依赖前端确认。
- HTTP handler 只做请求解析、确认校验和 Web JSON 结果转换，实际执行复用 `profileapp.Apply` 和 `updateapp.Run`。
- 执行完成后静态 UI 刷新 summary/status/install-map/version-drift。

仍后置：

- Web 卸载 skill。
- Web 删除 source。
- 跨项目批量更新所有下游项目。
- 操作历史、审计日志、远程访问和多用户权限模型。

### Phase 6：当前项目 Web 管理补齐

目标：在 Phase 5 当前项目 profile apply / update 执行闭环之上，补齐 Web 管理面的剩余当前项目写操作，让用户无需退回 CLI 即可完成 source、profile 和 uninstall 的常见管理动作。

实施计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase6-web-current-project-management.md`

已落地：

- Web source 管理：source add/sync/remove 均先生成 plan；remove plan 展示 installed lock record、profile target、indexed skill、collection 影响。
- Web profile 管理：支持 profile save、from-installed、from-collection 的 plan/run；保存前展示 added/removed/kept target 差异。
- Web uninstall：先生成 uninstall plan，明确 agent/scope/path/lock record，再要求 `confirm:true` 执行。
- Web history：以本地 JSONL 记录 Web 写操作的 action、request、plan/result、error 和时间，作为排查用的最小历史。

仍后置：

- Registry 发现能力。
- source ID 去掉 `local-` / `git-` 前缀、自定义 `--id/--name` 和 `source info <id>`。
- checksum / Git commit drift 精确判断。
- 远程 Web 访问、多用户权限和安全审计模型。

### Phase 7：跨项目更新闭环

目标：把 Phase 4/5/6 已经能查看的安装分布和版本差异，推进到安全的跨项目更新执行闭环。核心是先定义本机 project registry，只有已登记项目才能被 `update --all-projects` 和 Web 跨项目 update 操作触达。

实施计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`

已落地：

- Project registry：config 中新增 `projects`，CLI 提供 `skillc project list/add/remove/import-lock`。
- Update 边界修正：普通 project scope `skillc update` 只处理当前项目，跨项目能力必须显式使用 `--all-projects`。
- Cross-project update：新增 app service 生成跨项目 plan，并按 project 执行 `outdated` / `missing` 候选更新。
- CLI：新增 `skillc update --all-projects --check`、`--projects <id,id>` 和 `--yes` 确认执行路径。
- Web：新增 registered projects 查询、跨项目 update plan/run endpoints 和 Projects 视图中的选择/确认控件。
- History：Web 跨项目执行记录为 `update.all_projects`。

继续后置：

- Registry 发现能力。
- source ID 去掉 `local-` / `git-` 前缀、自定义 `--id/--name` 和 `source info <id>`。
- checksum / Git commit drift 精确判断。
- `skillc.profile.yaml` 项目 manifest / profile export/import 增强。
- 远程 Web 访问、多用户权限和安全审计模型。

## 11. 风险与取舍

### 11.1 Profile 与 Collection 混淆

风险：用户可能把 collection 当 profile。

处理：

- 结论：保留 collection，但降级为来源内分组/索引命名空间。
- 文档中明确：collection 属于 source 作者，profile 属于本机用户或项目。
- CLI 中移除顶级 `collection`，改为 `source collections` 和 `source skills --collection`。
- Web 中 collection 放在 source 详情下，不和 profile 平级突出。
- 提供显式转换：从 collection 保存为 profile。
- 命令名区分：`source collections` 是浏览，`profile apply` 是启用。

不推荐把 collection 直接更新为 profile：

- collection 来自上游 source，用户不一定能控制其内容变更。
- profile 是本机或项目的期望状态，应该稳定、可编辑、可审查。
- 如果两者合并，source 更新可能悄悄改变项目期望安装集，风险更高。

可以接受的折中：

- source 仓库未来可以声明“推荐 profiles”，但这应是新的 `profiles` metadata，不复用 collection。
- 用户可以一键把 collection 导入为 profile，然后 profile 后续独立演进；导入后默认不动态跟随 collection。

### 11.2 配置文件膨胀

风险：profiles 放入主 config 后，团队共享和个人配置混在一起。

处理：

- v0 先放主 config，简单可用。
- 后续支持 `skillc.profile.yaml` 项目文件或 `profile export/import`。

### 11.3 Apply 的删除语义

风险：profile apply 是否应删除 profile 中不再包含的已安装 skill。

建议：

- v0 不自动删除。
- 只安装缺失项、更新 profile 管理项。
- 后续通过 `profile sync --prune` 显式启用删除。

### 11.4 Web 操作安全

风险：Web 管理端如果直接执行写操作，容易误删或误装。

处理：

- 所有写操作先 plan。
- 本地监听默认 `127.0.0.1`。
- 不默认开放远程访问。

### 11.5 Registry 边界修正

风险：P8 把 Registry 限定为 source catalog，虽然避免了绕过 source/index/install/lock/profile 主链路，但偏离了 `docs/prd.md` 中“从第三方 Registry 搜索并安装 Skill”的主场景。P9 需要补回 skill 级 Registry，同时继续避免把远程结果直接写入 lock。

处理：

- Phase 8 保留为本机/HTTP JSON source catalog 子集，服务内部分享 source 的场景。
- Phase 9 增加 registry skill result、skill cache snapshot 和 `registry install`。
- `registry install` 必须先下载/解析到本地 cache，再复用 install service 写 lock；lock 记录 `source_type=registry` 和 registry provenance。
- `registry add-source` 继续只注册 source，不安装 Skill、不写 lock，定位为长期订阅 source 的可选入口。
- Registry 信任模型、Web Registry 页面、profile 推荐和签名校验继续后置。

## 12. 成功标准

v0 增强重构完成后，应满足：

- 用户能清楚区分 source、profile、lock；collection 只是 source 下的分组视图。
- 用户能把当前项目的一组 skills 保存成 profile。
- 用户能从上游 collection 生成 profile，而不是长期把 collection 当项目配置。
- 用户能在任意项目一条命令应用 profile。
- 所有安装/更新动作可以先预览。
- 用户能检查已安装 skills 是否过期。
- 用户能登记允许管理的本机项目，并对这些项目执行跨项目 update plan/run。
- Web 能查看 source、collection、profile、安装分布和版本差异，并对 registered projects 执行确认后的跨项目更新。
- CLI 仍能完成核心写操作；Web 是关系查看和批量管理入口。
- Registry 能从至少一个 JSON skill catalog 搜索并安装 Skill；P8 的 source catalog 能力保留为内部分享入口，单个 registry skill 不要求先 add-source。

## 13. 下一步建议

Phase 1/2/3/4/5/6/7/8/9 已经完成：profile 最小闭环、当前项目 status/update check、基于 `gookit/cliui` 的交互式选择、`skillc web` 本地管理查看、当前项目 profile apply / update 确认执行闭环、当前项目 Web source/profile/uninstall/history 管理补齐、project registry / `update --all-projects` / Web 跨项目更新闭环、source UX cleanup / JSON source catalog 子集 / 精确 drift metadata，以及 generic JSON Registry skill search/install、lock provenance 和 registry record restore/status/update 都已落地。

Phase 5 实施计划见：`docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`

Phase 6 实施计划见：`docs/superpowers/plans/2026-06-15-skillc-v0-phase6-web-current-project-management.md`

Phase 7 实施计划见：`docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`

Phase 8 实施计划见：`docs/superpowers/plans/2026-06-16-skillc-v0-phase8-registry-source-drift.md`

Phase 9 实施计划见：`docs/superpowers/plans/2026-06-16-skillc-v0-phase9-registry-skill-search-install.md`

下一步建议转向协作与治理能力：

- Project manifest / profile export-import：设计 `skillc.profile.yaml` 或 profile export/import，解决团队共享 profile 的落点。
- Registry 信任模型：catalog entry 签名、checksum、来源 allow/deny policy 和远程 registry 安全边界。
- Registry adapters：接入 skills.sh、SkillsMP、SkillsLLM 等公开站点搜索 API。
- Web Registry 页面：浏览 registry skill/source result、registry install plan/run、add-source plan/run 和 registry sync 状态可视化。
- Remote Web：远程访问、多用户权限和安全审计单独设计，不复用当前本地 JSONL history 作为安全审计系统。

Phase 7 已经把跨项目更新收敛到 registered projects allowlist。后续继续坚持当前安全边界：先 plan、后确认、handler 保持薄层、执行复用 app service。

## 14. 参考项目复审补充

参考分析见：`docs/design/skillc-reference-projects-analysis.md`。

复审 `skills-manager`、`skillshare`、`apm` 后，当前设计主线保持不变：Phase 1 继续优先做 profile 最小闭环，Web、Registry、跨项目版本差异、audit、项目 manifest 后置。

需要补强的约束：

- 安装 skill 不应自动加入某个当前激活 profile；profile 成员变更必须显式执行。
- `profile apply` 的 plan 输出应作为后续 Web、status、diff、outdated/check 复用的基础语义。
- 后期 lock 应增加 deployed files/hash，为安全卸载、drift detection 和跨项目 version drift 做准备。
- `collection` 继续只作为 source 内部分组，不升级为用户长期维护的组合概念。
- 后期 Registry 只负责发现，安装仍进入 source/index/install/profile/lock 统一链路。
