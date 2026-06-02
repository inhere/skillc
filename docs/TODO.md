# TODO

<!--
简单的直接使用一行 checklist 说明即可。
需要附带较长说明的，使用标题+说明方式新建。使用emoji 表情状态图标(wait: ⏳|ing: 🔄|done: ✅)
-->

- [x] 支持链接方式安装 skill，方便多项目维护更新（Windows 默认 junction，其他系统默认 symlink；显式 symlink 在 Windows 无权限时自动回退到 copy）
- [ ] 支持快速检测已经安装到项目的 skill 是否需要更新
  - [ ] 支持一键批量更新所有下游项目的 skills 版本
- [ ] 通过 web 界面管理 skills，包括安装、更新、删除、查看等操作
  - 参考 https://github.com/xingkongliang/skills-manager
- [ ] 希望支持类似 skills-manager 场景的功能，在项目里可以快速切换使用不同组合的 skills
  - 可以在项目里配置不同的 skill 组合，每个组合可以包含多个 skill
  - 可以在项目里快速切换使用不同的 skill 组合，方便测试和观察效果


参考项目：
- https://github.com/xingkongliang/skills-manager
- https://github.com/runkids/skillshare

