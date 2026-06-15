# TODO

<!--
简单的直接使用一行 checklist 说明即可。
需要附带较长说明的，使用标题+说明方式新建。使用emoji 表情状态图标(wait: ⏳|ing: 🔄|done: ✅)
-->

- [x] 支持链接方式安装 skill，方便多项目维护更新（Windows 默认 junction，其他系统默认 symlink；显式 symlink 在 Windows 无权限时自动回退到 copy）
- [x] 支持快速检测已经安装到项目的 skill 是否需要更新（当前项目维度已支持 `skillc update --check`）
  - [ ] 支持一键批量更新所有下游项目的 skills 版本
- [ ] 通过 web 界面管理 skills，包括安装、更新、删除、查看等操作
  - 参考 https://github.com/xingkongliang/skills-manager
  - 四期计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md`
  - 第一轮已新增 `skillc web` 本地查看入口和 plan-first 预览；Web 直接执行安装、更新、删除仍留到后续小阶段。
  - 五期计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`
  - 五期目标：在 Web 中补齐当前项目 profile apply / update 的计划、确认、执行闭环；卸载、source 删除和跨项目批量更新继续后置。
- [ ] 希望支持类似 skills-manager 场景的功能，在项目里可以快速切换使用不同组合的 skills
  - 可以在项目里配置不同的 skill 组合，每个组合可以包含多个 skill
  - 可以在项目里快速切换使用不同的 skill 组合，方便测试和观察效果
- [ ] 不要给 source 名称加 local/git 前缀; 现在无法方便的查看一个 source 的信息

## [ ] skillc 增强重构

设计草案：`docs/design/skillc-v0-enhance-design.md`
参考分析：`docs/design/skillc-reference-projects-analysis.md`
一期计划：`docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md`
一期状态：profile 最小闭环已完成，包括 profile 配置、from-installed/from-collection、apply/dry-run、source-scoped collection 浏览，以及移除 `install --collection` CLI。
二期计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`
三期计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase3-interactive-selection.md`
三期状态：`install --interactive`、`update --interactive`、`profile create --interactive` 已实现，基于 `gookit/cliui` 支持输入过滤和多选；最终验证记录见三期计划 Task 7。
四期计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md`
四期状态：第一轮新增 `skillc web` 本地管理入口，已覆盖 source/profile/status/install-map/version-drift 的查看和 plan-first 预览；Web 直接执行安装/更新/删除放到后续小阶段。
五期计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`
五期目标：Web 执行闭环第一步，只允许当前项目范围内 profile apply / update 在显式确认后执行；跨项目批量更新、卸载和 source 删除继续后置。

cli 优先，这样可以方便的进入任何目录进行操作
skillc 技能场景 - 自由选择多个技能配置为一个场景(eg: go-dev, flutter-dev)
在任意项目下 一键安装，启用 场景技能、集合技能
技能安装，更新 都支持交互式选择，过滤
web server 查看技能，源，也可以进行安装，更新，配置场景技能等管理操作

现有概念：
source -  skill源
collection - skill集合

参考项目：
- https://github.com/xingkongliang/skills-manager
- https://github.com/runkids/skillshare
- https://github.com/microsoft/apm
