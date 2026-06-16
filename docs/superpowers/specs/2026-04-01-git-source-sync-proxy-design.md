# Git Source Sync SyncOptions 与实时进度设计

## 背景

当前 `skillc` 已支持配置级 `proxy_url`，并且 `source sync` 在 Git source 场景下会通过 `sourceapp.Service.Sync` 调用 `gitx.Client.Sync` 执行 `git clone`。现有实现存在两个限制：

1. `gitx.Sync(url, dir, ref, proxyURL string)` 以位置参数传递代理，不利于后续扩展
2. `git clone` 在交互终端下不会像原生 `git clone` 一样实时显示进度

本次用户已确认需求边界：

- 将 `gitx.Sync` 最后一个参数升级为结构体参数，便于后续扩展
- 默认开启 clone 进度显示
- 仅在 **交互终端（TTY）** 下显示实时进度
- 进度输出到 `stderr`
- 结构体中可以预留 `Quiet` / `Verbose` 字段，但本次不强行扩展 CLI 语义
- `proxy_url` 仍仅作用于 **本次由 skillc 发起的 Git 网络命令**
- 不写入任何全局或仓库级 `git config`
- 本地 Git 命令（如 `rev-parse`）不使用代理，也不显示进度

## 目标

为 `skillc source sync` 的 Git 同步链路引入可扩展的 `SyncOptions`：

- 用结构体替代位置参数传递代理与输出控制
- 在交互终端中默认实时显示 `git clone` 进度
- 保持 stdout 结果输出稳定，不污染脚本消费场景
- 保持 local source 与本地 Git 命令行为不变

## 非目标

本次不包含：

- 修改用户全局 Git 配置
- 修改仓库本地 Git 配置
- 为所有 Git 命令无差别注入代理或输出
- 新增 CLI `--quiet` / `--verbose` 参数语义
- 统一 registry 或其他网络组件的代理/输出策略
- 为非交互场景强制打印 clone 进度

## 方案对比

### 方案 A：`SyncOptions` 显式参数对象（推荐）

```go
type SyncOptions struct {
    ProxyURL string
    Progress io.Writer
    Quiet    bool
    Verbose  bool
}

func (c *Client) Sync(url, dir, ref string, opts SyncOptions) (string, error)
```

优点：

- 直接解决“最后一个 string 参数不利扩展”的问题
- 后续若增加 `Depth`、`Auth`、`Mirror` 等选项，无需继续修改函数签名
- `proxy/progress/quiet/verbose` 都集中在 `gitx` 边界，职责清晰

缺点：

- 需要同步调整 `sourceapp` 的 `gitRunner` 接口与相关测试桩

### 方案 B：`CloneOptions` 只承载 clone 相关字段

优点：

- 对当前 `git clone` 场景更贴切

缺点：

- 若未来 `Sync` 改为 `clone + fetch + ls-remote` 等组合流程，名字会过窄
- 容易演变出多个近似 options 结构体

### 方案 C：更底层的 `ExecOptions` / `CommandOptions`

优点：

- 通用性最强

缺点：

- 对当前需求偏重
- 容易把本次简单需求抽象成执行器框架，超出 YAGNI 边界

## 推荐方案

采用 **方案 A：`SyncOptions` 显式参数对象**。

理由：它同时满足“把代理从位置参数升级为结构体参数”和“默认支持交互终端实时显示 clone 进度”两个目标，扩展性足够，但不会把本次需求设计得过重。

## 设计细节

### 1. 分层职责

#### CLI 层

CLI 不直接拼装 Git 参数，也不直接控制 Git 子进程细节。

`source sync` 仍然只负责：

- 解析 `source id`
- 调用 `sourceapp.Service.Sync`
- 输出同步结果

CLI 层保持“参数解析、输出格式、错误返回”的职责边界。

#### 应用服务层

`sourceapp.Service.Sync` 负责组装这次 Git 同步的 `SyncOptions`。

职责：

- 读取配置中的 `ProxyURL`
- 判断当前是否为交互终端
- 对 Git source 组装 `SyncOptions`
- local source 保持现有逻辑不变

建议在 `Service` 中增加可注入的交互终端判断函数，例如：

```go
isInteractive func() bool
```

默认实现使用真实终端判断，测试中可 stub 为 `true/false`，避免依赖真实控制台环境。

#### 基础设施层

`gitx.Client` 负责：

- 对网络型 Git 命令注入代理环境变量
- 对 `git clone` 应用进度输出选项
- 对本地 Git 命令保持静默执行
- 隐藏具体的 `exec.Command`、环境变量与 `Stdout/Stderr` 绑定细节

### 2. 接口调整

当前接口：

```go
type gitRunner interface {
    Sync(url, dir, ref, proxyURL string) (string, error)
}
```

调整后：

```go
type SyncOptions struct {
    ProxyURL string
    Progress io.Writer
    Quiet    bool
    Verbose  bool
}

type gitRunner interface {
    Sync(url, dir, ref string, opts gitx.SyncOptions) (string, error)
}
```

对应地：

- `sourceapp.Service.Sync` 在 Git source 路径上传入 `gitx.SyncOptions`
- `gitx.Client.Sync` 根据 `opts.ProxyURL`、`opts.Progress` 决定联网命令行为
- 现有测试 stub 同步更新为可断言接收到的 `SyncOptions`

### 3. 默认进度行为

`source sync` 的默认行为定义如下：

#### Git source + 交互终端

- `Progress = os.Stderr`
- `git clone` 实时显示原生进度

#### Git source + 非交互场景

- `Progress = nil`
- `git clone` 保持静默执行

#### 输出通道规则

- 进度统一走 `stderr`
- 业务结果与现有 CLI 文案继续走正常输出
- 这样不会污染 `stdout`，也更适合后续 `--json` / 脚本消费

#### `Quiet` / `Verbose`

- 这次先进入 `SyncOptions` 作为预留字段
- 本次不要求新增 CLI flag，也不要求实现复杂语义
- 当前默认行为仍以“TTY 显示进度，非 TTY 静默”为准

### 4. Git 命令执行规则

#### 网络命令

当前 `Sync` 中唯一的网络命令是 `git clone`。

规则：

- 若 `opts.ProxyURL == ""`：按当前方式执行
- 若 `opts.ProxyURL != ""`：仅在执行 `git clone` 时注入代理环境变量
- 若 `opts.Progress != nil`：将 clone 进度输出绑定到该 writer
- 为了稳定显示进度，`git clone` 参数建议显式追加 `--progress`

显式追加 `--progress` 的原因：Git 的进度输出对是否连接终端较敏感，直接声明更符合“像 `git clone` 一样实时显示”的目标。

#### 本地命令

当前 `Sync` 结束后会执行 `git rev-parse HEAD` 获取 `ResolvedRef`。

规则：

- `rev-parse` 不注入代理
- `rev-parse` 不绑定 `Progress`
- 即使 source 配置了 `proxy_url`，本地 Git 命令仍保持无代理、无进度输出执行

### 5. 环境变量策略

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

本次不额外处理 SOCKS 专用参数或 `NO_PROXY` 策略，因为需求中未提出，保持最小实现。

### 6. 数据流

#### Git source sync（交互终端 + 配置代理）

1. CLI 调用 `sourceapp.Service.Sync(sourceID)`
2. `sourceapp` 读取配置，拿到 `ProxyURL`
3. `sourceapp` 判断当前为交互终端，组装：

```go
gitx.SyncOptions{
    ProxyURL: data.ProxyURL,
    Progress: os.Stderr,
}
```

4. 定位到 Git source 后调用 `gitx.Client.Sync(url, dir, ref, opts)`
5. `gitx` 执行 `git clone --progress`，在该子进程上注入代理环境变量，并将输出绑定到 `stderr`
6. clone 成功后，`gitx` 执行 `git rev-parse HEAD`，不注入代理，也不绑定进度输出
7. 返回 `ResolvedRef`
8. `sourceapp` 更新 source 状态并重建索引

#### Git source sync（非交互终端）

流程相同，但 `Progress = nil`，因此 clone 保持静默。

#### Git source sync（未配置代理）

流程相同，但 `ProxyURL` 为空，不注入代理环境变量。

#### Local source sync

行为完全不变。

### 7. 错误处理

保持当前语义，不额外引入“代理错误”或“进度错误”新分类：

- `git executable not found: ...`
- `git clone failed: ...`
- `git rev-parse failed: ...`

如果开启了进度输出，Git 自身的报错信息会直接打印到终端；返回值仍保留现有错误包装，方便上层处理和测试断言。

本次不增加代理 URL 格式校验；若值非法，由 Git 命令失败直接暴露实际错误信息。

## 测试设计

遵循 TDD，先补失败测试，再做最小实现。

### `internal/infra/gitx/client_test.go`

新增或调整测试覆盖：

- `SyncOptions.ProxyURL` 只作用于 clone command
- 配置了 `ProxyURL` 时，clone command 带代理 env
- 未配置 `ProxyURL` 时，不注入代理变量
- `rev-parse` command 不带代理 env
- `Progress` 非空时，clone command 绑定输出 writer
- `Progress` 非空时，clone 参数包含 `--progress`
- `Progress` 为空时，clone command 不绑定输出 writer

`Quiet` / `Verbose` 本次只做结构预留，不要求补充复杂行为测试；如需要回归保护，可加一个“当前不会影响 clone 参数”的轻量断言。

### `internal/app/sourceapp/service_test.go`

新增或调整测试覆盖：

- Git source sync 会把 `SyncOptions.ProxyURL` 传给 git runner
- 交互终端时会传 `Progress`
- 非交互终端时 `Progress == nil`
- local source sync 不调用带 git options 的路径

建议让 `Service` 暴露可注入的 `isInteractive` 判断，以便测试稳定控制 TTY 分支。

### `internal/cli/app_test.go`

仅当交互终端判断逻辑最终放在 CLI 时，才需要新增 CLI 层测试；若判断保留在 `sourceapp.Service`，CLI 层无需新增与本次需求强绑定的测试。

### 回归测试

至少执行：

- `go test ./internal/app/sourceapp ./internal/infra/gitx`
- 如涉及 CLI 落点，再执行：`go test ./internal/cli`
- `go test ./...`

## 影响范围

预计变更文件：

- 修改：`internal/app/sourceapp/service.go`
- 修改：`internal/app/sourceapp/service_test.go`
- 修改：`internal/infra/gitx/client.go`
- 修改：`internal/infra/gitx/client_test.go`
- 可选修改：`internal/cli/app.go`
- 可选修改：`internal/cli/app_test.go`
- 修改：`mvp-arch.md`
- 修改：`mvp-plan.md`

## 与现有架构的一致性

该设计符合当前项目约束：

- CLI 层不承载业务编排
- `sourceapp` 负责组装业务语义上的 `SyncOptions`
- Git 命令执行细节继续下沉到 `internal/infra/gitx`
- 代理只影响 skillc 发起的网络 Git 命令
- clone 进度默认仅在交互终端显示，不污染脚本场景
- 不影响 `InstallEntry` / restore 语义
- 涉及 MVP 主链路，完成实现后需要至少执行 `go test ./...`

## 实施说明

实现阶段需要同步更新：

- `mvp-arch.md`：补充 `gitx.Sync` 使用 `SyncOptions`，以及 Git source sync 的默认进度行为
- `mvp-plan.md`：补充本任务对 `SyncOptions` 和 clone 进度输出的实现说明与阶段状态

本设计文档仅定义行为和分层边界，不在此阶段提交实现代码。
