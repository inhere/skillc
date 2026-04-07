# AGETNS for AI

**重要** 优先使用中文回复和响应

- 每完成一个阶段任务都需要在 plan 里对应地方标记完成；提交后再开始下一个任务。
- 变更实现时，若影响设计或任务状态，需要同步更新 `arch.md` 和 `plan.md`，避免文档漂移。
- CLI 层尽量只做参数解析、输出格式、错误返回；业务编排优先下沉到 `internal/app/*` service。
- install/restore 相关逻辑必须保持 `InstallEntry` 语义一致：写入 lock，restore 时基于 `SourceID + InstallEntry` 还原实际复制入口。
- 完成涉及 MVP 主链路的改动后，至少运行 `go test ./...` 再宣称完成。
- 如果有设计或开发方案，需要整理输出到 docs 目录下

## gcli 使用

- gcli.Command 的 Func 方法的第二个参数是解析后剩余的参数，一般不会直接使用