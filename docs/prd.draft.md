# Skillc Product

我需要一个使用 go 实现的 skill 管理工具，统一管理本机的 skills。

- 支持添加本地路径作为 skill 仓库源
- 支持添加 git 仓库作为 skill 仓库源
- 支持从三方平台搜索获取 skill（如 skills.sh, skillsmp.com, skillsllm.com 等）
- 支持搜索，下载，安装，删除skill等功能
- 支持多种agent（如 claude-code, opencode, codex 等）
- 支持安装到 全局目录或项目目录
- 安装时都是从本地 cache 中获取，避免重复下载
- 支持 repo, skill 更新功能
  - 允许检查更新 repo 中的 skill 到最新版本

配置文件初步参考：

```yaml
# 代理地址 http 或 git 等操作时，可以配置使用代理
proxy_url: http://localhost:7890

agent_tools:
  claude-code:
    dirname: .claude
    # user_dir: ~/{dirname} # 默认路径
    # project_dir: {dirname} # 默认路径
  opencode:
    dirname: .opencode
    user_dir: ~/.config/opencode
  codex:
    dirname: .claude


lock_file: ~/.config/skillc/skillc-install.lock # json 文件
repo_cache_dir: ~/.local/cache/skillc-repo-caches

git_repos:


agents_repos:

skills_repos:
  - https://github.com/ComposioHQ/awesome-claude-skills
  - https://github.com/sickn33/antigravity-awesome-skills
search_registry:
  - https://skills.sh/
  - https://skillsmp.com/
  - https://skillsllm.com/
```

