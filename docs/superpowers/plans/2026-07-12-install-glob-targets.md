# Install Glob Targets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 install 增加完整 glob 与 source 全量 target，并统一交互、非交互匹配语义。

**Architecture:** `searchapp` 负责共享 glob matcher 与交互候选合并，CLI 只传递拆分后的 targets。普通交互关键词继续复用 repoindex Filter，glob 使用 Go `path.Match`，安装执行链保持不变。

**Tech Stack:** Go 标准库 `path`、gcli、`github.com/gookit/goutil/x/assert`

设计文档：[2026-07-12-install-glob-targets-design.md](../specs/2026-07-12-install-glob-targets-design.md)

---

### Task 1: 共享 glob target matcher

**Files:**
- Modify: `internal/app/searchapp/service.go`
- Test: `internal/app/searchapp/service_test.go`

- [x] **Step 1: 写服务层失败测试**

用 `t.Run()` 覆盖 `flutter-*`、`*testing`、`flutter-?-pro`、字符范围、QualifiedName、SourceQualifiedName、`superpowers/*` 跨 collection、裸 `*`、非法 `[`、多 pattern 去重。测试通过 `ResolveInstallTargets` 调用真实 matcher。

- [x] **Step 2: 验证 RED**

运行：

```text
go test ./internal/app/searchapp -run 'TestService_ResolveInstallTargets.*Glob' -count=1
```

预期：完整 glob 与 source 全量场景 FAIL；现有代码只支持 Skill ID 末尾前缀。

- [x] **Step 3: 最小实现**

在 `resolveInstallTargetMatches` 中按以下顺序处理：collection mode、裸 `*` 拒绝、精确 `source/*`、含 glob 元字符的 `path.Match`、普通精确 target。matcher 对 `ID`、`QualifiedName`、`SourceQualifiedName` 任一字段命中即加入结果，无匹配返回现有 `skill not found`。

- [x] **Step 4: 验证 GREEN 并提交**

运行：

```text
go test ./internal/app/searchapp -count=1
```

预期 PASS。勾选 Task 1 并提交：

```text
feat(skillc): support glob install targets
```

### Task 2: 交互 install 复用 glob matcher

**Files:**
- Modify: `internal/app/searchapp/service.go`
- Modify: `internal/cli/manage_cmd.go`
- Test: `internal/app/searchapp/service_test.go`
- Test: `internal/cli/app_test.go`

- [x] **Step 1: 写交互失败测试**

新增命令级测试，索引含多个 `flutter-` skill 和无关 skill，执行 `install -i 'flutter-*' --agent universal --yes`，用现有 selector stub 断言候选只包含匹配项且进入 multi-select。另加普通 `-i flutter` 测试保护模糊 Search。

- [x] **Step 2: 验证 RED**

运行：

```text
go test ./internal/cli -run 'TestInstallCommandInteractive.*(Glob|Keyword)' -count=1
```

预期 glob 用例 FAIL 并输出 `no skills found`，普通关键词用例保持 PASS。

- [x] **Step 3: 最小实现**

新增 `SearchInstallCandidates(targets []string, agent string)`：一次加载索引；glob target 调共享 matcher后应用 agent filter，普通 target 调现有 repoindex Filter；结果按 `skillIdentityKey` 去重。CLI 交互分支把 `splitInstallTargets(targetArg)` 传入该方法。

- [x] **Step 4: 验证 GREEN 并提交**

运行：

```text
go test ./internal/app/searchapp ./internal/cli -run 'Test.*(Glob|Interactive.*Keyword)' -count=1
```

预期 PASS。勾选 Task 2 并提交：

```text
fix(skillc): apply glob matching to interactive install
```

### Task 3: 完整验证与交付

- [ ] **Step 1: 运行格式、相关包和全仓门禁**

```text
gofmt -w internal/app/searchapp/service.go internal/app/searchapp/service_test.go internal/cli/manage_cmd.go internal/cli/app_test.go
go test ./internal/app/searchapp ./internal/cli -count=1
go test ./...
git diff --check
```

预期全部 PASS。

- [ ] **Step 2: 独立审查、跟踪和推送**

检查完整 git diff，修复 Critical/Important；勾选计划，关闭 `lite-tools-0c4`，提交计划状态。执行 `git pull --rebase`、`git push`，确认 `main` 与 `origin/main` 同步。
