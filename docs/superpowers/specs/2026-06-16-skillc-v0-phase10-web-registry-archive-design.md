# Skillc v0 Phase 10 Web Registry and Archive Download Design

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-16 | v0.1 | Codex | 规划 Web Registry 页面与 registry skill archive `download_url` 下载安装能力 |

状态：Draft

相关文档：

- 总设计：`docs/design/skillc-v0-enhance-design.md`
- Phase 9 计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase9-registry-skill-search-install.md`
- Phase 10 实施计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md`
- PRD：`docs/prd.md`
- MVP 架构：`docs/mvp-arch.md`

## 1. 背景

P9 已经把 Registry 修正回 PRD 的主定位：从 registry catalog 搜索 Skill 级结果，并支持 `registry install <registry>/<skill>` 不经过 `add-source` 直接安装。当前剩余两个明显缺口：

- Web 管理页还没有 Registry 视图，用户仍需要用 CLI 查看 registry、搜索 skill、同步 catalog 和安装结果。
- Registry `SkillEntry` 已有 `download_url` 字段，但 materializer 仍强依赖 `source_url`，导致 zip/tar.gz archive 形式的 skill 无法直接安装。

P10 目标是把这两个缺口作为一个小阶段闭环：Web 上能完成 registry skill 的查看、搜索、同步、安装到当前项目；底层 materializer 能把 archive `download_url` 变成本地 registry skill cache snapshot，再复用既有 install/lock/update 链路。

## 2. 设计结论

推荐方案：**Archive download foundation + Web Registry 当前项目 MVP 一起做，但严格限制范围。**

Archive download 是 registry skill 安装链路的基础能力；Web Registry 如果只支持 `source_url` 结果，会把用户带到一个不完整的入口。因此 P10 应先实现 archive materializer，再把 Web Registry 接到已存在的 `registryapp` / `installapp` 服务上。

Web 页面只覆盖当前项目安装，不做跨项目批量 registry install。跨项目更新在 P7 已有独立闭环，registry skill 一旦安装并写入 lock，后续 status/update 已能继续沿用现有流程。

## 3. 范围

P10 做：

- `registry.SkillEntry.download_url` 支持 archive materialization。
- 支持 archive 格式：`.zip`、`.tar.gz`、`.tgz`。
- `download_url` 支持 `http://`、`https://`，测试和本机 catalog 支持本地文件路径。
- 本地 registry catalog 中相对 `download_url` 按 catalog 文件目录解析；远程 registry catalog 要求 `download_url` 是 HTTP(S) 绝对 URL。
- 下载 archive 后校验 SHA-256 checksum；支持 `sha256:<hex>` 和 raw hex 两种写法。
- 缺少 checksum 时允许安装，但在 Web/plan 结果中展示 warning。
- 解压时防止 archive path traversal，确保所有文件都落在目标 cache 目录内。
- Web 新增 Registry 页面，支持 registry list、skill search、source search、sync、install skill、add source。
- Web 所有写操作保持 plan-first：plan 预览，run 必须 `confirm:true`。
- Web action 记录 history：`registry.sync`、`registry.install`、`registry.add_source`。

P10 不做：

- 不接入真实 skills.sh / SkillsMP / SkillsLLM 站点 adapter。
- 不做账号、token、私有 registry auth。
- 不做签名、评分、审核、信任策略。
- 不做跨项目批量 registry install。
- 不做 profile/bundle registry 安装。
- 不做 archive cache GC 和全量 artifact 管理。
- 不重构 Web 静态资源打包；仍沿用当前单文件 embedded HTML，除非实施中发现局部拆分能明显降低风险。

## 4. Archive Download 设计

### 4.1 Materializer 输入优先级

`registryapp.materializer.Materialize` 使用以下策略：

1. `source_url` 非空：保留 P9 行为，Git URL 走 `gitx.Sync`，本地目录走 snapshot copy。
2. `source_url` 为空且 `download_url` 非空：下载或读取 archive，校验 checksum，安全解压到 registry cache。
3. 两者都为空：返回清晰错误。

`skillFromEntry` 不再要求 `source_url`，只要求 entry 已经被 materialize 到 `root`。这样 download-only entry 可以映射成 `source_type=registry` 的 `skill.Skill`。

### 4.2 URL 与路径规则

本地 catalog：

- `download_url: ./archives/go-pro.zip` 会解析为 `<catalog-dir>/archives/go-pro.zip`。
- 绝对文件路径保持原样。
- HTTP(S) URL 保持原样。

远程 catalog：

- `download_url` 必须是 `http://` 或 `https://`。
- 相对 URL 暂不支持，避免隐式拼接规则和安全边界变复杂。

### 4.3 Checksum 规则

`checksum` 表示 archive 原始字节的 SHA-256，而不是解压后目录 checksum。原因是：

- 下载过程可以边读边算 hash。
- 发布者容易生成 archive checksum。
- P10 不引入 manifest/signature，避免把 checksum 语义扩大到解压目录。

支持两种写法：

- `sha256:0123...abcd`
- `0123...abcd`

checksum 不匹配时 install 必须失败。checksum 缺失时允许继续，但 plan/result 显示 warning：`checksum is missing; archive integrity is not verified`。

### 4.4 Archive 安全

解压 zip/tar 时必须检查每个条目：

- 清理后的目标路径必须位于目标 cache 目录下。
- 跳过目录条目或按需创建目录。
- 拒绝绝对路径、`..` 逃逸路径、Windows drive path 逃逸。
- 普通文件写入前创建父目录。
- 文件权限使用保守默认值；不恢复可疑特殊文件。

P10 只解压普通文件和目录。symlink、device、hardlink 等特殊 tar entry 直接拒绝或跳过，避免 registry archive 能写出 cache 目录。

## 5. Web Registry 设计

### 5.1 页面组织

推荐布局：在现有 Web sidebar 中新增 `Registry` 一级页面。

页面分三块：

- 顶部工具条：keyword、registry filter、result kind segmented control（Skills / Sources）、Sync All。
- 主结果表：展示 skill/source search 结果，支持查看详情、install skill、add source。
- 右侧或下方 action panel：展示选中结果详情、plan 预览、确认执行按钮和 warning。

备选方案是三列工作台（registries / results / detail），但当前 Web 页面是单 sidebar + 单内容区域的结构，三列会增加 `manager_static.go` 的复杂度，也不如现有页面一致。P10 采用现有结构的单 Registry 页面即可。

### 5.2 Web API

新增查询 API：

- `GET /api/registries`
- `GET /api/registry/skills?keyword=&registry=`
- `GET /api/registry/sources?keyword=&registry=`

新增写操作 API：

- `POST /api/registry/sync/plan`
- `POST /api/registry/sync/run`
- `POST /api/registry/install/plan`
- `POST /api/registry/install/run`
- `POST /api/registry/add-source/plan`
- `POST /api/registry/add-source/run`

P10 不新增 Web registry add/remove。Registry catalog 的新增/删除仍先通过 CLI 完成，Web 第一轮聚焦“发现和安装”。

### 5.3 Plan / Run 语义

Sync plan：

- 输入：`registry_id` 可空；为空表示 sync all。
- 输出：将要同步的 registry 列表、当前位置、当前状态。

Sync run：

- 输入：`confirm:true`、`registry_id` 可空。
- 行为：调用 `registryapp.Sync(id)` 或 `SyncAll()`。
- history：`registry.sync`。

Install plan：

- 输入：`target`、`agent`、`scope`、`work_dir`。
- 行为：只解析 registry skill metadata，不下载、不解压、不安装。
- 输出：registry、skill id、version、agent、scope、install_entry、source/download、checksum 状态、warnings。

Install run：

- 输入：`confirm:true`、`target`、`agent`、`scope`、`work_dir`。
- 行为：materialize registry skill，然后调用 `installapp.RunResolved` 安装到当前项目/global scope。
- history：`registry.install`。

Add-source plan：

- 输入：`entry_id`、可选 `id`、`name`、`sync`。
- 行为：解析 source entry，预览将创建的 source。

Add-source run：

- 输入：`confirm:true`、`entry_id`、可选 `id`、`name`、`sync`。
- 行为：调用 `registryapp.AddSource`。
- history：`registry.add_source`。

## 6. 数据流

Archive install 数据流：

```text
registry catalog
  -> registryapp.Sync normalize/cache
  -> registryapp.MaterializeSkill(selector)
  -> materializer downloads/extracts archive into registry cache
  -> skill.Skill(source_type=registry, path=cache snapshot)
  -> installapp.RunResolved
  -> agent install dir + skillc lock
```

Web Registry install 数据流：

```text
browser
  -> /api/registry/install/plan
  -> Manager.PlanRegistryInstall
  -> registryapp.InfoSkill
  -> browser confirmation
  -> /api/registry/install/run confirm:true
  -> Manager.RunRegistryInstall
  -> registryapp.MaterializeSkill
  -> installapp.RunResolved
  -> history + refreshed dashboard/status
```

## 7. 测试策略

Archive materializer：

- local zip download_url 解压成功。
- local tar.gz download_url 解压成功。
- checksum `sha256:` 校验成功。
- checksum raw hex 校验成功。
- checksum mismatch 返回错误。
- zip path traversal 返回错误，且不在 cache 外写文件。
- tar path traversal 返回错误，且不在 cache 外写文件。
- download-only entry 能通过 `skillFromEntry` 映射 registry provenance。

Registry service normalization：

- 本地 catalog 相对 `download_url` 解析为 catalog 目录下路径。
- 远程 catalog 相对 `download_url` 返回错误。
- 远程 catalog HTTP(S) `download_url` 保持可用。

Web manager/server：

- `GET /api/registries` 返回配置 registry。
- registry skill/source search API 返回缓存结果。
- install plan 不 materialize，不写 lock。
- install run 要求 `confirm:true`。
- install run 成功后返回 installed runtime record，并记录 history。
- sync/add-source plan-run 遵循现有确认模式。

最终验证：

```bash
go test ./internal/domain/registry ./internal/app/registryapp ./internal/app/webapp ./internal/cli
go test ./...
```

## 8. 风险与取舍

- `manager_static.go` 已经较大，但 P10 不引入前端构建系统，避免扩大工具链。实施时可以把 Registry 的 JS 渲染函数局部整理清楚，但不做全量拆分。
- checksum 缺失仍允许安装，是为了支持内部临时 registry；Web 必须明确显示 warning，避免用户误以为 archive 已验证。
- 远程 catalog 不支持相对 `download_url`，短期不够灵活，但规则明确，安全边界简单。后续如果接入 provider adapter，可由 adapter 生成绝对 URL。
- archive 解压只支持普通文件和目录，会牺牲少量特殊包能力，但符合 skill 安装场景。

## 9. 验收标准

- CLI `registry install <registry>/<skill>` 能安装只有 `download_url` 的 zip/tar.gz registry skill。
- Web Registry 页面能列出 registry、搜索 skill/source、同步 registry、安装 registry skill 到当前项目、从 source result 添加 source。
- Web 写操作全部有 plan/run 两阶段，run 缺少 `confirm:true` 会被拒绝。
- 安装后的 registry skill 继续能被 `status`、`update --check`、`update` 识别。
- Archive checksum mismatch 和 path traversal 被测试覆盖。
- `go test ./...` 通过。
