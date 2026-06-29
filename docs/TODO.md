# TODO

<!--
简单的直接使用一行 checklist 说明即可。
需要附带较长说明的，使用标题+说明方式新建。使用emoji 表情状态图标(wait: ⏳|ing: 🔄|done: ✅)
-->

- [x] 支持链接方式安装 skill，方便多项目维护更新（Windows 默认 junction，其他系统默认 symlink；显式 symlink 在 Windows 无权限时自动回退到 copy）

## skillc 使用问题


## skillc 使用优化

- [ ] skillc ins 去掉 --agent 选项的默认值，没有设置时通过 cliui 的 interact newui 交互让用户选择(可以多选)
- [ ] skillc ins -i keyword 会直接输出 no skills found. 修复并优化为使用 interact newui 交互让用户选择(可以多选)确认

## [x] skillc 增强重构

- [x] 支持快速检测已经安装到项目的 skill 是否需要更新（当前项目维度已支持 `skillc update --check`）
  - [x] 支持一键批量更新所有下游项目的 skills 版本
    - 七期计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`
    - 七期状态：已通过 project registry、`update --all-projects` 和 Web registered projects 更新闭环落地。
- [ ] 通过 web 界面管理 skills，包括安装、更新、删除、查看等操作
  - 参考 https://github.com/xingkongliang/skills-manager
  - 四期计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md`
  - 第一轮已新增 `skillc web` 本地查看入口和 plan-first 预览。
  - 五期计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`
  - 五期状态：已补齐当前项目 profile apply / update 的计划、确认、执行闭环；卸载、source 删除和跨项目批量更新继续后置。
  - 六期计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase6-web-current-project-management.md`
  - [x] 六期状态：已补齐当前项目 Web source add/sync/remove、profile save/from-installed/from-collection、uninstall plan/run 和最小操作历史；跨项目批量更新继续后置。
  - 七期计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`
  - [x] 七期状态：已补齐 registered projects、跨项目 update plan/run 和 per-project 确认边界。
- [ ] 希望支持类似 skills-manager 场景的功能，在项目里可以快速切换使用不同组合的 skills
  - 可以在项目里配置不同的 skill 组合，每个组合可以包含多个 skill
  - 可以在项目里快速切换使用不同的 skill 组合，方便测试和观察效果
- [x] 不要给 source 名称加 local/git 前缀; 现在无法方便的查看一个 source 的信息
  - 八期计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase8-registry-source-drift.md`
  - 八期状态：已支持 `source add <path-or-url> --id/--name/--ref`、legacy local/git 子命令的自定义 id/name、新 source ID 不再强加 `local-` / `git-` 前缀，并新增 `source info <id>`。
- [x] 支持 Registry 发现 source catalog
  - 八期计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase8-registry-source-drift.md`
  - 八期状态：已落地本机/HTTP JSON source catalog 的 `registry list/add/remove/sync/search/info/add-source`，`registry add-source` 只注册 source，不安装 Skill 或写 lock。
  - 复核说明：这是内部分享 source 的便利子集，不等于 PRD 中从 skills.sh / skillsmp / skillsllm 等 Registry 搜索并安装单个 Skill 的完整能力。
- [x] 修正 Registry 为 Skill 搜索/安装入口
  - 九期计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase9-registry-skill-search-install.md`
  - 九期状态：已支持 generic JSON catalog 的 Skill 级搜索、`registry install <registry>/<skill>` 直接安装、registry lock provenance、restore/status/update 对 registry record 的处理；`registry add-source` 保留为把 source 结果加入长期管理的可选入口。skills.sh / SkillsMP / SkillsLLM 真实 adapter 后置。
- [x] 支持 Git resolved ref / local checksum 的精确 drift 判断
  - 八期计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase8-registry-source-drift.md`
  - 八期状态：index/lock/status/Web 已记录并展示 checksum/ref drift，版本相同时也能通过 `status` / `update --check` 标记 outdated。

设计草案：`docs/design/skillc-v0-enhance-design.md`
参考分析：`docs/design/skillc-reference-projects-analysis.md`
一期计划：`docs/superpowers/plans/2026-06-13-skillc-v0-phase1-profile.md`
一期状态：profile 最小闭环已完成，包括 profile 配置、from-installed/from-collection、apply/dry-run、source-scoped collection 浏览，以及移除 `install --collection` CLI。
二期计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase2-status-update-check.md`
三期计划：`docs/superpowers/plans/2026-06-14-skillc-v0-phase3-interactive-selection.md`
三期状态：`install --interactive`、`update --interactive`、`profile create --interactive` 已实现，基于 `gookit/cliui` 支持输入过滤和多选；最终验证记录见三期计划 Task 7。
四期计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase4-web-management.md`
四期状态：第一轮新增 `skillc web` 本地管理入口，已覆盖 source/profile/status/install-map/version-drift 的查看和 plan-first 预览。
五期计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase5-web-execution.md`
五期状态：Web 执行闭环第一步已落地，只允许当前项目范围内 profile apply / update 在显式确认后执行；跨项目批量更新、卸载和 source 删除继续后置。
六期计划：`docs/superpowers/plans/2026-06-15-skillc-v0-phase6-web-current-project-management.md`
六期状态：当前项目 Web 管理补齐已完成，覆盖 source 管理、profile 管理、uninstall 和最小 history，仍不做跨项目批量更新、Registry、source ID 命名清理或精确 drift。
七期计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase7-cross-project-update.md`
七期状态：已新增 project registry、`project` CLI、`update --all-projects` 和 Web 跨项目更新闭环。
八期计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase8-registry-source-drift.md`
八期状态：已完成 source UX cleanup、JSON source catalog 子集和精确 drift metadata；完整 Registry skill search/install 修正进入九期。
九期计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase9-registry-skill-search-install.md`
九期状态：已补回 generic JSON Registry skill 搜索/安装主链路；公开 Registry 站点 adapter、信任模型和 Web Registry 页面后置。

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
