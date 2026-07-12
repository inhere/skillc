# Install 完整 Glob Target 设计

| 修订日期 | 版本 | 说明 |
| --- | --- | --- |
| 2026-07-12 | v1 | 统一非交互与交互 install 的完整 glob、source 全量展开语义 |

实施计划：[2026-07-12-install-glob-targets.md](../plans/2026-07-12-install-glob-targets.md)

## 背景

非交互 install 目前只把末尾 `*` 解释为 Skill ID 前缀；交互 install 则把 pattern 原样交给普通文本 Search，导致 `skillc ins -i 'flutter-*'` 搜索字面量星号并返回 `no skills found`。`superpowers/*` 同样被误当作 Skill ID 前缀，无法表达安装一个 source 的全部 skills。

## 目标

- 支持完整 glob：`*`、`?`、字符范围及中间通配。
- glob 同时匹配 `Skill.ID`、`QualifiedName`、`SourceQualifiedName`。
- `source/*` 按精确 SourceID 展开该 source 下全部 collection 的 skills。
- 非交互解析与交互候选筛选复用同一 matcher。
- 保留普通交互关键词的现有模糊搜索。

## 匹配与错误规则

- 使用 Go 标准库 `path.Match`，不新增依赖。
- pattern 含 glob 元字符时进入 glob matcher；无 glob 的交互关键词仍调用普通 Search。
- `source/*` 优先按精确 SourceID 处理，不受 `*` 不跨 `/` 的规则限制。
- 裸 `*` 返回明确错误，禁止一次选择全部 source 的全部 skills。
- 非法 pattern 返回包含原 pattern 的 glob 格式错误，不降级为文本搜索。
- glob 无匹配返回 `skill not found: <pattern>`。
- 多 pattern 的匹配结果按稳定 skill identity 去重并保持索引顺序。

## 命令行为

```text
skillc ins -s global 'superpowers/*'
skillc ins 'flutter-*' '*-testing'
skillc ins -i 'flutter-*'
skillc ins -i 'superpowers/*'
```

PowerShell 中 glob 参数应使用单引号，避免 shell 提前展开。

交互 glob 只负责生成候选，之后复用现有 multi-select、agent 选择、安装计划和确认流程。非交互 glob 复用现有 ResolveInstallTargets、去重、批量安装和失败汇总流程。

## 实现边界

- `internal/app/searchapp/service.go`：共享 glob matcher、source 全量匹配、交互候选组合。
- `internal/cli/manage_cmd.go`：交互路径把多个 target 交给共享候选解析；普通关键词行为不变。
- 不修改 repoindex 通用 Search，避免改变 search/profile/web 调用方。
- 不引入新的 glob 包、命令或配置。

## 测试与验收

- 前缀、后缀、中间、`?` 和字符范围 glob 均可匹配。
- ID、QualifiedName、SourceQualifiedName 三类字段均可匹配。
- `source/*` 跨 collection 展开全部 source skills。
- 裸 `*` 被拒绝；非法 glob 返回明确错误。
- 多 pattern 结果去重。
- `ins -i 'flutter-*'` 产生正确候选并进入多选。
- `ins -i flutter` 继续使用模糊搜索。
- `go test ./internal/app/searchapp ./internal/cli` 与 `go test ./...` 全部通过。
