# Git Source Sync 代理设计

## 背景

当前 `skillc` 已支持配置级 `proxy_url`，并且 `source sync` 在 Git source 场景下会通过 `sourceapp.Service.Sync` 调用 `gitx.Client.Sync` 执行 `git clone`。但现有实现里，`proxy_url` 只被持久化，尚未参与 Git 同步流程。

用户已确认本次需求边界：

- 仅当 source 类型为 `git` 且配置了 `proxy_url` 时生效
- 仅作用于 **本次由 skillc 发起的 Git 网络命令**
- 不写入任何全局或仓库级 `git config`
- 本地 Git 命令（如 `rev-parse`）不使用代理

## 目标

为 `skillc source sync` 增加 Git 代理支持，使 Git source 在配置了 `proxy_url` 时能通过代理执行远端访问，同时保持本地 Git 命令行为不变。

## 非目标

本次不包含：

- 修改用户全局 Git 配置
- 修改仓库本地 Git 配置
- 为所有 Git 命令无差别注入代理
- 改变 local source 的同步逻辑
- 扩展到 registry 或其他网络组件的代理策略统一化

## 方案对比

### 方案 A：命令级环境变量注入（推荐）

在 `gitx` 执行网络型 Git 命令时，仅为该次 `exec.Command` 注入代理环境变量。

优点：

- 只影响当前这次 Git 调用
- 不污染用户环境和 Git 配置
- 与需求“仅本次 git 命令生效”完全一致
- 后续若新增 `fetch`、`ls-remote` 等网络命令，也可沿用同一规则

缺点：

- 需要在 `gitx` 内明确区分网络命令与本地命令

### 方案 B：`git -c http.proxy=...` 临时参数注入

在联网命令上拼接 `-c http.proxy=...` / `-c https.proxy=...`。

优点：

- 同样是单次命令生效
- 不修改 git config 文件

缺点：

- 命令构造更分散
- 后续每种网络命令都要单独补参数，扩展性一般

### 方案 C：写入 Git 配置

通过 `git config --global` 或仓库级配置写入代理。

不采用原因：

- 作用范围超出本次 sync
- 会引入用户环境副作用
- 与已确认需求冲突

## 推荐方案

采用 **方案 A：命令级环境变量注入**。

理由：需求的核心约束是“仅本次网络 Git 命令使用代理，不修改 git config”。命令级环境变量最贴合这个边界，也能让代理逻辑集中在 `gitx`，避免 CLI 或应用服务层拼接 Git 参数。

## 设计细节

### 1. 分层职责

#### CLI 层

CLI 无需新增参数，也不承担代理逻辑。

`source sync` 仍然只负责：

- 解析 `source id`
- 调用 `sourceapp.Service.Sync`
- 输出同步结果

这保持现有约束：CLI 只做参数解析、输出格式、错误返回。

#### 应用服务层

`sourceapp.Service.Sync` 负责决定“这次同步是否应带代理”。

职责：

- 读取配置中的 `ProxyURL`
- 当 source 类型为 `git` 时，将 `proxy_url` 传给 Git 基础设施层
- local source 保持现有逻辑不变

这意味着应用层持有代理策略判断，基础设施层只负责执行。

#### 基础设施层

`gitx.Client` 负责：

- 对网络型 Git 命令注入代理环境变量
- 对本地 Git 命令保持现状
- 隐藏具体的 `exec.Command` 环境拼装细节

## 2. 接口调整

当前接口：

```go
type gitRunner interface {
    Sync(url, dir, ref string) (string, error)
}
```

建议调整为显式接收代理参数：

```go
type gitRunner interface {
    Sync(url, dir, ref, proxyURL string) (string, error)
}
```

对应地：

- `sourceapp.Service.Sync` 在 Git source 路径上传入 `data.ProxyURL`
- `gitx.Client.Sync` 根据 `proxyURL` 决定是否为网络命令注入代理
- 现有测试 stub 同步更新为可断言接收到的代理值

这样可以避免 `gitx` 直接依赖配置存储，也避免把代理判断散落到 CLI 层。

## 3. Git 命令执行规则

### 网络命令

当前 `Sync` 中唯一的网络命令是 `git clone`。

规则：

- 若 `proxyURL == ""`：按当前方式执行
- 若 `proxyURL != ""`：仅在执行 `git clone` 时注入代理环境变量

### 本地命令

当前 `Sync` 结束后会执行 `git rev-parse HEAD` 获取 `ResolvedRef`。

规则：

- `rev-parse` 不注入代理
- 即使 source 配置了 `proxy_url`，本地 Git 命令仍保持无代理执行

这样可以严格满足“只限网络命令”的要求。

## 4. 环境变量策略

实现层面，Git 网络命令应在子进程环境中附加代理变量，而不是修改当前进程全局环境。

建议注入：

- `HTTP_PROXY=<proxy_url>`
- `HTTPS_PROXY=<proxy_url>`
- `http_proxy=<proxy_url>`
- `https_proxy=<proxy_url>`

理由：

- 兼容大小写差异
- 仅对子进程生效
- 不污染外部 shell 环境

本次不额外处理 SOCKS 专用参数或 `NO_PROXY` 策略，因为用户需求中未提出，保持最小实现。

## 5. 数据流

### Git source sync（配置了代理）

1. CLI 调用 `sourceapp.Service.Sync(sourceID)`
2. `sourceapp` 读取配置，拿到 `ProxyURL`
3. 定位到 Git source 后调用 `gitx.Client.Sync(url, dir, ref, proxyURL)`
4. `gitx` 执行 `git clone`，在该子进程上注入代理环境变量
5. clone 成功后，`gitx` 执行 `git rev-parse HEAD`，不注入代理
6. 返回 `ResolvedRef`
7. `sourceapp` 更新 source 状态并重建索引

### Git source sync（未配置代理）

流程与当前一致，仅新增的 `proxyURL` 参数为空。

### Local source sync

行为完全不变。

## 错误处理

- 代理配置为空：按无代理路径执行，不报错
- 代理配置存在但 Git clone 失败：保持当前错误返回语义，不额外包装为“代理错误”
- 本地 Git 命令失败：保持当前 `rev-parse` 错误语义
- local source：不读取、不使用代理逻辑

本次不增加代理 URL 格式校验；若值非法，由 Git 命令失败直接暴露实际错误信息。

## 测试设计

遵循 TDD，先补失败测试，再做最小实现。

### `internal/infra/gitx/client_test.go`

新增或调整测试覆盖：

- 配置了 `proxyURL` 时，网络命令会带代理环境执行
- 未配置 `proxyURL` 时，不注入代理变量
- 本地命令 `rev-parse` 不带代理

说明：如果直接断言 `exec.Command` 环境不方便，可通过抽出命令构造或注入 runner 的方式验证传入环境。

### `internal/app/sourceapp/service_test.go`

新增或调整测试覆盖：

- Git source sync 会把 `ProxyURL` 传给 git runner
- 未配置 `ProxyURL` 时传空值
- Local source sync 不调用带代理的 Git 路径

### 回归测试

至少执行：

- `go test ./internal/app/sourceapp ./internal/infra/gitx`
- `go test ./...`

## 影响范围

预计变更文件：

- 修改：`internal/app/sourceapp/service.go`
- 修改：`internal/app/sourceapp/service_test.go`
- 修改：`internal/infra/gitx/client.go`
- 修改：`internal/infra/gitx/client_test.go`
- 修改：`arch.md`
- 修改：`plan.md`

## 与现有架构的一致性

该设计符合当前项目约束：

- CLI 层不承载业务编排
- 代理判断放在应用层，命令执行细节下沉到 `internal/infra/gitx`
- 不影响 `InstallEntry` / restore 语义
- 涉及 MVP 主链路，完成实现后需要至少执行 `go test ./...`

## 实施说明

实现阶段需要同步更新：

- `arch.md`：补充 Git source sync 在 `proxy_url` 存在时的代理行为
- `plan.md`：新增本任务并按阶段标记 TDD / 验证状态

本设计文档仅定义行为和分层边界，不在此阶段提交实现代码。
