# Lockfile 项目分组与 agents 聚合设计

- 日期：2026-04-06
- 主题：按项目路径分组保存安装记录，并在 skill item 内聚合 `agents[]`

## 1. 背景

当前 lockfile 采用扁平 `[]Record` 结构，每条记录绑定单个 agent，并保存单一 `InstalledPath`。这与当前需求不匹配：

1. lockfile 需要按项目路径分组保存已安装 skills
2. 每个 skill item 内需要记录安装到哪些 agents，即 `agents []string`
3. `global` 安装记录需要独立保存到特殊 key，而不是混入某个项目路径
4. 不需要兼容旧版 lockfile

## 2. 目标

本次调整只解决 lockfile 持久化模型与相关 install/restore/update 行为，不扩大到无关重构。

目标包括：

- lockfile 顶层按 scope key 分组
- project scope 使用绝对项目路径作为 key
- global scope 使用 `__global__` 作为 key
- skill 安装记录按 skill 聚合，记录 `agents []string`
- restore 继续基于 `SourceID + InstallEntry` 还原实际复制入口
- 安装路径不再写入 lockfile，而是在运行时按 `scope + agent + project key` 重新计算

非目标：

- 不兼容旧版 lockfile 读取
- 不引入迁移逻辑
- 不改变 install source 的语义
- 不重做 CLI 输出格式，除非受新模型影响必须调整

## 3. 设计方案

### 3.1 顶层结构

lockfile 从当前的扁平数组改为按 scope key 分组：

```go
type File map[string][]Record
```

其中：

- `__global__`：保存 global scope 的安装记录
- `/abs/project/path`：保存对应项目的安装记录

示例：

```json
{
  "__global__": [
    {
      "skill_id": "hello-skill",
      "qualified_name": "marketplaces/hello-skill",
      "source_qualified_name": "repo-a/marketplaces/hello-skill",
      "source_id": "repo-a",
      "source_type": "local",
      "install_entry": "commands",
      "version": "1.0.0",
      "agents": ["claude-code", "codex"],
      "checksum": "abc",
      "installed_at": "2026-04-06T10:00:00Z",
      "updated_at": "2026-04-06T10:05:00Z",
      "pinned": false
    }
  ],
  "/workspace/demo": [
    {
      "skill_id": "foo",
      "qualified_name": "foo",
      "source_qualified_name": "repo-b/foo",
      "source_id": "repo-b",
      "source_type": "git",
      "install_entry": ".",
      "version": "2.0.0",
      "agents": ["claude-code"],
      "checksum": "def",
      "installed_at": "2026-04-06T11:00:00Z",
      "updated_at": "2026-04-06T11:00:00Z",
      "pinned": false
    }
  ]
}
```

### 3.2 Record 模型

`internal/domain/lock/model.go` 中的 `Record` 调整如下：

- 删除：`Agent string`
- 删除：`InstalledPath string`
- 新增：`Agents []string`

保留字段：

- `SkillID`
- `QualifiedName`
- `SourceQualifiedName`
- `Version`
- `SourceID`
- `SourceType`
- `InstallEntry`
- `Checksum`
- `InstalledAt`
- `UpdatedAt`
- `Pinned`

设计理由：

- 同一 skill 在多个 agent 上安装时，`InstalledPath` 不再是单值事实，继续保存会产生歧义
- 安装路径本来就可由 `scope + agent + workdir` 结合 agent resolver 重新计算，因此不应持久化为 lock 真相
- `Agents []string` 能准确表达“这个 skill 在该 scope key 下安装到了哪些 agent”

### 3.3 scope key 规则

新增统一规则函数（命名可实现时确定）：

- project scope -> 使用绝对项目路径
- global scope -> 使用 `__global__`

该规则应在 install / uninstall / restore / update 中统一复用，避免 key 生成不一致。

## 4. 行为设计

### 4.1 install

输入：`skill + agent + scope + workdir`

流程：

1. 根据 `scope + workdir` 计算 lockfile key
2. 根据 `scope + agent + workdir` 计算实际安装目录
3. 执行文件安装：`item.Path + InstallEntry -> targetPath`
4. 在该 key 对应的记录列表中查找匹配 skill record
5. 若不存在则创建 record，并写入当前 agent 到 `agents`
6. 若存在则向 `agents` 合并当前 agent，并刷新版本/来源/更新时间
7. 保存整个 lock file

匹配维度建议为：

- `SourceID + SkillID`
- 如需要避免同 ID 不同来源歧义，可保留 `SourceQualifiedName` 辅助判断

安装时的时间语义：

- 新记录：`InstalledAt = UpdatedAt = now`
- 已有记录追加 agent 或刷新版本：保留原 `InstalledAt`，更新 `UpdatedAt`

### 4.2 uninstall

输入：`skill + agent + scope + workdir`

流程：

1. 计算 lockfile key
2. 在该 key 下匹配目标 skill record
3. 计算当前 agent 的安装路径并执行删除
4. 从 record 的 `agents` 中移除该 agent
5. 若移除后 `agents` 为空，则删除整个 record
6. 保存 lockfile

边界：

- 只影响当前 key 下的记录，不跨项目或全局分组删除
- 同一个 skill 若同时装在多个 agent，卸载单个 agent 不应删除整个 skill record

### 4.3 restore

输入：`sourcePaths map[string]string`

流程：

1. 遍历 lockfile 所有 key
2. `__global__` 按 global scope 处理，其他 key 视为 project scope，且 key 本身就是对应项目路径
3. 对每条 record：
   - 用 `SourceID` 找到 source 根路径
   - 用 `InstallEntry` 计算复制入口
   - 对 `agents` 中每个 agent 重新解析目标安装路径
   - 分别执行安装
4. 返回 restore 结果

关键点：

- restore 不再依赖 `InstalledPath`
- restore 的安装目标完全依赖 lock key、agent、scope 和 resolver 实时计算
- 继续保持 `SourceID + InstallEntry` 为恢复复制源的真实依据

### 4.4 update

update 从 lockfile 读取“聚合 skill record”，再按 agent 展开执行实际重装。

流程调整：

1. 读取所有 key 下的 records
2. 根据用户目标筛选要更新的 skill records
3. 对每个 record 的每个 agent：
   - 依据 key 判断 scope / project 路径
   - 重新解析目标安装路径
   - 执行 reinstall/update
4. 写回更新后的聚合 record

匹配维度建议优先使用：

- `lock key + SourceID + SkillID`

必要时可补充：

- `SourceQualifiedName`
- `QualifiedName`

### 4.5 list

如果已安装列表依赖 lock records，则需要同步适配：

- 读取聚合 record
- 按 `agents` 展开为用户可读输出，或在输出层直接显示聚合后的 agents

本次不要求额外设计新的展示格式，但实现时需要保证已有 list/update 流程可消费新结构。

## 5. 影响范围

核心文件：

- `internal/domain/lock/model.go`
- `internal/infra/lockstore/json_store.go`
- `internal/app/installapp/service.go`
- `internal/app/updateapp/service.go`

高概率受影响的测试：

- `internal/infra/lockstore/json_store_test.go`
- `internal/app/installapp/service_test.go`
- `internal/app/updateapp/service_test.go`
- `tests/e2e/install_restore_test.go`

文档：

- `arch.md`
- `plan.md`

## 6. 实现边界与约束

- CLI 层继续只做参数解析、输出格式、错误返回
- lockfile 结构变化的业务编排优先放在 `internal/app/*`
- restore 语义必须保持：基于 `SourceID + InstallEntry` 找到真实复制入口
- 完成主链路改动后至少运行 `go test ./...`
- 若实现过程中影响设计或任务状态，需要同步更新 `arch.md` 和 `plan.md`

## 7. 测试策略

### 7.1 lockstore

覆盖：

- 新结构的保存与读取
- 空分组 / 多分组序列化
- `__global__` 与项目路径 key 正常往返

### 7.2 installapp

覆盖：

- project scope 安装写入项目路径 key
- global scope 安装写入 `__global__`
- 同一 skill 安装到多个 agent 时聚合到同一 record 的 `agents`
- 重复安装同一 agent 不产生重复 agent 项
- 卸载单个 agent 仅移除该 agent
- 卸载最后一个 agent 删除 record

### 7.3 restore

覆盖：

- 从项目 key 恢复到 project scope 安装目录
- 从 `__global__` 恢复到 global scope 安装目录
- 单条 record 的多个 agents 都会被恢复
- restore 复制源仍取自 `SourceID + InstallEntry`

### 7.4 updateapp

覆盖：

- 能从聚合 lock records 正确筛选目标
- 能按每个 agent 展开 update/reinstall
- 不同 key 下相同 skill_id 不串记录

### 7.5 e2e

至少保留一条主链路：

- install -> lock save -> uninstall -> restore

并验证：

- lockfile 顶层为对象而不是数组
- project / global 分组符合预期
- item 中 `agents` 内容符合预期

## 8. 风险与规避

### 风险 1：旧的“单 record 单 agent”匹配逻辑残留

影响：install/uninstall/update 可能误判记录或重复写入。

规避：

- 把 record 匹配逻辑抽到统一函数
- 明确区分“匹配 skill record”和“匹配 record 内某个 agent”两个层次

### 风险 2：restore/update 仍依赖 `InstalledPath`

影响：删除字段后会造成运行错误或测试失败。

规避：

- 统一改为运行时解析目标路径
- 为 restore/update 增补按 key 反推 scope/project path 的辅助函数

### 风险 3：project key 生成不一致

影响：同一项目可能写到不同 key，导致记录分裂。

规避：

- 所有写入和查询均复用同一个 key 计算函数
- project scope 一律先转绝对路径再入 lock

## 9. 推荐实施顺序

1. 调整 `lock.Record` 与 `lockstore` 文件结构
2. 改造 install/uninstall 主写入逻辑
3. 改造 restore 逻辑，移除 `InstalledPath` 依赖
4. 改造 update 逻辑，按 `agents` 展开执行
5. 更新 list / e2e / 单元测试
6. 同步更新 `arch.md` 与 `plan.md`
7. 运行 `go test ./...`

## 10. 结论

采用方案 A：

- lockfile 顶层按项目路径 / `__global__` 分组
- skill item 聚合保存 `agents []string`
- 不再保存 `InstalledPath`
- restore 继续基于 `SourceID + InstallEntry` 计算复制入口
- 不兼容旧版 lockfile，直接切换到新格式
