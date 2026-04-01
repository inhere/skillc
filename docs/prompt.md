# Prompts


## 2026.04.01

source sync scan 可能还有问题。类似这个本地仓库 ~/.claude/plugins/marketplaces/claude-code-workflows
  skills 在 plugins/{collection}/skills 下。

优化调整：
- 加入 skills 集合的概念
- install/uninstall 需要支持快捷安装/卸载一个集合的 skills
- 如果 source 没有找到skills目录，即skill在根目录
  - 如果扫描到多个skill, source name 当作集合名称
  - 只有一个skill, 则是一个独立的顶级 skill，没有集合名称
- 如果 source 找到了 skills 目录，且只有一个。source name 当作集合名称
- 如果 source 下有很多 skills 目录，类似 claude-code-workflows，skills 父目录名当作集合名称
  - 一个 source 可以有多个集合

请思考并规划如何调整实现