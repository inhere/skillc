# Collection 浏览命令设计

## 背景

当前项目已经具备 source 管理、skill 索引、search/show 等能力，且 `skill.Skill` 模型中已经存在 `Collection` 字段。用户希望补充一个独立的集合浏览命令，用于查看已有集合列表，以及查看某个集合下的 skill 列表。

本设计明确：**集合的定义以 skill 元数据中的 `collection` 字段为准**，而不是目录层级或 source 名称推导。

## 目标

新增独立 CLI 命令：

- `skillc collection list`
- `skillc collection skills <collection>`

支持：

- 列出所有已有集合
- 展示每个集合下的 skill 数量
- 展示每个集合涉及的 source 数量
- 查看指定集合下的 skill 列表

## 非目标

本次不包含：

- `collection show <name>` 详情页
- collection 级安装/卸载语义变更
- 目录结构推导 collection
- registry 维度额外聚合视图
- `search` 命令行为调整

## 设计方案

### 方案选择

采用：**独立 `collection` 顶级命令 + 基于索引聚合实现**。

具体命令形态：

- `skillc collection list`
- `skillc collection skills <collection>`

之所以不复用 `search` 命令，是为了避免把“关键字搜索”和“集合浏览”混在一起。集合浏览本质上是对索引结果的聚合视图，应有独立入口。

### 分层职责

#### CLI 层

新增 `internal/cli/collection_cmd.go`，职责仅包括：

- 注册 `collection` 顶级命令
- 解析 `list` 和 `skills <collection>` 子命令参数
- 调用应用服务
- 用表格输出结果
- 返回用户可理解的错误

CLI 不承担聚合逻辑。

#### 应用服务层

在 `internal/app/searchapp/service.go` 增加 collection 相关入口，继续复用现有 index store 读取能力：

- `ListCollections() ([]repoindex.CollectionSummary, error)`
- `ListCollectionSkills(collection string) ([]skill.Skill, error)`

这样可以保持 “索引读取在 searchapp，聚合规则在 repoindex” 的现有分层风格。

#### 基础设施层

新增 `internal/infra/repoindex/collection.go`，承载纯聚合逻辑：

- 从 `[]skill.Skill` 生成 collection 摘要列表
- 根据 collection 名过滤 skill 列表
- 负责排序与计数规则

这样避免把 collection 聚合逻辑继续堆进 `search.go`，保持文件职责清晰。

## 数据结构

新增轻量聚合视图模型：

```go
type CollectionSummary struct {
    Name        string
    SkillCount  int
    SourceCount int
}
```

说明：

- `Name`：collection 名称
- `SkillCount`：属于该 collection 的 skill 数量
- `SourceCount`：包含该 collection 的去重 source 数量

`collection skills <collection>` 不需要新的 DTO，直接返回 `[]skill.Skill` 即可，CLI 只展示：

- `Name`
- `Description`

## 规则定义

### `collection list`

聚合规则：

1. 仅统计 `skill.Collection != ""` 的 skill
2. 按 `skill.Collection` 分组
3. 同名 collection 跨 source 合并为一条记录
4. `SkillCount` 为该组内 skill 总数
5. `SourceCount` 为该组内去重后的 source 数量
6. 输出按 collection 名字升序排列，保证结果稳定

表格列为：

- `Collection`
- `Skills`
- `Sources`

### `collection skills <collection>`

过滤规则：

1. 仅返回 `skill.Collection == <collection>` 的 skill
2. 结果按 `skill.Name` 升序排列，保证输出稳定
3. 默认展示字段：`Name | Description`
4. 不额外展示 source 信息，保持输出简洁

### 空值规则

- 没有 `collection` 字段的 skill 不进入集合列表
- 没有 `collection` 字段的 skill 也不会出现在 `collection skills <collection>` 结果中

### 歧义规则

本设计中，collection 视图按 collection 名直接聚合，不引入 `source/collection` 级子命令。

也就是说：

- `collection list` 展示的是全局 collection 汇总视图
- `collection skills foo` 展示的是所有 source 中 `collection=foo` 的 skill

这与当前安装侧对 collection 歧义的处理不同，但符合“浏览视图”的预期：浏览允许跨 source 汇总，安装仍可维持严格歧义消解。

## 错误处理

- 索引文件不存在：返回空列表，不报错
- `collection list` 无数据：CLI 输出空结果或友好提示
- `collection skills <collection>` 找不到集合：返回 `collection not found: <name>`
- 参数缺失：返回 `collection name is required`

## 测试设计

### repoindex 聚合测试

新增测试覆盖：

- 同一 collection 在单个 source 中的聚合
- 同一 collection 跨多个 source 的聚合
- 空 collection skill 被过滤
- `SourceCount` 去重正确
- `collection skills` 结果按名称排序
- collection 不存在时报错

### searchapp 服务测试

新增测试覆盖：

- 索引存在时可列出 collection 摘要
- 索引不存在时返回空结果
- 可列出指定 collection 的 skill
- 指定 collection 不存在时返回错误

### CLI 测试

新增测试覆盖：

- `collection` 命令已注册
- `collection list` 输出包含集合名、skill 数量、source 数量
- `collection skills demo` 输出包含 skill 名称与描述
- 集合不存在时返回错误

## 影响范围

预计变更文件：

- 新增：`internal/cli/collection_cmd.go`
- 修改：`internal/cli/app.go`
- 修改：`internal/cli/app_test.go`
- 新增：`internal/infra/repoindex/collection.go`
- 新增或修改：`internal/infra/repoindex/collection_test.go`
- 修改：`internal/app/searchapp/service.go`
- 修改：`internal/app/searchapp/service_test.go`

## 与现有架构的一致性

该设计符合当前项目约束：

- CLI 层只做参数解析与输出
- 业务入口继续放在 `internal/app/*` service
- collection 作为索引聚合视图，不影响 install/restore 语义
- 属于 MVP 主链路上的可见能力补充，完成后应至少执行 `go test ./...`

## 后续可扩展点

未来若需要扩展，可在当前结构上继续增加：

- `skillc collection show <collection>`
- `skillc collection skills <collection> --source <source>`
- `collection` 维度的 JSON 输出
- collection 级安装前的浏览/确认能力

当前版本先保持最小闭环，只实现用户已确认的两个命令。
