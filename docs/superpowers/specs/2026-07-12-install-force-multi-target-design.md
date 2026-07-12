# skillc install 强制覆盖与批量命令优化设计

| 修订日期 | 版本 | 说明 |
| --- | --- | --- |
| 2026-07-12 | v1 | 新增 install 强制覆盖、多 skill、多 agent，以及 rm 批量容错设计 |

实施计划：[2026-07-12-install-force-multi-target.md](../plans/2026-07-12-install-force-multi-target.md)

## 目标

- `skillc install -f/--force` 可覆盖当前 agent、scope 下由其他源安装的同名 skill。
- `install --agent` 接受逗号分隔的多个 agent name。
- `install` 接受多个空格分隔的 skill id/name，并兼容现有逗号列表。
- `skillc rm` 批量卸载时单项失败不再中断后续目标。

## 非目标

- 不改变未指定 `--force` 时的跨源同名冲突保护。
- 不删除其他 agent、其他 scope 或其他项目的安装记录。
- 不重构现有 install/uninstall 命令结构，不引入新依赖。

## 命令行为

### install 多目标与多 agent

位置参数按参数边界和逗号共同展开，去除空白与空项，并按首次出现顺序去重。例如：

```text
skillc install foo bar,baz --agent codex,claude-code
```

解析为 skill `foo`、`bar`、`baz`，分别安装到 `codex`、`claude-code`。`--skill` 继续作为单值兼容入口；显式提供 `--skill` 时沿用当前优先级，不与位置参数混合，避免暗中改变已有语义。

未显式指定 `--agent` 时，继续使用现有交互式 agent 多选流程。

### force 覆盖

安装服务获得显式 `Force` 标志。检测到当前 agent、scope 下存在相同 skill ID、但来源不同的记录时：

- `Force=false`：返回现有冲突错误。
- `Force=true`：安装新来源内容，并将当前 agent、scope 对应锁记录更新为新来源。

同一锁记录若还关联其他 agent，只移除本次 agent 的旧关联；其他 agent、scope 和项目记录保持不变。安装写入失败时不得提前丢弃旧锁记录。

### rm 批量容错

`rm` 对每个目标独立执行。某个目标无法解析、未安装或卸载失败时，将其加入失败结果并继续后续目标；最后统一打印成功项和失败项。只要存在失败项，命令返回汇总错误，使脚本仍能识别部分失败。

## 实现边界

- CLI 层负责收集位置参数、拆分逗号值、去重，并绑定 `--force`。
- `installapp` 负责强制覆盖的锁记录与文件安装语义，所有 CLI 安装路径复用该行为。
- uninstall 批处理在服务层返回逐项结果，CLI 只负责展示，避免循环中首错返回。

## 测试与验收

使用现有 `github.com/gookit/goutil/x/assert`，同一方法的多场景使用 `t.Run()`：

- `install foo bar,baz` 能解析并安装三个目标。
- `--agent codex,claude-code` 对两个 agent 安装，空白、空项和重复值被清理。
- 跨源同名安装在无 `--force` 时失败，在有 `--force` 时仅替换当前 agent/scope。
- force 安装失败时旧锁记录保持可用。
- `rm missing existing` 报告前项失败但仍卸载后项，并返回部分失败结果。
- 先运行聚焦测试，最终运行 `go test ./...`。

