# Skillc v0 Phase 11 SkillsMP Provider Adapter Design

| 修订时间 | 版本 | 作者 | 说明 |
| --- | --- | --- | --- |
| 2026-06-19 | v0.1 | Codex | 规划首个真实 Registry provider adapter：SkillsMP 远程搜索与安装映射 |

状态：Draft

相关文档：

- 总设计：`docs/design/skillc-v0-enhance-design.md`
- Phase 10 设计：`docs/superpowers/specs/2026-06-16-skillc-v0-phase10-web-registry-archive-design.md`
- Phase 10 实施计划：`docs/superpowers/plans/2026-06-16-skillc-v0-phase10-web-registry-archive.md`
- PRD：`docs/prd.md`
- MVP 架构：`docs/mvp-arch.md`

## 1. 背景

P9/P10 已经补齐 generic JSON Registry 的 Skill 级搜索、安装、archive download 和 Web Registry 页面，但它仍要求用户或团队先准备 `skillc-registry.json`。这覆盖了内部分享场景，没有覆盖 PRD 里更早提到的公开 skills 收集站点发现能力。

已探测的公开站点里，SkillsMP 最适合作为第一个真实 provider：

- 搜索 API 可直接返回 JSON：`GET https://skillsmp.com/api/v1/skills/search?q=<query>&page=<page>&limit=<limit>`。
- 返回字段包含 `id`、`name`、`description`、`author`、`githubUrl`、`skillUrl`、`stars`、`updatedAt`。
- `githubUrl` 形如 `https://github.com/<owner>/<repo>/tree/<ref>/<path>`，可以稳定映射到现有 `registry.SkillEntry` 的 `source_url`、`source_ref` 和 `install_entry`。

Phase 11 只做 SkillsMP。skills.sh 和 SkillsLLM 已确认存在可搜索 API，但安装定位字段不如 SkillsMP 稳定，后续等 provider 模型验证后再做。

## 2. 设计结论

推荐方案：**新增最小 provider registry 类型，只实现 SkillsMP 远程搜索 adapter，安装继续复用现有 Registry materialize/install 链路。**

Provider adapter 不新增独立安装模型。它只负责把远程搜索结果转换成 `registry.SkillEntry`，然后写入现有 `registry-index.json` cache。后续 `registry info`、`registry install`、Web Registry install 都继续读同一份 cache。

P11 不做 provider interface 大框架。当前只有一个 provider，实现可以是 `registryapp` 内部的一个小 adapter 文件和少量分支；等第二个 provider 真的落地时，再抽象公共接口。

## 3. 用户路径

新增 SkillsMP registry：

```bash
skillc registry add https://skillsmp.com --id skillsmp --name SkillsMP --provider skillsmp
```

搜索：

```bash
skillc registry search go --registry skillsmp
```

查看详情：

```bash
skillc registry info skillsmp/<skill-id>
```

安装：

```bash
skillc registry install skillsmp/<skill-id> --agent codex --scope project
```

Web Registry 页面沿用现有查询和安装入口。用户选择 SkillsMP registry 并输入 keyword 后，页面展示远程搜索结果，安装按钮继续走现有 plan/run。

## 4. 配置模型

现有 `type: local` 和 `type: http` 保持不变：

- `local`：本地 `skillc-registry.json`。
- `http`：远程 generic JSON catalog URL。

P11 新增：

```yaml
registries:
  - id: skillsmp
    name: SkillsMP
    type: provider
    provider: skillsmp
    url: https://skillsmp.com
```

`registry add` 行为：

- 未传 `--provider` 时，保持现有行为：HTTP(S) value 仍创建 `type: http`。
- 传 `--provider skillsmp` 时，创建 `type: provider`，`provider: skillsmp`。
- `url` 默认清理为站点 base URL，允许 `https://skillsmp.com/`。

不做 host 自动推断。显式 `--provider` 可以避免把普通 HTTP JSON catalog 误判成站点 adapter。

## 5. SkillsMP 映射规则

SkillsMP API 响应中的单条结果：

```json
{
  "id": "kbediako-co-skills-codex-orchestrator-skill-md",
  "name": "codex-orchestrator",
  "author": "Kbediako",
  "description": "Primary entrypoint...",
  "githubUrl": "https://github.com/Kbediako/CO/tree/main/skills/codex-orchestrator",
  "skillUrl": "https://skillsmp.com/creators/kbediako/co/skills-codex-orchestrator",
  "stars": 2,
  "updatedAt": "1781679284"
}
```

映射为：

```json
{
  "id": "kbediako-co-skills-codex-orchestrator-skill-md",
  "name": "codex-orchestrator",
  "description": "Primary entrypoint...",
  "source_url": "https://github.com/Kbediako/CO.git",
  "source_ref": "main",
  "install_entry": "skills/codex-orchestrator",
  "homepage": "https://skillsmp.com/creators/kbediako/co/skills-codex-orchestrator",
  "registry_id": "skillsmp",
  "registry_url": "https://skillsmp.com",
  "tags": ["skillsmp", "author:Kbediako"]
}
```

GitHub tree URL 解析规则：

- 只接受 `https://github.com/<owner>/<repo>/tree/<ref>/<path>`。
- `source_url` 生成 `https://github.com/<owner>/<repo>.git`。
- `source_ref` 使用 tree 中的 `<ref>`。
- `install_entry` 使用 tree 中的 `<path>`。
- 不支持无法解析 path 的结果；这些结果跳过并计入 warning。

`updatedAt` 暂不映射为 `version`。它是站点索引更新时间，不等同于 skill 版本；把它写成版本会让 update/status 语义变脏。

## 6. 搜索与缓存语义

SkillsMP provider 是远程搜索，不是完整 catalog mirror：

- `registry search <keyword> --registry skillsmp` 会请求 SkillsMP API。
- keyword 不能为空；空 keyword 返回清晰错误。
- 默认请求 `page=1&limit=50`。
- 搜索成功后，把本次返回的 normalized `SkillEntry` 合并进 `registry-index.json`。
- 同 registry、同 skill id 的旧 cache entry 被新结果替换。
- 其他 keyword 搜到的 SkillsMP 结果保留，方便后续 `info/install`。

`registry sync skillsmp` 不做全量同步。SkillsMP API 空 query 会返回 400，而且站点没有明确全量 catalog 语义。P11 对 provider sync 返回错误：

```text
provider registry does not support sync without keyword; use registry search <keyword> --registry skillsmp
```

后续如果需要后台刷新，可以另加 `registry sync skillsmp --query <keyword>`，P11 不做。

## 7. Web 行为

Web Registry 页面复用现有 API，不新增页面：

- `GET /api/registry/skills?keyword=go&registry=skillsmp` 对 provider registry 执行远程搜索并返回结果。
- 搜索结果写入 cache，使现有 `install/plan` 和 `install/run` 可通过 `skillsmp/<skill-id>` 找到 entry。
- keyword 为空时返回 400 风格错误文案，提示输入关键词。
- install plan/run 不直接访问 SkillsMP；它只读取 cache 中的 `SkillEntry`，然后 clone Git repo 中的 `install_entry`。

这保持 Web handler 薄层，不把 provider 细节扩散到 Web。

## 8. 错误处理

- HTTP 非 2xx：返回 `skillsmp search failed: HTTP <status>`。
- JSON 解析失败：返回 `skillsmp search response is invalid`。
- SkillsMP 返回 `success:false`：返回站点错误信息；没有 message 时返回通用错误。
- GitHub tree URL 无法解析：跳过该条，继续处理其他结果；如果全部跳过，返回错误。
- 网络超时：沿用 registry service 当前 HTTP client timeout。

搜索结果中的单条坏数据不应导致整次搜索失败，除非没有任何可安装结果。

## 9. 测试范围

单元测试使用本地 `httptest.Server` 模拟 SkillsMP API：

- `registry.New` 支持 `--provider skillsmp` 对应的 provider registry。
- SkillsMP response 能正确映射 `source_url/source_ref/install_entry/homepage/tags`。
- 搜索 provider registry 会合并写入 cache，并替换同 ID 旧结果。
- 空 keyword 返回错误。
- `registry sync` provider registry 返回“不支持无关键词 sync”的错误。
- CLI `registry add --provider skillsmp` 和 `registry search --registry skillsmp` 的参数路径可用。
- Web `/api/registry/skills` 对 provider registry 能返回 normalized results。

MVP 主链路改动完成后运行：

```bash
go test ./...
```

## 10. 不做事项

- 不做 skills.sh adapter。
- 不做 SkillsLLM adapter。
- 不做 provider interface registry/factory。
- 不做 provider auth、token、rate-limit 配置。
- 不做 provider 全量 sync。
- 不把 SkillsMP `updatedAt` 当版本。
- 不新增独立 Web 页面或第二套安装流程。

## 11. 后续扩展

如果 SkillsMP adapter 验证稳定，下一步再考虑：

- skills.sh：先做搜索展示，安装映射需确认 `source + skillId` 的真实目录规则。
- SkillsLLM：更像 repo/provider discovery，可先做 source result 或 repo result，不急着伪装成单 skill install。
- Provider sync query：`registry sync <id> --query <keyword>`。
- 信任模型：source allowlist、checksum、签名、安装前风险提示。
