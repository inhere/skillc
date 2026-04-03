# Install 批量目标与 collection 安装设计

## 背景

当前 `skillc install` 已支持基于单个 `skill-id` 解析目标并执行安装，且 install service 已具备对 `[]skill.Skill` 做批量安装的能力。但 CLI 入口仍偏单目标语义，缺少以下能力：

- 一次安装多个目标
- 显式按 collection 安装
- 基于 `skill.ID` 前缀的批量匹配
- 安装前统一确认与 `-y/--yes` 跳过确认
- 解析失败或单项安装失败时的“部分成功，继续执行”语义

项目当前分层要求明确：CLI 层尽量只做参数解析、输出格式、错误返回；业务编排优先下沉到 `internal/app/*` service。因此本设计的核心是：在不破坏现有 install 主链路的前提下，为 install 命令补上批量目标解析、确认和部分成功语义。

## 目标

本次设计目标：

1. 为 `skillc install` 增加 `-c, --collection` 选项
2. 为 `skillc install` 增加 `-y, --yes` 选项
3. 保持 `skill-id` 作为位置参数，但支持逗号分隔多个目标
4. 支持 `prefix-*` 语法，并且仅按 `skill.ID` 做前缀匹配
5. 支持一次命令中混用多个普通 skill 目标、多个 prefix 目标，以及显式 collection 目标
6. 解析失败时提示并跳过，不中断整个批次
7. 安装单项失败时提示并跳过，继续安装后续项
8. lock file 仅记录实际安装成功的项

## 非目标

本次不包含：

- 变更 `search` / `show` 的已有解析语义
- 引入新的顶级 install 子命令
- 支持更复杂的 glob/正则匹配（仅支持 `prefix-*`）
- 改变 restore 语义
- 改变 uninstall 语义
- 引入交互式逐项确认（本次仅统一确认一次）

## 已确认的交互语义

### 1. `-c, --collection`

`-c` 为**强制 collection 安装模式**：

- 当传入 `-c` 时，每个输入目标都必须按 collection 目标解释
- 解析结果为该 collection 下全部 skills
- 本次不做“优先 collection，找不到再退回 skill”的模糊语义

### 2. 多目标混用

一次 install 命令允许混用不同目标形式，例如：

- `a,b,c`
- `a,foo-*`
- `repo-tools,hello-*`（带 `-c` 时 collection 语义生效）

目标之间彼此独立解析，最终汇总成一个待安装 skill 列表。

### 3. `-y, --yes`

`-y` 跳过全部确认，包括：

- 单个 skill 安装
- 多 skill 批量安装
- collection 展开后的批量安装
- prefix 展开后的批量安装

### 4. `prefix-*` 匹配范围

`prefix-*` 仅按 `skill.ID` 匹配，不匹配：

- `QualifiedName`
- `SourceQualifiedName`
- 其他别名或显示名

例如：

- `hello-*` 只匹配 `skill.ID` 以 `hello-` 开头的技能
- 即使某个 `QualifiedName` 以该前缀开头，只要 `skill.ID` 不匹配，也不应命中

### 5. 部分成功语义

对于一批输入目标：

- 某个目标解析失败：提示错误并跳过
- 某个已解析出的 skill 安装失败：提示错误并跳过
- 其他项继续执行
- 最终命令返回汇总结果，而不是遇错即停

### 6. 未传 `-c` 时的 collection 边界

当**未传 `-c`** 时：

- install 优先按 skill 目标解析
- `prefix-*` 仅按前缀规则展开 skill
- 即使输入看起来像 collection，也不主动按 collection 语义展开

也就是说，collection 展开必须显式要求，不能隐式触发。

## 设计方案

采用：**CLI 做输入拆分与确认，app service 做目标解析，install service 做安装执行与结果汇总**。

这是一个介于“最小补丁”和“全量重构”之间的方案：

- 不把 collection / prefix 规则直接塞进 CLI
- 不重写 install 主链路
- 保持当前 `searchapp` 负责索引读取与目标解析，`installapp` 负责安装执行

## 分层职责

### CLI 层：`internal/cli/manage_cmd.go`

职责：

- 给 `install` 命令增加 `-c` / `-y`
- 读取原始 `skill-id` 参数
- 做逗号拆分、去空白、去除空 token
- 调用应用服务解析批量目标
- 在安装前输出统一确认信息
- 在安装后输出成功项 / 失败项汇总

CLI 不负责：

- collection 聚合规则
- prefix 匹配规则
- 实际安装复制逻辑
- lock 记录写入逻辑

### 应用服务层：`internal/app/searchapp/service.go`

新增 install 目标解析入口，职责：

- 将原始目标列表解析为结构化结果
- 按 `-c` / 非 `-c` 模式应用不同解析规则
- 支持 `prefix-*` 按 `skill.ID` 展开
- 收集失败目标与失败原因
- 对最终待安装 skills 做去重与稳定排序（如需要）

建议新增一个面向 install 的返回结构，例如：

```go
type InstallTargetResolveResult struct {
    Resolved []skill.Skill
    Failed   []TargetError
}

type TargetError struct {
    Target string
    Reason string
}
```

这样 CLI 可直接消费结构化结果，而不是依赖字符串拼接。

### 安装服务层：`internal/app/installapp/service.go`

保留 install service 的核心职责：

- 接收已解析的 `[]skill.Skill`
- 逐项执行安装
- 记录成功项与失败项
- 仅对成功项写 lock file

现有 `InstallMulti()` 是“任一失败即返回”的全-or-无语义，本次需要调整为批量结果聚合语义。建议新增结构化执行结果，例如：

```go
type BatchInstallResult struct {
    Installed []lockpkg.Record
    Failed    []InstallItemError
}

type InstallItemError struct {
    SkillID string
    Reason  string
}
```

这样可明确表达“部分成功”。

## 输入解析规则

### 1. 原始参数拆分

`skill-id` 保持为单个位置参数，但其内容支持逗号分隔：

- 输入：`a,b,c`
- 解析为：`["a", "b", "c"]`

拆分规则：

1. 以英文逗号 `,` 分隔
2. 对每个 token 做 `strings.TrimSpace`
3. 空 token 丢弃
4. 若最终为空，仍视为“未提供 skill-id”，保持现有 restore 语义

### 2. 非 `-c` 模式

当未传 `-c` 时，对每个 token 按如下规则处理：

#### 普通目标

普通目标按 skill 语义解析：

- 优先匹配单个 skill
- 不主动展开 collection
- 沿用现有 install/search 侧的 skill 目标解析能力

#### prefix 目标

当 token 形如 `prefix-*` 时：

- 将 `prefix-` 作为前缀
- 仅遍历索引中的 `skill.ID`
- 命中所有 `strings.HasPrefix(skill.ID, prefix)` 的 skill
- 若无匹配，则记录失败项

#### collection 形态输入

即使 token 的文本形态看起来像 collection 名或 collection 路径：

- 只按普通 skill 目标语义处理
- 不自动按 collection 展开

这是为了保证 collection 安装必须显式使用 `-c`。

### 3. `-c` 模式

当传入 `-c` 时，对每个 token 按 collection 目标解析：

- token 必须解释为 collection
- 解析结果为该 collection 下全部 skills
- 若 collection 不存在，则记录失败项
- 不退回到 skill 单项解析

### 4. 去重规则

同一个 skill 可能通过多种输入重复命中，例如：

- `hello-skill,hello-*`
- `repo-tools,hello-*`（在 `-c` 模式下 collection 展开与 prefix 结果重叠）

最终待安装列表应去重，避免重复安装。同一 skill 的去重键应与 install 当前主标识保持一致，优先使用稳定的唯一身份字段，例如：

- `SourceQualifiedName`（若已保证唯一）
- 或 `(SourceID, QualifiedName)` 组合

去重后保留一份待安装记录即可。

## 确认与输出设计

### 1. 默认统一确认

当存在可安装项时，在真正安装前统一确认一次。

确认信息至少应包含：

- 原始输入目标数
- 成功解析出的 skill 数
- 失败目标数
- 是否包含 collection 展开
- 是否包含 prefix 展开

示例文案方向：

```text
about to install 7 skills from 4 targets (1 failed, collection expansion enabled)
continue? [y/N]
```

本次设计只要求“统一确认”，不要求逐项确认。

### 2. `-y` 跳过确认

当传入 `-y` 时：

- 不做任何确认提示
- 直接进入安装执行阶段

### 3. 失败项输出

对于解析失败项与安装失败项，CLI 应分别输出，但都不应立即终止整个批次。

建议输出分组：

- `resolve skipped:`
- `install failed:`
- `installed:`

这样可以让用户快速区分失败发生在“解析阶段”还是“执行阶段”。

## 执行语义

### 1. 解析阶段

- 逐项解析输入目标
- 失败项收集到 `Failed`
- 成功项汇总到 `Resolved`
- 若最终 `Resolved` 为空，则命令结束并返回“no installable skills found”类错误或友好提示

### 2. 安装阶段

- 按去重后的 `Resolved` 列表逐项安装
- 某个 skill 安装失败时，收集错误并继续后续项
- 成功项写入 lock file
- 失败项不写 lock file

### 3. 最终结果

命令结束时返回汇总结果：

- 成功安装项
- 解析跳过项
- 安装失败项

退出码策略可保持当前简单语义：

- 若至少成功安装一个 skill，可视为命令整体成功
- 若没有任何成功项，则返回错误

具体错误码实现保持与现有 CLI 错误映射一致。

## 数据结构建议

### CLI 请求对象

`InstallReq` 可扩展为更贴近命令语义的字段，例如：

```go
type InstallReq struct {
    SkillID     string
    Agent       string
    Scope       string
    WorkDir     string
    Collection  bool
    Yes         bool
}
```

其中：

- `SkillID` 仍承接原始位置参数
- `Collection` 表示启用 collection 解析模式
- `Yes` 表示跳过确认

### 解析结果对象

```go
type InstallTargetResolveResult struct {
    Resolved []skill.Skill
    Failed   []TargetError
}

type TargetError struct {
    Target string
    Reason string
}
```

### 执行结果对象

```go
type BatchInstallResult struct {
    Installed []lockpkg.Record
    Failed    []InstallItemError
}

type InstallItemError struct {
    SkillID string
    Reason  string
}
```

如果需要兼容现有 `CommandResult`，也可以在不大改对外接口的前提下扩展字段，而不是直接替换。

## 测试设计

### `internal/app/searchapp/service_test.go`

覆盖：

1. 逗号分隔目标可独立解析
2. 普通 skill 与 prefix 目标可混用
3. `prefix-*` 仅匹配 `skill.ID`
4. prefix 无匹配时返回失败项但不中断其他目标
5. `-c` 模式下按 collection 展开全部 skills
6. 非 `-c` 模式下 collection 形态输入不自动展开
7. 重复命中项会去重

### `internal/app/installapp/service_test.go`

覆盖：

1. 多个已解析 skill 可逐项安装
2. 单项安装失败时继续后续项
3. lock file 仅包含成功安装项
4. 最终结果中能区分已安装项与失败项
5. 无任何成功项时返回合理错误

### `internal/cli/app_test.go`

覆盖：

1. `install` 命令注册了 `-c` / `-y`
2. `install a,b` 能拆分多目标
3. `install -c toolbox` 触发 collection 解析路径
4. `install hello-*` 触发 prefix 解析路径
5. `-y` 时跳过确认
6. 存在解析失败项时命令继续执行并输出失败摘要
7. 存在安装失败项时命令继续执行并输出成功/失败汇总

### 端到端 / 回归测试

若现有 e2e 已覆盖 install 主链路，建议补一个最小回归场景：

- 多目标输入中一部分成功、一部分失败
- `-c` collection 安装展开多个 skills
- prefix 批量安装只匹配 `skill.ID`

## 影响范围

预计涉及文件：

- 修改：`internal/cli/manage_cmd.go`
- 修改：`internal/cli/app_test.go`
- 修改：`internal/app/searchapp/service.go`
- 修改：`internal/app/searchapp/service_test.go`
- 修改：`internal/app/installapp/service.go`
- 修改：`internal/app/installapp/service_test.go`
- 可能修改：`arch.md`
- 可能修改：`plan.md`

其中 `arch.md` / `plan.md` 需要在实现落地时同步更新 install 的批量解析、确认与部分成功语义，避免文档漂移。

## 与现有架构的一致性

本设计保持以下边界：

- CLI 层只做参数解析、交互确认、输出汇总
- install 目标解析规则下沉到 app service，而不是散落在 CLI
- install service 继续负责安装执行与 lock 写入
- restore 仍保持“未提供 `skill-id` 时从 lock file 恢复”的既有语义
- 不改变 `InstallEntry` 与 `SourceID + InstallEntry` 的既有恢复约束

## 后续扩展点

未来可在当前结构上继续扩展：

- `--dry-run`：仅展示解析与安装计划，不实际执行
- `--json`：输出结构化批量结果
- 批量确认时展示更详细的展开清单
- uninstall 对齐相同的多目标 / prefix / confirm 语义

当前版本先保持最小闭环，只实现本次已确认的 install 批量目标能力。
