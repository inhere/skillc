# Skill 选择详情精简设计

| 修订日期 | 版本 | 说明 |
| --- | --- | --- |
| 2026-07-12 | v1 | 明确交互选择详情有意隐藏 collection，并同步测试契约 |

实施计划：[2026-07-12-skill-select-detail.md](../plans/2026-07-12-skill-select-detail.md)

## 背景与决策

`skillDetail()` 曾展示 `source`、`collection` 和 `version`。collection 会使终端列表过长，而 source 已足以表达来源，因此提交 `93014df` 有意隐藏 collection。当前两个失败测试仍断言旧展示格式，属于测试契约未同步，不是生产逻辑回归。

## 变更范围

- 保持 `internal/cli/interactive_cmd.go` 不变。
- 更新 `internal/cli/app_test.go` 中两个 `skillSelectItems` 测试：
  - 稳定 target 仍使用 source-qualified name。
  - Label 仍展示 skill 名称和 ID。
  - Detail 只展示非空的 source 与 version。
  - 明确断言 Detail 不包含 `collection=`，记录精简终端显示的产品决策。

## 验收

- 两个指定测试通过。
- `go test ./internal/cli` 通过。
- `go test ./...` 通过。
- 不修改生产代码，不新增依赖。
