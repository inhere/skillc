# Skill Update 命令设计

## 背景

当前 `skillc` 已具备 source 管理、索引、install / uninstall / list、以及基于 lock file 的 restore 能力，但缺少独立的 `skillc update` 命令来刷新已安装 skill。

从现有实现看：

- `install` 已经会写入 lock file，记录 `SourceID`、`InstallEntry`、`InstalledPath`、`UpdatedAt` 等信息
- `install` 无参数时已支持基于 lock file 做 restore
- Git source sync 已逐步具备“先同步来源，再消费索引/缓存”的分层基础

用户确认的第一版目标是：**补齐一个以“更新已安装 skill”为核心的 `skillc update` 命令，而不是先做基于目标参数的复杂升级体系。**

## 目标

新增 CLI 命令：

- `skillc update [--agent <agent>] [--scope <scope>] [--yes]`

第一版支持：

- 默认按已安装项更新，而不是要求显式传 skill 目标
- 优先使用 lock file 中的安装记录作为更新输入
- 当 lock file 不存在或为空时，按 `--agent` / `--scope` 扫描当前已安装目录作为 fallback
- 在执行 skill 更新前，自动先同步相关来源（source sync）
- 从来源重新安装到原位置，完成“刷新到最新来源内容”的效果
- 成功更新后刷新 lock 记录，保证后续 restore / list / update 语义一致

## 非目标

第一版明确不包含：

- `skillc update <skill-id>` / collection / source 目标参数更新
- `--all`、`--dry-run`、`--json`
- 基于版本号或 checksum 的“是否需要更新”判断
- `pinned` 跳过逻辑的完整实现
- `outdated` / “可升级预览” 能力
- 交互式逐项确认
- 改变 install / restore / list 的已有外部语义

## 方案选择

### 方案 A：lock 优先，缺失时扫描已安装（推荐）

行为：

- `skillc update` 默认按 `--agent` / `--scope` 过滤已安装项
- 有 lock 记录时，直接使用 lock 记录中的来源和安装路径执行更新
- 没有 lock 记录时，扫描已安装目录，并用索引做唯一匹配后更新

优点：

- 最大化复用现有 lock 机制
- 与当前 `install -> lock -> restore` 主链路一致
- 对已有未建 lock 的安装结果仍提供回退路径
- 实现风险和歧义控制都较低

缺点：

- fallback 扫描需要明确定义“未命中 / 多命中”的跳过规则

### 方案 B：始终扫描已安装目录

优点：

- 不依赖 lock 文件

缺点：

- 容易产生来源歧义
- 浪费现有 lock 元数据
- 后续难以平滑演进到 pinned / outdated / 定向 update

### 方案 C：仅允许 lock 驱动

优点：

- 规则最清晰
- 实现最稳

缺点：

- 无法兼容当前缺少 lock 的安装场景
- 不符合用户确认的 fallback 预期

## 结论

采用 **方案 A：lock 优先，缺失时扫描已安装**。

## 设计

### 命令形态

第一版命令合同：

```bash
skillc update [--agent <agent>] [--scope <scope>] [--yes]
```

说明：

- `--agent`、`--scope` 复用现有 manage 命令选项绑定逻辑
- 不接收 `skill-id` 参数
- `--yes` 在第一版中可先保留为统一 CLI 选项风格；即使命令无确认分支，也不阻碍后续扩展

### 分层职责

#### CLI 层

在 `internal/cli/manage_cmd.go` 新增 `buildUpdateCommand()`，职责仅包括：

- 解析 `--agent` / `--scope` / `--yes`
- 读取配置
- 调用 update 应用服务
- 输出 `updated / skipped / failed` 结果
- 返回用户可理解的错误

CLI 不负责：

- 收集更新候选
- 来源同步策略
- 索引匹配逻辑
- 文件复制安装逻辑

#### 应用服务层

新增 `internal/app/updateapp/service.go` 作为独立应用服务。

它负责：

1. 根据 lock 或安装目录扫描收集待更新项
2. 按来源去重并先执行 source sync
3. 调用安装服务把 skill 重新安装到原位置
4. 汇总 `updated / skipped / failed`
5. 在 fallback 成功后补写 lock 记录

之所以不把 update 逻辑继续塞进 `installapp`，是因为 update 比 install 多了一层“确定更新候选集 + 同步来源 + 汇总跳过原因”的编排职责，已经是独立 use case。

#### install 应用服务

`internal/app/installapp/service.go` 继续负责“真正的安装动作”。

update 不自行实现复制逻辑，而是在必要时复用 install service 的安装能力。若现有 install service 只能按目标 root 安装，则允许补一个“安装到指定路径”的小入口，但不把 update 候选收集逻辑放入 install service。

#### source / index 层

- source sync 继续由 source 相关 service / infra 承担
- fallback 扫描时使用现有索引作为唯一匹配依据
- 不做模糊匹配，不按描述、名称近似值猜测来源

### 更新输入来源

#### 路径 1：lock 记录（默认优先）

若 lock file 存在且包含符合 `agent/scope` 过滤条件的记录，则直接基于 lock 记录更新。

每条记录至少提供：

- `SkillID`
- `Agent`
- `Scope`
- `SourceID`
- `InstallEntry`
- `InstalledPath`
- 可选 `QualifiedName` / `SourceQualifiedName`

这条路径最稳定，因为已有来源标识和安装落点，不需要重新猜测。

#### 路径 2：扫描已安装目录（fallback）

当 lock file 不存在或为空时：

1. 根据 `agent + scope` 解析目标安装目录
2. 扫描其直接子目录，视为候选已安装 skill
3. 用目录名去当前索引中匹配 `skill.ID`
4. 仅当命中唯一 skill 时，才纳入更新候选

规则：

- 未命中：`skipped`
- 命中多个：`skipped`
- 唯一命中：允许更新，并在成功后补写 lock

第一版 fallback 不要求推断更细的 source-qualified 语义。

### 更新执行流程

建议执行流程如下：

1. 加载配置并解析 `agent/scope`
2. 收集更新候选（lock 优先，否则扫描 fallback）
3. 按 `SourceID` 去重，同步所有涉及来源
4. 对每个候选项执行重新安装
5. 成功项刷新或补写 lock 记录
6. 输出汇总结果

### 来源同步规则

`skillc update` 在真正安装前，必须自动触发相关来源的 sync。

规则：

- 以更新候选中的 `SourceID` 去重
- 每个来源仅 sync 一次
- 某来源 sync 失败时，其对应 skill 全部记为 `failed`
- 其他来源不受影响，继续执行

这与 PRD 中“Skill Update 自动先执行 Source Update”的方向一致。

### 安装语义

第一版 update 的本质不是“比较版本后升级”，而是：

**同步来源后，将 skill 从来源重新安装到原位置。**

具体规则：

- lock 路径：覆盖安装到 `record.InstalledPath`
- fallback 路径：覆盖安装到扫描得到的现有安装目录
- 成功后更新 lock 中的 `UpdatedAt`
- 若来源元数据发生变化，可同步刷新 `Version`、`QualifiedName`、`SourceQualifiedName` 等字段

### 输出契约

定义三类结果：

#### updated

成功完成来源同步并重装。

示例：

```text
updated hello-skill /path/to/install
```

#### skipped

用于“不需要报错但无法安全更新”的场景，例如：

- 没有任何可更新项
- fallback 扫描时索引未命中
- fallback 扫描时索引多命中

示例：

```text
skipped hello-skill index match not found
skipped world-skill ambiguous index match
```

#### failed

用于真实执行失败，例如：

- 来源不存在
- source sync 失败
- 安装覆盖失败
- lock 引用的来源已失效

示例：

```text
failed demo-skill source sync failed: ...
failed hello-skill install failed: ...
```

### 跳过与继续策略

第一版 update 采用“逐项尽量继续”的策略：

- 单个来源 sync 失败，不中断其他来源
- 单个 skill 安装失败，不中断其他 skill
- 扫描歧义项跳过，不中断整体命令

只有全局前置错误（如配置解析失败、非法 scope）才直接返回命令错误。

## 数据模型建议

update 服务内部建议引入轻量候选视图，例如：

```go
type UpdateCandidate struct {
    SkillID       string
    Agent         string
    Scope         string
    SourceID      string
    InstallEntry  string
    InstalledPath string
    MatchSource   string // lock | scan
}
```

说明：

- 这是 update 编排阶段的内部 DTO，不要求暴露到 domain 层
- `MatchSource` 仅用于调试/日志/测试区分候选来源

## 错误处理

- 配置加载失败：直接返回错误
- `agent` / `scope` 非法：直接返回错误
- lock 文件不存在：不报错，进入扫描 fallback
- lock 文件为空：不报错，进入扫描 fallback
- fallback 扫描目录不存在：视为无可更新项
- source sync 失败：对应 skill 记为 failed
- install 失败：对应 skill 记为 failed

## 测试设计

### CLI 测试

在 `internal/cli/app_test.go` 增加：

- `update` 命令已注册
- lock 路径下执行 `skillc update --agent ...` 输出 `updated ...`
- lock 缺失时可走扫描 fallback
- fallback 未命中 / 多命中时输出 `skipped ...`
- 无可更新项时输出友好提示或空结果，不返回错误

### updateapp 服务测试

新增 `internal/app/updateapp/service_test.go`，覆盖：

- 从 lock 收集候选并过滤 `agent/scope`
- lock 缺失时走扫描 fallback
- fallback 唯一匹配成功
- fallback 未命中 / 多命中被跳过
- 多个 candidate 按 `SourceID` 去重 sync
- 某来源 sync 失败时，仅该来源候选失败
- 安装失败时继续其他 skill
- fallback 成功后补写 lock

### install service 复用测试

若为 update 增加了新的 install 复用入口，则在 `internal/app/installapp/service_test.go` 补最小覆盖：

- 可安装到指定已存在路径
- 成功后保留/刷新 lock 语义一致

### 回归测试

完成实现后至少运行：

```bash
go test ./...
```

因为该改动影响 MVP 主链路中的 install / lock / source sync 协作关系。

## 影响范围

预计新增 / 修改文件：

- 修改：`internal/cli/manage_cmd.go`
- 修改：`internal/cli/app.go`
- 修改：`internal/cli/app_test.go`
- 新增：`internal/app/updateapp/service.go`
- 新增：`internal/app/updateapp/service_test.go`
- 视实现需要修改：`internal/app/installapp/service.go`
- 视实现需要修改：`internal/app/installapp/service_test.go`
- 视实现需要修改：`internal/app/sourceapp/service.go`
- 修改：`mvp-arch.md`
- 修改：`mvp-plan.md`

## 与现有架构的一致性

该设计符合项目当前约束：

- CLI 层仅负责参数解析、输出格式、错误返回
- 业务编排优先下沉到 `internal/app/*` service
- install / restore 继续保持 `InstallEntry` 语义一致
- update 先复用已有 lock / source / install 能力，再考虑更复杂升级语义

## 后续可扩展点

在当前结构基础上，后续可以继续扩展：

- `skillc update <skill-id>` 定向更新
- collection / source 维度更新
- `--all`
- `--dry-run`
- `pinned` 自动跳过
- `outdated` 预览
- 版本比较 / checksum 变化检测

第一版先保持最小闭环：**能安全地更新当前已安装 skill，并与现有 lock/source/install 主链路对齐。**
