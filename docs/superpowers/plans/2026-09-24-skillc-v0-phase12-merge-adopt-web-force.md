# Phase 12: Merge, Adopt, Web Force, Agent Canonicalization

**Goal:** 让「项目里改过的 skill」成为一等公民：按文件三方合并、把改动回写源、Web 提供 force 入口，并把 agent 名称在所有入口统一为正式名称。

**Architecture:** 合并判定放在 `internal/domain/install`（纯函数，输入 baseline/current/incoming 三份文件清单），执行与备份复用 `apputil.OverwriteGuard` 与 `installapp`；adopt 复用 `installapp` + `sourceapp` 重建索引；agent 归一化收敛到 `cfg.Config.CanonicalAgentName` + `lockstore` 的 resolver。

**Tech Stack:** Go、gcli、`github.com/gookit/goutil/x/assert`

---

## 背景

- Phase 11 之前，`up` 只能整体跳过或整体覆盖被本地改动的 skill；缺少按文件判断与回写能力。
- lock 记录里 agent 名混用别名（`claude`/`agents`）与正式名（`claude-code`/`universal`），
  导致 `up`（默认 `-a universal`）静默跳过部分记录。

## 文件结构

- `internal/domain/config/model.go`：`CanonicalAgentName` / `CanonicalAgentNames`。
- `internal/infra/lockstore/json_store.go`：`WithAgentResolver`，Load/Save 归一化 agent。
- `internal/domain/install/merge.go`（新增）：三份清单 → 合并计划（纯函数）。
- `internal/domain/lock/model.go`：`installed_files`（相对路径 → 内容哈希）。
- `internal/app/installapp/service.go`：安装后写入文件清单、按计划合并、adopt 支持。
- `internal/app/sourceapp/service.go`：`Reindex`（不拉取，仅重扫索引）。
- `internal/cli/manage_cmd.go`：`update --merge/--keep-local`、`adopt` 命令。
- `internal/app/webapp/*`：update/uninstall 的 `force` 入口与展示。

## Task 1: agent 名称统一为正式名称

- [x] `cfg.Config.CanonicalAgentName` / `CanonicalAgentNames`，默认 alias 补 `agents` → `universal`。
- [x] `lockstore.WithAgentResolver`：Load/Save 归一化 `record.Agents`（去重 + 排序）。
- [x] install/update/uninstall/list/project/web 入口统一归一化；`update -a` 支持逗号多选。
- [x] 测试：`config`、`lockstore`、`installapp`、`listapp`、`updateapp`。
- [x] 实测：`skillc ls` 在真实 lock 上把 `claude`/`agents` 显示为 `claude-code`/`universal`。

## Task 2: 按文件三方合并

- [x] `installed_files` 写入 lock（copy 模式），并在安装/更新后刷新。
- [x] `domain/install` 合并计划：本地未改→取上游；本地改+上游未改→保留本地；
      双方都改→冲突（默认保留本地 + 输出 `.incoming`）；上游删除→未改则删除。
- [x] `update --merge` 执行合并，`--force` 保持整体覆盖语义；报告 kept/merged/conflict。
- [x] 测试：三份清单的纯函数用例 + 端到端合并。

## Task 3: adopt（把项目改动回写源）

- [x] `skillc adopt <skill>`：把安装目录相对源的差异写回源目录（local source），
      git 源拒绝并提示改用工作副本。
- [x] `--dry-run` 输出计划；写回前备份源文件；写回后重建索引并刷新 lock 指纹。
- [x] 测试：local 源写回、git 源拒绝、dry-run 不写。

## Task 4: Web force 入口

- [x] update/uninstall 请求支持 `force`（update 另支持 `merge`），plan 中标记 `local_changes`。
- [x] 静态 UI 增加 force/merge 勾选与提示；跨项目更新同样透传；结果 JSON 暴露 `backed_up`/`merged`。
- [x] 测试：`manager_actions` 的 force/merge 透传与 uninstall force。

## Task 5: 文档与验证

- [x] README（中英）、`docs/TODO.md`、设计文档 changelog。
- [x] `gofmt`、`go vet ./...`、`go test ./...` 全绿 + CLI/Web 端到端冒烟。
