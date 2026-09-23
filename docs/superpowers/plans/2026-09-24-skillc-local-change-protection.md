# Local Change Protection Implementation Plan

**Goal:** `skillc up` / `install` / `uninstall` 不再静默覆盖或删除安装目录里的本地改动。

**Architecture:** lock 记录里新增 `installed_checksum`（copy 模式安装目录的部署指纹）。`installapp` 在覆盖或删除前统一调用 `apputil.OverwriteGuard`：内容与指纹不一致时默认拒绝，`--force` 或缺少指纹时先备份旧目录。`updateapp` 把拒绝映射为 skipped 并把备份路径回传到 CLI/Web；`statusapp` 负责只读展示。

**Tech Stack:** Go、gcli、`github.com/gookit/goutil/testutil/assert`

---

## 背景

- `up` 是 `update` 的别名，copy 模式重装走 `agentfs.installCopy` 逐文件覆盖，源里存在的文件会被静默重写。
- lock 里原有的 `checksum` 是「安装时索引里的源 checksum」，不能当部署指纹：`Restore` 只更新 `install_mode` 不更新 checksum，索引过期时也会错。
- link 模式（symlink/junction）下项目目录就是源目录，覆盖不会丢内容；风险转移到源仓库缓存（`git fetch` + `reset --hard` + `clean -fd`）。

## 文件结构

- 新增 `internal/infra/hashx/dir.go` 的 `SumDeployed`：忽略 `__pycache__`、`node_modules`、`*.pyc/*.pyo/.DS_Store/Thumbs.db` 的目录指纹。
- 新增 `internal/infra/fsx/copy.go`：`CopyDir` / `CopyFile` 目录快照。
- 新增 `internal/domain/install/deployed.go`：`Drift` 判定与 `Fingerprint`。
- 新增 `internal/app/apputil/deployed.go`：`OverwriteGuard`（拒绝策略 + 备份）与 `ErrLocalChanges`。
- 修改 `internal/domain/lock/model.go`：`installed_checksum`。
- 修改 `internal/domain/config/*`、`internal/infra/configstore/yaml_store.go`、`internal/app/configapp/service.go`：`backup_dir`（默认 `~/.cache/skillc/backups`）。
- 修改 `internal/app/installapp/service.go`：`installInto` / `ReinstallAtPath` / `Restore` 覆盖前守卫、安装后写入指纹、`Uninstall` 删除前守卫、uninstall plan 标记 `local_changes`。
- 修改 `internal/app/updateapp/service.go`：`Req.Force`、`Result.BackedUp`、`ErrLocalChanges` → skipped。
- 修改 `internal/app/statusapp`、`listapp`、`webapp`：`LocallyModified` / `modified` 计数与展示。
- 修改 `internal/infra/gitx/client.go`：缓存脏检查。
- 修改 CLI：`update -f`、`uninstall -f`、`install -f` 帮助与输出。

## Task 1: 部署指纹与守卫

- [x] 新增 `hashx.SumDeployed`（保持 `SumDir` 语义不变，仅用于 source 扫描）。
- [x] 新增 `fsx.CopyDir`/`CopyFile`。
- [x] 新增 `domain/install.DetectDrift`/`Fingerprint`：只有 `install_mode == copy` 且记录过指纹时才可比对。
- [x] 新增 `apputil.OverwriteGuard`：一致直接放行；不一致默认 `ErrLocalChanges`；`--force` 或缺少指纹时先备份到 `<backup_dir>/<scope-label>/<skill>/<timestamp>`。

## Task 2: 安装链路接入

- [x] `installInto` / `ReinstallAtPath` / `Restore` 覆盖前守卫，安装成功后写入 `installed_checksum` 与生效的 `install_mode`。
- [x] `Restore` 跳过被改动的目录并通过 `CommandResult.Skipped` 上报，不再整体中断。
- [x] `Uninstall` 删除前守卫，`--force` 才删；`PlanUninstall` 输出 `local_changes`。

## Task 3: update 与展示

- [x] `updateapp`：`Req.Force` 透传到 reinstall 服务；`ErrLocalChanges` 计入 `Skipped`，备份路径进入 `Result.BackedUp`。
- [x] CLI：`update -f`、`uninstall -f`，打印 `backed up` / `skipped`；跨项目更新同样透传 force。
- [x] `status` / `update --check` 在 Reason 列标记 `locally modified`，summary 输出 `modified=` 计数；Web 状态项与 metric 增加 `local_modified` / `modified`。

## Task 4: git 缓存与测试隔离

- [x] `gitx.Sync` 在 `reset --hard`/`clean -fd` 与删除缓存重建前检查 `git status --porcelain`，有本地改动则返回 `ErrDirtyCache`。
- [x] 测试配置统一覆盖 `backup_dir`，避免测试写入真实 `~/.cache/skillc/backups`。

## Task 5: 验证

- [x] 新增/更新测试：`hashx`（部署指纹忽略运行时产物）、`apputil`（拒绝/备份/link 跳过）、`installapp`（指纹写入、拒绝、强制备份、restore 跳过、卸载拦截）、`updateapp`（skipped 与备份上报）、`statusapp`（modified 标记）、`gitx`（脏缓存拒绝）。
- [x] `gofmt`、`go vet ./...`、`go test ./...` 全部通过，且不再向真实 home 写入备份目录。
