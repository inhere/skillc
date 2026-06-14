# Skillc 参考项目设计分析

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-13 | v0.1 | Codex | 分析 `skills-manager`、`skillshare`、`apm` 对 `skillc` v0 增强设计的参考价值 |

状态：Draft

相关文档：

- 主设计文档：`docs/design/skillc-v0-enhance-design.md`
- 一期开发计划：`docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md`
- 任务入口：`docs/TODO.md`

## 1. 调研范围

已拉取或缓存到 `tmp/skillc-reference-projects/`：

- `apm/`：完整 clone `https://github.com/microsoft/apm`。
- `skillshare-sparse/`：sparse clone `https://github.com/runkids/skillshare`，包含 README、命令源码、schema、website docs 等轻量路径。
- `skills-manager-sparse/`：sparse clone `https://github.com/xingkongliang/skills-manager`，包含 README、CHANGELOG、核心源码路径等轻量路径。
- 另外保留了早期下载的 `skills-manager.zip`、`skillshare.zip` 和已解出的 README 缓存。完整 zip 因仓库大资源下载超时，后续改用 sparse clone 和 raw 文档补齐。

本次重点不是评审实现质量，而是提炼概念模型、命令组织、Web 管理和后续路线中值得 `skillc` 吸收的设计。

## 2. 总体结论

当前 `skillc` v0 增强设计方向基本正确，不需要推翻。三个参考项目共同证明了几个判断：

- 用户需要的不是单次安装命令，而是“可复用、可检查、可恢复”的本地技能环境管理。
- `profile`/`preset` 这类用户维护的组合，应与上游 `collection`/目录分组严格分开。
- Web 的价值主要在关系视图和批量管理，不是替代 CLI。
- 安装和更新必须有只读检查、dry-run plan、差异查看和可审计记录。
- 后期需要 manifest/lock/registry/audit，但不要把这些一次性塞进 Phase 1。

对当前一期计划的影响：

- Phase 1 仍应保持 `profile` 最小闭环，不加入 Web、Registry、跨项目版本差异、audit、central sync。
- 需要明确一条产品规则：安装 skill 不应自动加入某个“当前激活 profile”。profile 成员变化必须显式执行。
- `profile apply` 的 dry-run/plan 是正确方向，后续 `status`、`outdated/check`、Web 都应复用同一套 plan 语义。

## 3. `skills-manager`

参考路径：

- `tmp/skillc-reference-projects/skills-manager-sparse/README.md`
- `tmp/skillc-reference-projects/skills-manager-sparse/README.zh-CN.md`
- `tmp/skillc-reference-projects/skills-manager-sparse/CHANGELOG.md`
- `tmp/skillc-reference-projects/skills-manager-sparse/CHANGELOG-zh.md`

### 3.1 值得吸收

`skills-manager` 的核心模型是：

- Central Library：统一技能库。
- Preset：命名技能组合。
- Global Workspace：按 Agent 查看全局技能目录，包含外部安装的技能。
- Project Workspace：查看和管理项目本地技能目录。
- Linked Workspace：管理任意外部 skills root。
- Agent badge：每张 skill 卡片展示它同步到了哪些 Agent。
- Update tracking：Git skill 检查远端更新，本地 skill 重新导入。
- Git backup：技能库版本化和多机同步。

对 `skillc` 最有价值的是 Web 信息架构：

- Sources/Library：技能来源和统一索引。
- Profiles：组合管理。
- Workspaces/Projects：项目和 Agent 落点。
- Install Map：某个 skill/profile 被安装到了哪些项目、哪些 Agent。
- Version Drift：同一个 skill 在不同项目里的版本/commit/checksum 差异。
- Activity/Logs：安装、更新、同步操作记录。

这与当前主设计中的 Web 定位一致：Web 主要查看和管理 CLI 不方便表达的关系，而不是成为唯一写入口。

### 3.2 对 Profile/Collection 的启发

`skills-manager` 从 `scenario` 改名到 `preset`，并在 changelog 里明确修正过一个关键问题：安装 skill 不再自动加入 active preset，因为 active preset 会漂移，容易把技能放进不该进的组合。

这直接支持 `skillc` 当前设计：

- `profile` 是用户拥有的组合，生命周期由用户控制。
- `collection` 是 source 作者提供的分组，不应直接承担项目期望状态。
- 从 collection 创建 profile 时应展开成明确 skill targets，而不是长期动态跟随 collection。
- 不要设计“当前激活 profile 自动吸收安装项”的隐式行为。

### 3.3 不建议照搬

不建议 Phase 1 照搬 `skills-manager` 的中央技能库 + 桌面应用路线。

原因：

- `skillc` 现有主链路是多 source index + install lock，已经能表达“从哪里来”和“装到了哪里”。
- `skills-manager` 是桌面应用优先，`skillc` 当前目标是 CLI 优先、Web 辅助。
- 引入 SQLite 中央库会显著扩大 v0 复杂度，也会和现有 YAML config / JSON lock 产生迁移成本。

可后期吸收：

- Web 关系视图。
- 操作日志。
- Git backup/snapshot。
- `--json` 机器输出。
- 外部 skills root 隔离状态目录，类似 `~/.skills-manager/external/<name>-<hash>/`。

## 4. `skillshare`

参考路径：

- `tmp/skillc-reference-projects/skillshare-sparse/README.md`
- `tmp/skillc-reference-projects/skillshare-sparse/website/docs/understand/source-and-targets.md`
- `tmp/skillc-reference-projects/skillshare-sparse/website/docs/understand/project-skills.md`
- `tmp/skillc-reference-projects/skillshare-sparse/website/docs/reference/commands/status.md`
- `tmp/skillc-reference-projects/skillshare-sparse/website/docs/reference/commands/diff.md`
- `tmp/skillc-reference-projects/skillshare-sparse/website/docs/reference/commands/check.md`
- `tmp/skillc-reference-projects/skillshare-sparse/website/docs/reference/commands/sync.md`
- `tmp/skillc-reference-projects/skillshare-sparse/schemas/config.schema.json`
- `tmp/skillc-reference-projects/skillshare-sparse/schemas/project-config.schema.json`

### 4.1 值得吸收

`skillshare` 与 `skillc` 技术形态最接近：Go 单二进制、本地优先、CLI + embedded Web UI。

值得吸收的点：

- `status`：聚合展示 source、tracked repositories、targets、agents、extras、audit、version。
- `diff`：在写入前展示 source 与 target 的差异，支持 plain text、TUI、JSON、stat、patch。
- `check`：只读检查更新，不修改文件。
- `sync --dry-run`：所有批量写入前先预览。
- Project mode：存在 `.skillshare/config.yaml` 时自动进入项目模式。
- Project config：项目内 manifest 可提交到 git，别人 clone 后 `install -p && sync` 重现环境。
- `.metadata.json`：记录 install source、repo URL、subdir、version/commit，用于 update/check。
- Per-target mode：merge/copy/symlink 按 target 配置。
- Per-target include/exclude：按目标过滤技能。
- SKILL.md `targets`：skill 作者可声明适配哪些目标。
- Web UI 单二进制嵌入：运行 `skillshare ui`，不要求用户额外装前端运行时。

### 4.2 对 `skillc` 的启发

`skillshare` 的 source/target 模型可转化为 `skillc` 的术语：

- `skillc source`：上游技能来源，不是安装目标。
- install target：Agent + scope + project path，是技能落点。
- profile：用户维护的一组期望安装项。
- lock：实际安装状态。

为了避免术语冲突，`skillc` 文档里建议少用裸 `target`，多写 `install target`、`agent target` 或 `profile target`。

`skillshare` 的 `status`、`diff`、`check` 三件套值得作为 `skillc` Phase 2 的核心参考：

- `skillc status`：当前项目/Agent/scope 的安装健康摘要。
- `skillc profile diff <name>`：profile 期望与当前项目实际状态差异。
- `skillc outdated` 或 `skillc update --check`：只读检查可更新项。

### 4.3 不建议照搬

不建议在 `skillc` Phase 1 引入完整 `sync` 模型。

原因：

- `skillshare` 的 install 只改 source，sync 再分发到 targets；`skillc` 当前 install 直接写到项目/全局目标。
- 若现在引入 `sync`，会把 `source` 从“上游来源”变成“本地中央库”，与现有 source/index/lock 模型冲突。
- 用户当前最急的诉求是“组合一键应用到任意项目”，`profile apply --dry-run/--yes` 足够覆盖。

可以吸收其安全性和体验原则：

- 写操作先 plan。
- 更新检查只读。
- JSON 输出可脚本化。
- copy/symlink/junction 的状态要可诊断。
- 项目模式后期用项目 manifest 解决团队共享。

## 5. `apm`

参考路径：

- `tmp/skillc-reference-projects/apm/README.md`
- `tmp/skillc-reference-projects/apm/apm.yml`
- `tmp/skillc-reference-projects/apm/apm.lock.yaml`
- `tmp/skillc-reference-projects/apm/docs/src/content/docs/reference/manifest-schema.md`
- `tmp/skillc-reference-projects/apm/docs/src/content/docs/reference/lockfile-spec.md`
- `tmp/skillc-reference-projects/apm/docs/src/content/docs/reference/cli/install.md`
- `tmp/skillc-reference-projects/apm/docs/src/content/docs/reference/cli/outdated.md`
- `tmp/skillc-reference-projects/apm/docs/src/content/docs/reference/cli/deps.md`
- `tmp/skillc-reference-projects/apm/docs/src/content/docs/reference/cli/audit.md`
- `tmp/skillc-reference-projects/apm/docs/src/content/docs/guides/operating-installed-context.md`

### 5.1 值得吸收

`apm` 是更完整的“agent dependency manager”：

- `apm.yml`：声明项目 agent dependencies。
- `apm.lock.yaml`：记录 resolved commit、ref、version、deployed files、file hashes。
- `apm install --frozen`：按 lockfile 重现，不重新解析。
- `apm outdated`：只读检查锁定依赖是否落后。
- `apm deps list/tree/why`：解释依赖树和“为什么安装了这个包”。
- `apm audit`：内容安全、lockfile consistency、drift detection、SARIF/JSON 输出。
- Registry/Marketplace：后期发现和分发层，但最终仍进入 manifest/lock。

对 `skillc` 最关键的启发是 lockfile 设计：

- lock 不只记录“装了哪个 skill”，还应记录“写了哪些文件”。
- 每个 deployed file 应可记录 hash，用于 drift 检测和安全卸载。
- uninstall/prune 只能删除 lock 里声明由工具管理的文件。
- `--frozen`/CI 模式后期可保证项目技能环境可重现。

### 5.2 对 Registry 的启发

`apm` 的 registry/marketplace 证明：Registry 不应替代 source，也不应直接成为写入逻辑。

推荐 `skillc` 后期 Registry 定位：

- Registry 负责发现：搜索 skill/source/profile。
- Registry result 在安装前解析成可缓存的 source 或 source snapshot。
- 实际 install/profile/lock 仍复用统一链路。
- lock 记录 registry provenance，避免后续卸载或更新被恶意 registry 重定向。

这与主设计里“Registry 后期实现，先保留扩展点”的决策一致。

### 5.3 不建议照搬

不建议 Phase 1 引入完整 manifest/lock/audit/policy。

原因：

- `apm` 的问题域包括多 primitive、MCP、LSP、policy、transitive deps，远大于 `skillc` 当前目标。
- `skillc` 当前最小杠杆是 profile，而不是依赖解析器。
- 过早引入强 manifest 会增加使用门槛。

但应在设计中预留：

- `skillc.profile.yaml` 或 `skillc.yaml` 项目 manifest。
- lock 中的 deployed files/hash。
- `skillc audit` 或 `skillc status --drift`。
- `skillc outdated`。
- registry provenance。

## 6. 对当前设计的校验

### 6.1 保持不变

以下当前设计可以保持：

- CLI 优先，Web 辅助。
- 新增 `profile` 作为“技能场景”主概念。
- collection 移到 `source` 下，作为上游分组视图。
- `profile create --from-collection <source>/<collection>` 展开为明确 skill targets。
- Phase 1 只做 profile 最小闭环。
- Web 放到后续阶段，重点是 source/profile/install map/version drift。
- Registry 后期实现。

### 6.2 建议补强

建议在后续设计中补强：

- 明确 `install target` 术语，避免与 `source target/profile target` 混淆。
- `profile apply` 不引入“当前激活 profile 自动吸收安装项”。
- Phase 2 增加 `outdated` 作为 `update --check` 的语义清晰入口，`update --check` 可作为别名或兼容 flag。
- `status` 输出至少包含 installed、missing、outdated、orphan、unmanaged、source-error。
- 后期 lock 增加 deployed files/hash，为安全 uninstall、drift detection、跨项目 version drift 做准备。
- Web 写操作全部 plan-first，与 CLI plan 共享 service。
- 配置文件加 JSON Schema/YAML language server 注释，降低手写 profile/manifest 成本。

## 7. 分阶段吸收建议

### Phase 1：Profile 最小闭环

只吸收：

- profile/preset 与 collection/source group 的边界。
- profile apply dry-run plan。
- 不自动把 install 加入 active profile。
- source-scoped collection 浏览。

不吸收：

- central library sync。
- Web。
- registry。
- audit。
- project manifest。
- cross-project drift。

### Phase 2：Status / Outdated / Diff

吸收 `skillshare` 和 `apm` 的只读检查模型：

- `skillc status`
- `skillc profile diff <name>`
- `skillc outdated` 或 `skillc update --check`
- JSON 输出
- local source checksum / git source commit 比较

### Phase 3：Project Manifest

参考 `skillshare` project config 和 `apm.yml`：

- 支持项目内 `skillc.profile.yaml` 或 `skillc.yaml`。
- 可提交到 git。
- 新成员 clone 项目后执行 `skillc profile apply` 或 `skillc install --frozen` 复现。
- 用户个人全局 profiles 与项目 manifest 分离。

### Phase 4：Web 管理

参考 `skills-manager` 和 `skillshare ui`：

- 使用 Go 单二进制嵌入静态 Web UI。
- Dashboard 展示当前项目、Agent、profile、健康摘要。
- Sources 页面展示 source、collections、skills。
- Profiles 页面编辑组合并 apply/diff。
- Install Map 页面展示 skill/profile 到 projects/agents/scopes 的关系。
- Version Drift 页面批量检查和更新。
- Logs 页面展示操作记录。

### Phase 5：Registry / Audit / Policy

参考 `apm`：

- Registry discovery。
- lock provenance。
- deployed file hashes。
- drift detection。
- hidden Unicode / prompt injection 基础 audit。
- 企业策略或 allowlist 放到更后期。

## 8. 一期计划是否需要调整

结论：不需要扩大范围。

建议只在 Phase 1 文档中补充三条约束：

- 安装 skill 不自动修改任何 profile。
- `profile apply` 的 plan 输出是后续 Web/diff/status 复用的基础语义。
- 本期不引入 project manifest，但 profile export/import 要避免把后期路径封死。

这样一期仍然能快速验证核心抽象，同时不会被 Web、Registry、audit、manifest 拖住。
