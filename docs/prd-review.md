# Skillc PRD 审核报告

> 审核日期：2026-03-29
> 审核文件：prd.md (Draft v1)
> 状态：待修复

---

## 问题分级说明

- 🔴 **高**：阻塞设计或开发，必须进入开发前确认
- 🟡 **中**：影响功能完整性和用户体验，建议 M1/M2 前解决
- 🟢 **低**：细节完善，可后续迭代

---

## 一、逻辑不一致或矛盾

| ID | 级别 | 章节 | 问题描述 | 状态 |
|----|------|------|----------|------|
| L1 | 🔴 高 | §6 vs §9.4 | Registry 与 Source 安装链路未厘清。Registry 搜索到的 Skill，安装时是映射回 Git 源还是独立包？Lock 记录的 `source_id` 对 Registry 来源的 Skill 应填什么？ | ❓ 待确认 |
| L2 | 🔴 高 | §6 vs §12 | Source 模型的 `type` 枚举不完整。§6 定义三种来源（local/git/registry），但 §9.3 来源管理仅支持 local 和 git，Registry 单独管理——Source `type` 是否含 registry？ | ❓ 待确认 |
| L3 | 🔴 高 | §9.6 vs §18 | 安装冲突策略是 M2 的阻塞决策，却留在"待确认"中。§9.6 引用"冲突处理"，但策略（覆盖/跳过/阻止）未定义。 | ❓ 待确认 |
| L4 | 🟡 中 | §9.8 vs §10 | `skillc update` 与 `skillc source sync` 的关系歧义。§9.8 区分了 Source Update 和 Skill Update 两阶段，但未说明 `skillc update` 是否自动先执行 source sync。 | 🔧 可修复 |
| L5 | 🟡 中 | §2 vs §9.2 | codex 的 Skill 目录规范未定义。draft 配置中 codex 的 dirname 写的是 `.claude`，与 claude-code 相同，疑似复制错误。 | 🔧 可修复 |
| L6 | 🟢 低 | §1 | 章节编号跳跃（§2 后直接跳到 §6），中间章节（§3~§5、§7~§8）缺失，暗示内容被裁剪未标注。 | 🔧 可修复 |

---

## 二、功能遗漏

| ID | 级别 | 参照标准 | 遗漏描述 | 状态 |
|----|------|----------|----------|------|
| F1 | 🔴 高 | npm/pip | 无从锁文件恢复命令（类似 `npm install` 无参数）。Lock File 的核心价值之一是一键恢复全部已安装 Skill，PRD 未提及。 | 🔧 可修复 |
| F2 | 🔴 高 | npm/pip/brew | 无版本约束安装语法。`skillc install <skill-id>` 未支持指定版本（如 `@1.2.0`），Lock 记录有 `version` 字段但安装时如何写入未说明。 | ❓ 待确认 |
| F3 | 🔴 高 | 全文 | Skill `version` 字段语义完全未定义。SemVer？Git tag？Git commit SHA？本地路径源如何确定版本？这是安装/更新/锁文件逻辑的基石。 | ❓ 待确认 |
| F4 | 🔴 高 | §6 vs §12 | `entry`（§6）与 `install_entry`（§12）命名不一致，且语义不清。指向单文件、目录还是一组文件？不同 Agent 的 entry 格式是否不同？ | 🔧 可修复 |
| F5 | 🟡 中 | brew/npm | 缺少 `skillc registry add/remove/list` 子命令组。Registry 只能手动编辑 YAML，与 `skillc source` 的 CLI 体验严重不一致。 | 🔧 可修复 |
| F6 | 🟡 中 | brew/npm | 缺少 `skillc config set/get` 子命令。用户无法通过 CLI 修改配置，只能手动编辑 YAML，是 CLI 工具的反模式。 | 🔧 可修复 |
| F7 | 🟡 中 | brew/npm | 缺少 `skillc doctor` 命令：诊断环境健康（agent 目录可写、Git 可用、配置文件合法、锁文件一致性等）。 | 🔧 可修复 |
| F8 | 🟡 中 | npm/pip | 缺少 `--all` 批量操作能力。`skillc update` 和 `skillc uninstall` 无法操作某 agent 下的全部 Skill。 | 🔧 可修复 |
| F9 | 🟡 中 | pip/npm | 缺少 `skillc validate` 命令：Skill 作者发布前校验包结构（元数据完整、entry 存在、supported_agents 合法等）。 | 🔧 可修复 |
| F10 | 🟢 低 | pip/brew | 缺少 `skillc export` / `skillc import`：导出/导入已安装清单，便于环境迁移。Lock File 部分承担此职责，但未显式说明。 | 🔧 可修复 |
| F11 | 🟢 低 | 全文 | 缺少开发者软链接模式（`skillc link`）：本地开发 Skill 时需软链接到 Agent 目录进行实时调试。§18 将其列为"待确认"。 | ❓ 待确认 |

---

## 三、CLI 设计问题

| ID | 级别 | 章节 | 问题描述 | 状态 |
|----|------|------|----------|------|
| C1 | 🔴 高 | §10 | `--agent` 和 `--scope` 无缺省行为定义。`skillc install foo` 不带参数时报错？使用默认 agent？安装到全部 agent？这是最常用命令的核心体验。 | ❓ 待确认 |
| C2 | 🟡 中 | §10 | `skillc init` 语义不清。初始化全局配置？项目级配置？应说明在全局首次使用 vs 项目级首次使用时的不同行为。 | 🔧 可修复 |
| C3 | 🟡 中 | §10 | `skillc source add` 缺少 `--name` / `--id` 参数。Source 模型有 `name` 和 `id` 字段，但 CLI 只接受路径或 URL，name/id 从哪来？ | 🔧 可修复 |
| C4 | 🟡 中 | §10 | 缺少 `--yes/-y` 确认跳过标志。安装/删除/更新等变更操作是否需要交互确认？CI/CD 场景需要支持非交互模式。 | 🔧 可修复 |
| C5 | 🟡 中 | §10 | 缺少 `--dry-run` 支持。install/uninstall/update 应支持预览模式，显示将要执行的操作但不实际执行。 | 🔧 可修复 |
| C6 | 🟡 中 | §10 | 缺少结构化输出支持。`skillc list`、`skillc search` 未提供 `--json` / `--format` 标志，不利于脚本化和管道使用。 | 🔧 可修复 |
| C7 | 🟢 低 | §10 | 缺少 `skillc version` 和 `skillc help` 命令。标准 CLI 工具应有版本查询和帮助系统。 | 🔧 可修复 |
| C8 | 🟢 低 | §10 | 缺少 Shell 补全支持说明（bash/zsh/fish/powershell）。Go CLI 工具（cobra 等）通常支持补全命令生成。 | 🔧 可修复 |
| C9 | 🟢 低 | §10 | `skillc show` 展示内容未说明。展示远程元数据？本地安装状态？版本历史？来源信息？ | 🔧 可修复 |

---

## 四、数据模型问题

| ID | 级别 | 章节 | 问题描述 | 状态 |
|----|------|------|----------|------|
| D1 | 🔴 高 | §12 | Skill `version` 语义未定义（见 F3）。 | ❓ 待确认 |
| D2 | 🔴 高 | §6 vs §12 | `entry` / `install_entry` 命名不统一（见 F4）。 | 🔧 可修复 |
| D3 | 🟡 中 | §12 | Lock 记录缺少 `source_type` 字段。仅有 `source_id`，若 source 被删除则无法判断来源类型，影响 orphan 检测和 restore 逻辑。 | 🔧 可修复 |
| D4 | 🟡 中 | §12 | Git Source 模型缺少 `branch` / `tag` / `ref` 字段。只有 `url`，无法指定跟踪特定分支或 tag。 | 🔧 可修复 |
| D5 | 🟡 中 | §12 | Lock 记录缺少 `integrity` / `checksum` 字段。无法验证安装后文件完整性，无法检测篡改或损坏。 | 🔧 可修复 |
| D6 | 🟡 中 | §12 | Skill 元数据缺少 `author` / `maintainer` 字段。 | 🔧 可修复 |
| D7 | 🟡 中 | §12 | Skill 元数据缺少 `license` 字段。开源生态标准元数据。 | 🔧 可修复 |
| D8 | 🟡 中 | §11 vs §12 | Lock File 存储格式未明确。配置用 YAML，锁文件用什么？JSON？YAML？TOML？draft 注释写的是 json，但 PRD 正文未确认。 | 🔧 可修复 |
| D9 | 🟢 低 | §12 | Lock 记录缺少 `pinned` 标志。无法阻止特定 Skill 被 `update` 命令升级。 | 🔧 可修复 |
| D10 | 🟢 低 | §12 | `installed_path` 存储绝对路径在跨平台共享项目时会失效。应考虑相对路径或平台无关的表示方式。 | 🔧 可修复 |

---

## 五、错误场景遗漏

| ID | 级别 | 章节 | 遗漏场景 | 状态 |
|----|------|------|----------|------|
| E1 | 🔴 高 | §14 | 并发操作冲突：多个终端同时运行 `skillc install` / `skillc update` 时对锁文件和缓存的并发写入保护。 | 🔧 可修复 |
| E2 | 🔴 高 | §14 | 锁文件损坏或格式不合法：用户手动编辑导致解析失败时的恢复策略。 | 🔧 可修复 |
| E3 | 🟡 中 | §14 | 磁盘空间不足：缓存和安装过程中磁盘满的处理（尤其是大型 Git 仓库克隆）。 | 🔧 可修复 |
| E4 | 🟡 中 | §14 | 网络中断（部分完成）：Git clone 或 Registry 下载中途断开，残留文件的清理策略。 | 🔧 可修复 |
| E5 | 🟡 中 | §14 | Skill 安装目标路径已存在非 Skillc 管理的同名文件/目录（不在锁文件中但物理路径冲突）。 | 🔧 可修复 |
| E6 | 🟡 中 | §14 | Agent 自身升级后目录结构变更，已安装 Skill 变成 orphan。 | 🔧 可修复 |
| E7 | 🟡 中 | §14 | Skillc 自身升级后旧版配置/锁文件格式的迁移策略。 | 🔧 可修复 |
| E8 | 🟢 低 | §14 | 符号链接循环：本地路径源指向的目录包含循环符号链接。 | 🔧 可修复 |
| E9 | 🟢 低 | §14 | Git submodule 场景：Skill 仓库使用 Git submodule 时的处理。 | 🔧 可修复 |

---

## 六、里程碑与验收标准问题

| ID | 级别 | 章节 | 问题描述 | 状态 |
|----|------|------|----------|------|
| M1 | 🔴 高 | §16 vs §17 | 验收标准未按里程碑分拆。§17 的 9 条验收标准混合覆盖 M1~M3，无法作为各阶段交付依据。 | 🔧 可修复 |
| M2 | 🟡 中 | §17 | 性能指标不可量化。"1000 个 Skill 规模下可接受响应速度"缺具体数值（如 P95 < Xms）。 | 🔧 可修复 |
| M3 | 🟡 中 | §16 | M1 未明确 `skillc init` 的交付。init 是用户首次使用的入口，应在 M1 明确。 | 🔧 可修复 |
| M4 | 🟢 低 | §16 | 里程碑无时间估算。M1/M2/M3 无工时预估，无法排期。 | 🔧 可修复 |

---

## 七、跨平台问题

| ID | 级别 | 章节 | 问题描述 | 状态 |
|----|------|------|----------|------|
| P1 | 🔴 高 | §11 | 默认配置路径未按平台定义。Linux（XDG `~/.config/skillc`）、macOS（`~/Library/Application Support/skillc`）、Windows（`%APPDATA%\skillc`）规范不同，PRD 未明确采用哪种策略。 | 🔧 可修复 |
| P2 | 🟡 中 | §9.2 | Agent 目录路径的 Windows 差异。各 Agent 在 Windows 上的目录（如 `%USERPROFILE%\.claude\`）未定义。 | 🔧 可修复 |
| P3 | 🟡 中 | 全文 | Git 可用性假设。PRD 假设系统已安装 Git，但未说明 Git 不存在时的行为（降级？报错指引安装？）。 | 🔧 可修复 |
| P4 | 🟢 低 | §9.3 | Windows 长路径问题。Windows 默认 260 字符路径限制可能影响深层嵌套的 Skill 仓库。 | 🔧 可修复 |

---

## 八、其他问题

| ID | 级别 | 章节 | 问题描述 | 状态 |
|----|------|------|----------|------|
| O1 | 🔴 高 | §18 | Skill 仓库标准目录结构和 `skill.yaml` 规范是开发前必须确认的阻塞项，直接决定索引和安装实现。 | ❓ 待确认 |
| O2 | 🟡 中 | §15 | 安全模型中"默认不执行 Skill 内脚本"描述模糊。Skill 本身是供 Agent 执行的指令，"脚本"具体指安装 hook 脚本？post-install 脚本？需区分 Skill 内容与安装脚本。 | 🔧 可修复 |
| O3 | 🟡 中 | §9.4 | Registry 协议规范未定义。第三方平台如何接入？REST API？静态 JSON 索引？接入规范直接影响 Registry 适配器设计。 | ❓ 待确认 |
| O4 | 🟡 中 | 全文 | 缺少 Skill 生命周期状态机。发现→缓存→安装→更新→删除的状态流转未图示，各环节触发条件和数据变化不清晰。 | 🔧 可修复 |
| O5 | 🟢 低 | 全文 | 缺少日志设计细节。§13 提到"debug 日志开关"，但日志格式、日志文件位置、日志轮转策略均未定义。 | 🔧 可修复 |
| O6 | 🟢 低 | 全文 | 缺少国际化考量。Skill 名称/描述是否支持 Unicode？搜索时是否支持 CJK 模糊匹配？ | 🔧 可修复 |

---

## 关键阻塞项汇总（进入开发前必须确认）

以下 7 个问题必须在设计/开发启动前明确，否则核心流程无法落地：

| # | 问题 | 关联 ID |
|---|------|---------|
| 1 | Skill 仓库标准目录结构和 `skill.yaml` 规范是否强制要求 | O1 |
| 2 | `version` 字段语义（按 source type 分别定义） | F3/D1 |
| 3 | Registry vs Source 的关系及安装链路（Lock 记录如何处理 Registry 来源） | L1/L2 |
| 4 | `--agent` / `--scope` 缺省行为 | C1 |
| 5 | 安装冲突策略（覆盖/跳过/阻止） | L3 |
| 6 | 版本约束安装语法（`install foo@1.0.0` 是否支持） | F2 |
| 7 | Registry 接入协议规范（REST API / 静态索引 / 其他） | O3 |

---

## 修复进度追踪

- [x] 审核报告写入
- [x] 修复可确定问题（标记为"🔧 可修复"的条目）
- [x] 向用户确认阻塞项（标记为"❓ 待确认"的条目）
- [x] 根据用户确认结果更新 PRD

## 已确认决策（2026-03-30）

| # | 问题 | 决策 |
|---|------|------|
| 1 | Skill 元数据格式 | 从 `SKILL.md` YAML front matter 读取，无需 `skill.yaml` |
| 2 | version 语义 | 从 SKILL.md 读取；回退：git→commit SHA，local→内容指纹，registry→返回字段 |
| 3 | Registry 接入协议 | 优先 JSON 索引，无则 HTTP API |
| 4 | 安装冲突默认策略 | 提示用户选择（覆盖/跳过/取消）；`--yes` 时默认覆盖 |
| 5 | `--agent/--scope` 缺省 | 不带 `--agent` 装到所有已配置 Agent；不带 `--scope` 默认 global |
| 6 | 版本约束安装 | 支持 `install foo@1.0.0`，默认 latest |
| 7 | Agent 目录推断 | 默认 `.{name}`；全局目录所有平台统一用 `~/.{name}/` |
| 8 | 软链接开发模式 | 后续版本支持，首期不实现 |
