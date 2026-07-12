# Install Force and Multi-Target Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `install` 增加强制覆盖、多 skill、多 agent，并让 `rm` 批量执行遇错后继续。

**Architecture:** CLI 只负责多值解析和选项传递；跨源覆盖由 `installapp.Service` 统一处理。卸载批处理保留逐项执行方式，但聚合错误后再返回，避免首错中断。

**Tech Stack:** Go、gcli、`github.com/gookit/goutil/x/assert`

设计文档：[2026-07-12-install-force-multi-target-design.md](../specs/2026-07-12-install-force-multi-target-design.md)

---

## 文件结构

- 修改 `internal/cli/manage_cmd.go`：install/rm 参数绑定、多值解析、force 透传。
- 修改 `internal/cli/app_test.go`：命令级多 skill、多 agent 与 rm 继续执行回归测试。
- 修改 `internal/app/installapp/service.go`：Service force 配置、跨源记录替换、卸载错误聚合。
- 修改 `internal/app/installapp/service_test.go`：force 精确作用域与卸载聚合测试。

### Task 1: install 多 skill 与多 agent 参数

- [ ] **Step 1: 写失败测试**

在 `internal/cli/app_test.go` 增加命令测试：执行 `install --yes --agent universal,claude-code hello world`，断言两个 skill 均在两个 agent 目录安装。测试配置复用当前 install 测试的临时 index、source 与 agent tool 设置。

- [ ] **Step 2: 验证 RED**

运行：`go test ./internal/cli -run 'TestInstallCommand.*Multiple' -count=1`

预期：FAIL，当前 install 位置参数不接受多个值，且逗号 agent 被当成单个名称。

- [ ] **Step 3: 最小实现**

在 `buildInstallCommand` 中将位置参数改为多值：

```go
c.AddArg("skill", "skill ID/name, allow multiple", false, true)
```

新增一个包内 helper，输入 `[]string`，对每一项再按逗号拆分、`TrimSpace`、忽略空项并按首次出现顺序去重；skill 位置参数和显式 agent 都复用此 helper。显式 `--skill` 仍优先且作为单个输入传入 helper。

- [ ] **Step 4: 验证 GREEN**

运行：`go test ./internal/cli -run 'TestInstallCommand.*Multiple' -count=1`

预期：PASS。

- [ ] **Step 5: 更新计划并提交**

勾选 Task 1，提交 `internal/cli/manage_cmd.go`、`internal/cli/app_test.go` 和本计划：

```text
feat(skillc): support multiple install targets and agents
```

### Task 2: force 覆盖当前 agent/scope 的跨源同名 skill

- [ ] **Step 1: 写失败测试**

在 `internal/app/installapp/service_test.go` 用 `t.Run()` 覆盖：

```go
service := NewService(lockFile).WithForce(true)
```

先从 source-a 安装 `ship` 到 `universal` 和 `claude-code`，再从 source-b 强制安装到 `universal`。断言 universal 内容和锁记录切到 source-b，claude-code 仍指向 source-a。另保留既有“不带 force 拒绝跨源同名”的测试。

- [ ] **Step 2: 验证 RED**

运行：`go test ./internal/app/installapp -run 'TestService_Install.*DifferentSources' -count=1`

预期：FAIL，`WithForce` 尚不存在。

- [ ] **Step 3: 最小服务实现**

在 `Service` 增加 `force bool` 和链式方法：

```go
func (s *Service) WithForce(force bool) *Service {
	clone := *s
	clone.force = force
	return &clone
}
```

`installInto` 找到冲突且 `force=false` 时保持原错误；`force=true` 时先执行文件安装，成功后从冲突记录移除当前 agent，空 agent 记录删除，再 `upsertRecord` 新来源记录。这样文件安装失败不会提前改锁数据。

- [ ] **Step 4: 验证服务 GREEN**

运行：`go test ./internal/app/installapp -run 'TestService_Install.*DifferentSources' -count=1`

预期：PASS。

- [ ] **Step 5: 写 CLI RED 并透传 force**

在 `internal/cli/app_test.go` 增加 `install --force --yes --agent universal source-b/ship` 命令测试，先确认 FAIL。然后给 `ManageOptions` 增加 `Force bool`，绑定 `c.BoolOpt(&opts.Force, "force", "f", false, "overwrite same skill installed from another source")`，并在 `runResolvedInstall` 创建服务时调用 `.WithForce(opts.Force)`。

- [ ] **Step 6: 验证 CLI GREEN 并提交**

运行：`go test ./internal/app/installapp ./internal/cli -run 'Test.*(Force|DifferentSources)' -count=1`

预期：PASS。勾选 Task 2 并提交：

```text
feat(skillc): add force install across sources
```

### Task 3: rm 批量失败后继续

- [ ] **Step 1: 写失败测试**

在 `internal/app/installapp/service_test.go` 新增测试：安装 `existing` 后调用 `UninstallMulti([]string{"missing", "existing"}, ...)`，断言返回错误，同时 `existing` 的安装目录和锁记录已删除。

- [ ] **Step 2: 验证 RED**

运行：`go test ./internal/app/installapp -run TestService_UninstallMultiContinuesAfterFailure -count=1`

预期：FAIL，当前实现遇到 `missing` 后立即返回，`existing` 未卸载。

- [ ] **Step 3: 最小实现**

使用标准库 `errors.Join` 聚合逐项错误：

```go
var errs []error
for _, skillID := range skillIDs {
	if err := s.Uninstall(skillID, agentName, scope); err != nil {
		errs = append(errs, fmt.Errorf("%s: %w", skillID, err))
	}
}
return errors.Join(errs...)
```

CLI 保持返回该汇总错误，不把部分失败误报为成功。

- [ ] **Step 4: 验证 GREEN 并提交**

运行：`go test ./internal/app/installapp -run TestService_UninstallMultiContinuesAfterFailure -count=1`

预期：PASS。勾选 Task 3 并提交：

```text
fix(skillc): continue batch uninstall after failures
```

### Task 4: 完整验证与交付

- [ ] **Step 1: 格式与聚焦测试**

运行：`gofmt -w internal/cli/manage_cmd.go internal/cli/app_test.go internal/app/installapp/service.go internal/app/installapp/service_test.go`

运行：`go test ./internal/cli ./internal/app/installapp -count=1`

预期：PASS。

- [ ] **Step 2: MVP 全量门禁**

运行：`go test ./...`

预期：PASS。

- [ ] **Step 3: 静态差异检查**

运行：`git diff --check`，确认只保留本任务文件与用户已有的 CodeQL 修改。

- [ ] **Step 4: 完成跟踪与提交**

勾选 Task 4，提交计划最终状态，关闭 `lite-tools-84c`；随后执行 `git pull --rebase`、`git push`、`git status --short --branch`，确认分支与 origin 同步且仅保留用户原有修改。

