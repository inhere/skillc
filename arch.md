# Skillc Architecture

## 1. 目标与范围

### 1.1 文档目标

本文基于 `prd.md` 与 `prd-review.md`，给出 Skillc 的总体技术架构、模块边界、核心数据流、里程碑拆分与测试策略，作为后续实现与任务分解的设计依据。

### 1.2 MVP 范围

MVP 聚焦打通本地 Skill 生命周期的主链路，覆盖以下能力：

- 配置初始化与配置读写
- 本地路径源与 Git 仓库源管理
- Skill 索引与搜索
- 安装、卸载、列表展示
- Lock File 记录与批量恢复
- 多 Agent 安装适配（`claude-code`、`opencode`、`codex`）
- 缓存复用与基础环境诊断

### 1.3 非目标

本阶段明确不做：

- GUI 或 Web 管理界面
- 在线账号体系与云端同步
- Skill 发布平台
- Skill 运行时兼容性治理
- 复杂依赖树与依赖解析
- 开发者软链接模式（后续版本再支持）

---

## 2. 设计原则

### 2.1 Source 与 Agent 解耦

Skillc 的核心价值是把“Skill 来源管理”和“Agent 安装适配”拆开。来源层负责发现与缓存，Agent 层负责落地安装，二者通过统一 Skill 模型衔接。

### 2.2 缓存优先

所有安装动作都应优先从本地缓存读取。只有在缓存不存在、失效或用户显式触发同步时，才访问 Git 或 Registry。

### 2.3 CLI 与业务逻辑分离

CLI 只负责参数解析、输出格式与错误码。具体业务流程放在应用服务层，避免命令实现膨胀为难测试的脚本式逻辑。

### 2.4 标准化数据模型

无论 Skill 来自 local、git 还是 registry，最终都映射为统一的 `Skill` 模型；无论安装到哪个 Agent，最终都写入统一的 `LockRecord` 模型。

### 2.5 人机双模式

默认输出要对人类用户友好，同时通过 `--yes`、`--dry-run`、`--json` 支持 CI/CD 与脚本化场景。

### 2.6 小步可扩展

MVP 先打通 `source -> index -> install -> lock` 主链路，再扩展 `registry`、`update`、`doctor` 强化能力，避免首期过度设计。

---

## 3. 总体架构

### 3.1 分层结构

建议采用“核心领域 + 应用服务 + 基础设施 + CLI 装配”的四层架构：

1. `cmd/`
   - CLI 入口与子命令注册
   - 参数解析、帮助文本、输出格式、退出码

2. `internal/app/`
   - 应用服务层
   - 编排完整业务流程：`init`、`source add`、`search`、`install`、`uninstall`、`list`

3. `internal/domain/`
   - 领域模型与核心规则
   - 定义 `Skill`、`Source`、`LockRecord`、`InstallPlan` 等模型与校验/规划规则

4. `internal/infra/`
   - 基础设施实现
   - 文件系统、Git、缓存、YAML/JSON 持久化、文件锁、checksum、终端交互等

### 3.2 架构选择对比

#### 方案 A：核心领域 + 应用服务 + CLI 装配（推荐）
- 优点：模块边界清晰，业务规则集中，可测试性最好，适合后续增加 `update` / `registry`
- 缺点：前期设计成本略高于直接命令驱动

#### 方案 B：传统 CLI 命令驱动
- 优点：起步快
- 缺点：命令层容易直接拼接业务逻辑，后续复杂度会迅速上升

#### 方案 C：任务/事件流驱动
- 优点：适合更复杂的异步编排与可观测性
- 缺点：对当前 MVP 过重

最终建议采用 **方案 A**。

### 3.3 核心调用链

典型安装调用链建议为：

`cmd install` -> `installapp.Service` -> 读取配置 -> 检查/刷新缓存 -> 定位 Skill -> 解析 Agent 目录 -> 生成安装计划 -> 检测冲突 -> 执行复制 -> 写入 Lock File -> 输出结果

---

## 4. 核心模块设计

### 4.1 config

职责：
- 解析默认配置路径和 `--config`
- 自动生成最小默认配置
- 展开 `~`、环境变量、相对路径
- 提供统一运行时配置对象

边界：
- 只负责“配置是什么”
- 不负责安装、同步、搜索等业务流程

### 4.2 source

职责：
- 管理 `local` / `git` / `registry` 三类来源
- 提供增删查列与同步能力
- 维护 `last_sync_at`、`status`、`error_message`

边界：
- 只关心来源管理与同步状态
- 不直接负责安装到 Agent

说明：
- CLI 层面对 local/git 使用 `skillc source`
- CLI 层面对 registry 使用 `skillc registry`
- 数据模型层统一抽象为 `Source`

### 4.3 repo index

职责：
- 扫描本地源或 Git 缓存目录
- 发现 Skill 子目录
- 解析 `SKILL.md` YAML front matter
- 生成标准化 Skill 索引

边界：
- 负责把来源转换为可搜索的 Skill 列表
- 不负责安装目标推导与 lock 维护

### 4.4 cache

职责：
- 管理 repo cache、skill cache、registry cache
- 记录索引构建结果
- 提供缓存命中与过期判断

边界：
- 只负责存储与路径组织
- 不承载业务策略判断

### 4.5 agent adapter

职责：
- 根据 `agent + scope` 解析目标安装目录
- 隐藏不同 Agent 的目录结构差异
- 保证安装产物结构满足对应 Agent 的加载约定

边界：
- 只关心“装到哪里、目录如何组织”
- 不关心 Skill 来源细节

### 4.6 install

职责：
- 生成安装计划
- 做冲突检测
- 执行复制安装
- 支持卸载与批量恢复
- 驱动写入 lock file

边界：
- 是 MVP 最核心的业务编排层
- 依赖 source/index/cache/agent/lock，但不直接实现底层细节

### 4.7 lock

职责：
- 维护已安装 Skill 记录
- 提供查询、追加、删除、批量恢复输入
- 原子写入与并发保护

边界：
- 只记录“安装事实”
- 不负责重新解析来源数据

---

## 5. 核心数据模型

### 5.1 Config

```go
type Config struct {
    ProxyURL         string
    AgentTools       map[string]AgentToolConfig
    LockFile         string
    RepoCacheDir     string
    SkillCacheDir    string
    RegistryCacheDir string
    Sources          SourceConfigGroup
    Registries       []RegistryConfig
}
```

### 5.2 AgentToolConfig

```go
type AgentToolConfig struct {
    Dirname    string
    UserDir    string
    ProjectDir string
}
```

### 5.3 Source

```go
type Source struct {
    ID           string
    Type         SourceType // local | git | registry
    Name         string
    Path         string
    URL          string
    Ref          string
    LastSyncAt   time.Time
    Status       string
    ErrorMessage string
}
```

### 5.4 Skill

```go
type Skill struct {
    ID              string
    Name            string
    Description     string
    Version         string
    Tags            []string
    SupportedAgents []string
    SourceID        string
    SourceType      SourceType
    InstallEntry    string
    Homepage        string
    Author          string
    License         string
}
```

### 5.5 LockRecord

```go
type LockRecord struct {
    SkillID       string
    Agent         string
    Scope         string // global | project
    Version       string
    SourceID      string
    SourceType    string
    InstallEntry  string
    ResolvedRef   string
    InstalledPath string
    Checksum      string
    InstalledAt   time.Time
    UpdatedAt     time.Time
    Pinned        bool
}
```

### 5.6 InstallPlan

```go
type InstallPlan struct {
    SkillID         string
    Agent           string
    Scope           string
    SourcePath      string
    InstallEntry    string
    TargetRoot      string
    TargetPath      string
    ConflictMode    string // overwrite | skip | cancel
    ExistingManaged bool
    DryRun          bool
}
```

### 5.7 关键建模原则

1. **Source 与 Registry 在模型层统一**
   - CLI 体验分离，数据模型统一
   - 便于 lock、search、update 共用逻辑

2. **Skill 是标准化结果，不是原始文件快照**
   - 上游差异在 adapter/parser 层消化
   - 下游 install/list/update 只依赖统一 Skill 模型

3. **LockRecord 是安装事实最小闭包**
   - 只保留恢复安装所必需的信息
   - 不复制整份 metadata，避免锁文件膨胀与漂移

4. **路径解析统一收口**
   - 所有 Agent 目录推导必须经过 resolver
   - 禁止各命令自行拼路径，避免跨平台失控

---

## 6. 关键业务流程

### 6.1 `skillc source add local <path>`

1. CLI 解析参数
2. `sourceapp` 校验路径与去重
3. 生成或确认 source id / name
4. 写回配置
5. 触发索引构建
6. 输出 source 状态

### 6.2 `skillc search <keyword>`

1. 读取配置与来源列表
2. 检查本地索引缓存
3. 从 index 中查询 Skill
4. 关联 lock 记录补充安装状态
5. 以 table 或 JSON 输出结果

### 6.3 `skillc install <skill-id>[@<version>]`

1. 读取配置
2. 根据 skill-id 从索引定位 Skill
3. 若缓存过期，先同步相关来源
4. 通过 agent adapter 解析目标目录
5. 生成 `InstallPlan`
6. 检测路径冲突
7. 执行复制安装
8. 计算 checksum
9. 原子写 lock file
10. 输出安装结果

### 6.4 `skillc install`（无参数，批量恢复）

1. 读取 lock file
2. 遍历每条已安装记录
3. 基于 `SourceID` 重新定位来源路径
4. 基于 `InstallEntry` 还原实际复制入口
5. 来源有效则重新安装
6. 汇总成功/失败结果

### 6.5 `skillc uninstall <skill-id>`

1. 查询 lock 记录
2. 定位安装路径
3. 删除目标产物
4. 删除对应 lock 记录
5. 返回幂等结果

---

## 7. 目录结构建议

建议采用如下最小可扩展目录：

```text
cmd/
  skillc/
    main.go

internal/
  app/
    configapp/
      service.go
    sourceapp/
      service.go
    searchapp/
      service.go
    installapp/
      service.go
    listapp/
      service.go
    doctorapp/
      service.go

  domain/
    agent/
      model.go
      resolver.go
    cache/
      model.go
    config/
      model.go
      defaults.go
    install/
      model.go
      planner.go
      conflict.go
    lock/
      model.go
    skill/
      model.go
      parser.go
    source/
      model.go
      sync_policy.go

  infra/
    configstore/
      yaml_store.go
    lockstore/
      json_store.go
    repoindex/
      scanner.go
      parser.go
      search.go
    gitx/
      client.go
    fsx/
      copy.go
      atomic.go
      paths.go
    hashx/
      sha256.go
    agentfs/
      installer.go
    termui/
      prompt.go
      output.go
    filelock/
      lock.go
```

### 7.1 关键文件职责建议

- `domain/skill/parser.go`
  - 只负责解析 `SKILL.md` front matter
- `infra/repoindex/scanner.go`
  - 负责扫描来源目录并发现 Skill 根目录
- `domain/install/planner.go`
  - 负责安装计划生成与目标路径规划
- `domain/install/conflict.go`
  - 负责路径冲突判定
- `infra/agentfs/installer.go`
  - 负责文件复制与删除，不负责规则决策

---

## 8. 里程碑与实施顺序

### 8.1 M1：配置、来源、索引、搜索

目标：让系统先“认识 Skill”。

交付：
- `skillc init`
- `skillc config show/get/set`
- `skillc source add/list/remove/sync/status`
- local/git 来源索引
- `skillc search`
- `skillc show`
- `skillc doctor` 基础检查

完成标志：
- 能添加至少一个 local 源和一个 git 源
- 能扫描并搜索 Skill
- 能看到标准化 Skill 元数据

### 8.2 M2：安装主链路

目标：打通“可安装、可卸载、可恢复”。

交付：
- agent adapter
- `skillc install`
- `skillc uninstall`
- `skillc list`
- lock file 原子写入
- checksum
- 冲突处理
- `skillc install` 无参数批量恢复

完成标志：
- 能安装到指定 Agent 的 global/project 目录
- 能卸载并正确清理 lock
- 能从 lock file 恢复安装
- 能识别 installed/missing/orphan 状态

### 8.3 M3：更新与 Registry

目标：打通“来源刷新 + 技能升级 + 第三方搜索”。

交付：
- `skillc registry add/list/remove/refresh`
- registry search adapter
- `skillc outdated`
- `skillc update`
- `skillc cache info/clean/rebuild`
- `skillc validate`
- shell completion

完成标志：
- 能从至少一个 registry 搜索并安装 Skill
- 能识别并更新已安装 Skill
- 单个来源失败不影响整体命令执行

### 8.4 首批开发顺序

#### Phase 1：底座
1. 配置模型与默认路径
2. 路径展开与目录初始化
3. 文件锁、原子写、checksum
4. CLI 主入口与公共输出

#### Phase 2：来源与索引
5. source 配置读写
6. local source add/list/remove
7. git source add/list/remove/sync
8. `SKILL.md` parser
9. repo scanner + index cache
10. search/show

#### Phase 3：安装
11. agent resolver
12. install planner
13. conflict detector
14. copy installer
15. lock store
16. install / uninstall / list / restore

#### Phase 4：诊断与补强
17. doctor
18. missing/orphan 状态检查
19. dry-run / yes / json
20. Windows 路径与边界修补

### 8.5 首批任务组建议

#### Task Group A：项目骨架
- 建立 `cmd/skillc` 和 `internal/{app,domain,infra}`
- 接入基础 CLI 框架
- 统一错误输出与退出码约定

#### Task Group B：配置系统
- Config model
- 默认路径解析
- YAML store
- `init/show/get/set`

#### Task Group C：Source 管理
- Source model
- local/git source CRUD
- git sync
- source status

#### Task Group D：Skill 索引
- `SKILL.md` front matter parser
- repo scanner
- index persistence
- search/show

#### Task Group E：安装系统
- agent resolver
- install planner
- conflict handling
- file copy/remove
- checksum
- lock file store

#### Task Group F：状态与诊断
- list installed
- missing/orphan detection
- doctor

### 8.6 MVP 命令优先级建议

第一批建议优先锁定以下命令：

```bash
skillc init
skillc config show
skillc source add
skillc source list
skillc source sync
skillc search
skillc install
skillc uninstall
```

第二批再补充：

```bash
skillc list
skillc show
skillc doctor
skillc config get
skillc config set
skillc install   # 无参数 restore
```

---

## 9. 测试策略

### 9.1 单元测试优先模块

建议高密度覆盖以下规则模块：

- `skill/parser`：`SKILL.md` front matter 解析
- `agent/resolver`：global/project 路径解析
- `install/conflict`：冲突判定
- `install/planner`：安装计划生成
- `config/defaults`：默认路径与展开逻辑
- `lockstore`：lock 记录增删改查与状态判断

目标：
- 把最容易演化的规则稳定住
- 为后续 `update/registry` 演进提供回归保护

### 9.2 集成测试重点

建议使用临时目录开展集成测试，覆盖以下关键链路：

- `source add local` -> 扫描 -> `search`
- `source add git` -> sync -> 扫描 -> `search`
- `install` -> 文件落地 -> lock 写入
- `uninstall` -> 文件删除 -> lock 删除
- `install` 无参数 -> restore

测试环境尽量使用临时目录模拟：
- config dir
- cache dir
- fake agent dir
- fake skill repo

避免依赖真实用户目录。

### 9.3 外部依赖测试策略

- **Git**：通过接口隔离，单测中使用 stub；少量集成测试再调用真实 git
- **Registry**：M3 通过 HTTP test server 模拟
- **终端交互**：prompt 逻辑抽象成接口，测试中注入固定选择

---

## 10. 风险与后续扩展

### 10.1 主要技术风险

#### 风险一：Agent 目录规则漂移
问题：不同 Agent 未来目录结构可能变化，Windows 路径兼容也容易出错。

控制方式：
- 所有路径解析统一收口到 `agent resolver`
- 禁止命令层自己拼接安装路径

#### 风险二：来源扫描规则复杂化
问题：过早支持太多目录变体会显著增加索引器复杂度。

控制方式：
- MVP 只支持“每个 Skill 一个子目录，子目录含 `SKILL.md`”
- 无 `SKILL.md` 的目录只做有限提示，不进入完整安装链路

#### 风险三：install / lock / cache 耦合失控
问题：CLI 工具很容易把查找、复制、写锁、输出揉进一个命令函数。

控制方式：
- `installapp.Service` 只负责编排
- planner / conflict / installer / lockstore 明确分层
- 每个步骤都可单测

#### 风险四：MVP 范围膨胀
问题：若首轮同时压入 registry、update、doctor 全量能力，交付风险会快速上升。

控制方式：
- 第一阶段聚焦来源、搜索、安装、卸载、list/restore
- `registry` 与 `update` 明确延后到 M3

### 10.2 后续扩展方向

- `registry` 多站点适配与统一下载缓存
- `update` / `outdated` / `pinned` 完整升级链路
- `doctor` 诊断与修复建议增强
- `validate` 面向 Skill 作者的包结构校验
- `link` 开发者软链接模式
- 批量安装与批量更新
- shell completion 与更丰富的终端交互输出
