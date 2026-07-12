# Skill Select Detail Test Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 `skillSelectItems` 测试与有意隐藏 collection 的终端展示行为一致。

**Architecture:** 生产逻辑保持不变，仅更新两个过期断言。测试继续保护稳定 target、Label/Detail 分离，并显式禁止 Detail 恢复 collection。

**Tech Stack:** Go、`github.com/gookit/goutil/x/assert`

设计文档：[2026-07-12-skill-select-detail-design.md](../specs/2026-07-12-skill-select-detail-design.md)

---

### Task 1: 同步 skillSelectItems 测试契约

**Files:**
- Modify: `internal/cli/app_test.go:188-224`

- [x] **Step 1: 验证现有 RED**

运行：

```text
go test ./internal/cli -run 'TestSkillSelectItems(UseStableSourceQualifiedTargets|KeepLabelAndDetailSeparate)' -count=1
```

预期：两个测试 FAIL，实际 Detail 为 `source=repo-a version=1.2.3`，旧断言仍要求 collection。

- [x] **Step 2: 更新最小断言**

保持 target 与 Label 断言，详情断言调整为：

```go
assert.Contains(t, items[0].Detail, "source=repo-a")
assert.Contains(t, items[0].Detail, "version=1.2.3")
assert.NotContains(t, items[0].Detail, "collection=")
```

另一个精确格式断言调整为：

```go
assert.Eq(t, "source=repo-a version=1.2.3", items[0].Detail)
```

- [x] **Step 3: 验证 GREEN**

运行相同聚焦命令，预期 PASS。

- [x] **Step 4: 运行完整门禁**

运行：

```text
go test ./internal/cli -count=1
go test ./...
git diff --check
```

预期全部 PASS。

- [x] **Step 5: 完成跟踪并提交推送**

勾选全部步骤，关闭 `lite-tools-3zo`，提交：

```text
test(skillc): align compact skill select detail
```

执行 `git pull --rebase`、`git push`，确认 `main` 与 `origin/main` 同步。
