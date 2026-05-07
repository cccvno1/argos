# Argos README 重新设计 + 文档统一 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 统一 Argos 活跃文档（README + SKILL.md + CLI 帮助），修复所有参数不一致问题，归档过时 specs。

**Architecture:** 人类区用叙事性工作流描述，Agent 区放完整命令参考。每个命令标注全部参数和必需/可选。参照 superpowers 的文档风格。

**Tech Stack:** Markdown, Go (cli.go 小修改), git mv 归档

---

## 文件结构

- **重写**: `README.md` — 全文替换为新结构
- **修改**: `skills/capture-knowledge/SKILL.md` — 对齐 README Agent 区
- **修改**: `internal/cli/cli.go` — `printUsage` 补全参数
- **归档**: `docs/superpowers/specs/` 下 4 个过时文件 → `archived/`
- **新增**: `docs/superpowers/specs/archived/` 目录

---

### Task 1: 归档过时 Specs

**Files:**
- Create: `docs/superpowers/specs/archived/` (directory)
- Move: 4 files

- [ ] **Step 1: 创建归档目录并移动文件**

```bash
mkdir -p docs/superpowers/specs/archived
git mv docs/superpowers/specs/2026-05-02-argos-agent-knowledge-authoring-protocol-design.md docs/superpowers/specs/archived/
git mv docs/superpowers/specs/2026-05-02-argos-authoring-v2-contract-and-harness-design.md docs/superpowers/specs/archived/
git mv docs/superpowers/specs/2026-05-03-argos-author-inspect-authoring-packet-design.md docs/superpowers/specs/archived/
git mv docs/superpowers/specs/2026-05-03-argos-authoring-dogfood-productization-design.md docs/superpowers/specs/archived/
```

- [ ] **Step 2: 提交归档**

```bash
git add docs/superpowers/specs/archived/ docs/superpowers/specs/
git commit -m "docs: archive obsolete authoring specs"
```

---

### Task 2: 修复 CLI printUsage 参数缺失

**Files:**
- Modify: `internal/cli/cli.go`

- [ ] **Step 1: 更新 printUsage 中的命令示例**

在 `internal/cli/cli.go` 的 `printUsage` 函数中，找到命令示例部分，修改以下行：

将:
```go
fmt.Fprintln(w, "  argos project add --id <project> --name <name> --path <path>")
```
改为:
```go
fmt.Fprintln(w, "  argos project add --id <project> --name <name> --path <path> --tech-domain <domain> --business-domain <domain>")
```

- [ ] **Step 2: 运行测试确认无破坏**

```bash
go test ./internal/cli -count=1
```

Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add internal/cli/cli.go
git commit -m "fix: complete project add parameters in cli usage"
```

---

### Task 3: 重写 README.md — 人类区

**Files:**
- Rewrite: `README.md`

- [ ] **Step 1: 写入新 README**

完整写入以下内容到 `README.md`：

```markdown
# Argos

Argos is a local-first knowledge substrate for AI coding workflows.

## 工作方式

Argos 让你和 AI Agent 协作管理项目知识。你通过对话表达意图，Agent 在后台操作 Argos。

### 检索知识

Agent 在工作前自动检索项目已有的知识。Argos 根据当前任务和代码上下文，
判断每一条知识的匹配程度：

- **strong**：强烈建议 Agent 读取并使用这条知识
- **partial**：有相关内容，但存在缺口，Agent 应谨慎参考
- **weak**：仅作参考，不构成强指导
- **none**：无相关知识，Agent 不会引用 Argos

Agent 在工作结束后，会引用所使用知识的 ID，方便追溯。

### 写入知识

当你要求记录经验、决策或文档时：

1. Agent 检查已有知识，设计知识结构
2. Agent 将知识设计呈现给你，由你确认或修改
3. 你批准设计后，Agent 写入草稿
4. 你再次确认后，Agent 发布知识并重建索引

每一步决策都记录在来源追踪（provenance）中：谁在哪个环节批准了什么、
基于什么原因。这形成了一条从"我想记录这个"到"知识已就位"的完整证据链。

### 审查知识

Agent 可以汇总所有待审查的知识变更：
- 哪些草稿等待发布决定
- 哪些已发布知识缺少来源记录
- 哪些来源记录与当前内容不一致

这些汇总供你和 PR reviewer 逐条检查。来源追踪提供证据，
但不替代人工 review。

## 安装

```bash
go install ./cmd/argos
```

## 初始化

在项目根目录运行：

```bash
argos init
```

这会在项目中创建知识目录结构。之后所有知识操作由 AI Agent 在后台完成，
你通过对话引导即可。

---

## Agent 操作参考
```

- [ ] **Step 2: 提交人类区**

```bash
git add README.md
git commit -m "docs: rewrite README human section with narrative workflow"
```

---

### Task 4: 重写 README.md — Agent 检索区

**Files:**
- Modify: `README.md` (追加内容)

- [ ] **Step 1: 追加检索知识章节到 README**

在 README.md 末尾追加：

```markdown
### 检索知识

Agent 使用以下命令检索项目知识。所有命令搭配 `--json` 获取结构化输出。

#### 查找知识

```bash
argos knowledge find --json --project <project> --task <task> --query <query>
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `--json` | 是 | JSON 输出 |
| `--project` | 是 | 项目标识 |
| `--task` | 否* | 当前任务描述（与 `--query` 至少一个） |
| `--query` | 否* | 搜索查询（与 `--task` 至少一个） |
| `--phase` | 否 | 工作阶段 |
| `--files` | 否 | 相关文件路径 |
| `--types` | 否 | 知识类型过滤 |
| `--tags` | 否 | 标签过滤 |
| `--domains` | 否 | 领域过滤 |
| `--status` | 否 | 状态过滤 |
| `--include-deprecated` | 否 | 包含已废弃知识 |
| `--limit` | 否 | 返回条数上限 |

返回 `support.level`（strong/partial/weak/none）、`why_matched`、`usage`、`search_status`、
`missing_needs`、`next_steps`。不返回完整正文。使用 `knowledge read` 读取正文。

#### 列出知识清单

```bash
argos knowledge list --json --project <project>
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `--json` | 是 | JSON 输出 |
| `--project` | 否 | 项目标识 |
| `--domain` | 否 | 领域过滤 |
| `--types` | 否 | 类型过滤 |
| `--include-deprecated` | 否 | 包含已废弃知识 |

返回知识清单和方向指引，不返回正文。

#### 读取知识正文

```bash
argos knowledge read --json <id>
```

获取指定知识 ID 的完整正文和元数据。使用 `find` 或 `list` 先获取 ID。

#### 引用知识

```bash
argos knowledge cite --json <id>...
```

返回引用元数据，报告缺失的 ID。

#### 获取上下文和标准

```bash
argos context --json --project <project> --phase <phase> --task <task>
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `--json` | 是 | JSON 输出 |
| `--project` | 是 | 项目标识 |
| `--phase` | 是 | 工作阶段 |
| `--task` | 是 | 当前任务描述 |

返回项目上下文和工作标准，**不含**正文。阅读推荐知识用 `knowledge read`。
```

- [ ] **Step 2: 提交检索区**

```bash
git add README.md
git commit -m "docs: add agent retrieval reference to README"
```

---

### Task 5: 重写 README.md — Agent 写入区

**Files:**
- Modify: `README.md` (追加内容)

- [ ] **Step 1: 追加写入知识章节**

在 README.md 末尾追加：

```markdown
### 写入知识

完整写入流程：design → provenance → publish → index → findback

#### 项目注册

写入项目相关知识的先决步骤：

```bash
argos project list --json
argos project add --id <project> --name <name> --path <path> --tech-domain <domain> --business-domain <domain>
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `--id` | 是 | 项目标识 |
| `--name` | 是 | 项目名称 |
| `--path` | 是 | 项目路径 |
| `--tech-domain` | 否 | 技术领域（可重复） |
| `--business-domain` | 否 | 业务领域（可重复） |

如果目标项目不存在，`knowledge check` 会返回 `review-needed`。

#### 知识设计

```bash
argos knowledge design --json --project <project> --intent <intent>
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `--json` | 是 | JSON 输出 |
| `--project` | 是 | 项目标识 |
| `--intent` | 是 | 用户的知识意图 |
| `--future-task` | 否 | 知识未来的使用场景 |
| `--phase` | 否 | 当前工作阶段 |
| `--query` | 否 | 辅助搜索词 |
| `--files` | 否 | 相关文件 |
| `--domains` | 否 | 领域 |
| `--tags` | 否 | 标签 |
| `--draft-path` | 否 | 草稿路径 |

返回 `write_guidance` 和 `knowledge_design_template`。将 template 写入
`write_guidance.design_path`，再根据设计写入草稿。

#### 设计校验

```bash
argos knowledge check --json --design <design.json> --draft <draft-path>
```

检查草稿是否与设计一致。返回 `result: pass` 或问题清单。

#### 来源追踪

在写入前必须建立来源追踪：

```bash
argos provenance start --json --design <design.json> --draft <draft-path>
```

返回 `provenance_id`，后续步骤需要此 ID。

```bash
argos provenance record-decision --json --provenance <id> --stage design --decision approved --decided-by <actor> --role knowledge_owner --source conversation --reason "<reason>" --recorded-by <agent>

argos provenance record-decision --json --provenance <id> --stage draft_write --decision approved --decided-by <actor> --role knowledge_owner --source conversation --reason "<reason>" --recorded-by <agent>

argos provenance record-check --json --provenance <id>

argos provenance record-decision --json --provenance <id> --stage publish --decision approved --decided-by <actor> --role knowledge_owner --source conversation --reason "<reason>" --recorded-by <agent>

argos provenance verify --json --provenance <id>
```

| record-decision 参数 | 必需 | 说明 |
|------|------|------|
| `--provenance` | 是 | 来源记录 ID |
| `--stage` | 是 | 阶段：design / draft_write / publish |
| `--decision` | 是 | 决策：approved / changes_requested / rejected |
| `--decided-by` | 是 | 决策者 |
| `--role` | 是 | 决策者角色 |
| `--source` | 是 | 决策来源 |
| `--reason` | 是 | 决策理由 |
| `--recorded-by` | 是 | 记录者 Agent |

`verify` 返回 `result: pass` 时，可以发布。

#### 发布

```bash
argos knowledge publish --provenance <id>
```

将草稿发布为正式知识。发布后草稿中的 `status: draft` 转为 `status: active`。

#### 重建索引

```bash
argos index
```

发布后必须重建索引，使新知识可检索。

#### 验证发布

```bash
argos knowledge find --json --project <project> --query "<query>"
```

确认新知识可被检索到。

#### 完整 13 步写入流程

```
1. argos project list --json
2. argos project add --id <project> --name <name> --path <path> (如缺)
3. argos knowledge design --json --project <project> --intent "<intent>"
4. 根据 design 输出写入 design.json 和草稿文件
5. argos provenance start --json --design <design.json> --draft <draft-path>
6. argos provenance record-decision --stage design --decision approved
7. argos provenance record-decision --stage draft_write --decision approved
8. argos provenance record-check --json --provenance <id>
9. argos provenance record-decision --stage publish --decision approved
10. argos provenance verify --json --provenance <id>
11. argos knowledge publish --provenance <id>
12. argos index
13. argos knowledge find --json --project <project> --query "<query>"
```
```

- [ ] **Step 2: 提交写入区**

```bash
git add README.md
git commit -m "docs: add agent write reference to README"
```

---

### Task 6: 重写 README.md — Agent 审查 + MCP + Adapter + 命令速查

**Files:**
- Modify: `README.md` (追加内容)

- [ ] **Step 1: 追加审查、MCP、Adapter、命令速查章节**

在 README.md 末尾追加：

```markdown
### 审查和审计

#### 查看来源状态

```bash
argos provenance status --json --provenance <id>
```

单个来源记录的完整状态：各阶段决策是否齐全、检查是否通过、
哪些环节需要补充。

#### 列出来源记录

```bash
argos provenance list --json --state <state> --project <project> --knowledge-id <id>
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `--json` | 是 | JSON 输出 |
| `--state` | 否 | draft / published / all（默认 all） |
| `--project` | 否 | 项目过滤 |
| `--knowledge-id` | 否 | 知识 ID 过滤 |

#### 知识审计

```bash
argos knowledge audit --json --project <project> --include-published
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `--json` | 是 | JSON 输出 |
| `--project` | 否 | 项目过滤 |
| `--include-published` | 否 | 包含已发布的健康记录 |

返回汇总：`summary.open`、`summary.ready_to_publish`、`summary.blocked`、`summary.problems`、
`summary.published`、`summary.official_missing_provenance`。每条问题记录包含
category、severity、action 建议。

Audit 和 status 提供证据组织，不替代 PR review。发布前请运行 audit，
配合人工或 PR 审查逐条检查。

### MCP 集成

启动本地 MCP 服务器：

```bash
argos mcp
```

通过 `tools/list` 发现工具，`tools/call` 调用。

#### MCP 工具列表

| 工具 | 用途 |
|------|------|
| `argos_context` | 获取项目上下文和工作标准（不含正文） |
| `argos_standards` | 获取项目标准 |
| `argos_find_knowledge` | 查找知识，返回匹配度、解释和下一步建议（不含正文） |
| `argos_list_knowledge` | 列出知识清单（不含正文） |
| `argos_read_knowledge` | 读取完整知识正文 |
| `argos_cite_knowledge` | 引用知识，返回引用元数据 |
| `argos_design_knowledge` | 返回写入指导和设计模板 |
| `argos_check_knowledge` | 检查草稿与设计的一致性 |

所有检索类工具在运行前需 `argos index`。

### Adapter

为不支持 MCP 的 Agent 工具生成适配器文件：

```bash
argos install-adapters
```

生成的适配器保留宿主工作流控制权，优先使用 MCP，回退到 CLI JSON 或 Markdown。

### 命令速查

| 命令 | 用途 |
|------|------|
| `argos init` | 初始化知识目录 |
| `argos validate [--inbox] [--path <path>]` | 验证知识格式 |
| `argos index` | 重建检索索引 |
| `argos project add --id <id> --name <name> --path <path> --tech-domain <d> --business-domain <d>` | 注册项目 |
| `argos project list --json` | 列出项目 |
| `argos context --json --project <p> --phase <p> --task <t>` | 获取上下文 |
| `argos knowledge design --json --project <p> --intent <i>` | 设计知识 |
| `argos knowledge check --json --design <d> --draft <d>` | 校验草稿 |
| `argos knowledge publish --provenance <id>` | 发布知识 |
| `argos knowledge find --json --project <p> --task <t> --query <q>` | 查找知识 |
| `argos knowledge list --json --project <p> --domain <d>` | 列出知识清单 |
| `argos knowledge read --json <id>` | 读取知识正文 |
| `argos knowledge cite --json <id>...` | 引用知识 |
| `argos knowledge audit --json --project <p>` | 知识审计 |
| `argos provenance start --json --design <d> --draft <d>` | 创建来源记录 |
| `argos provenance record-decision --json --provenance <id> --stage <s> --decision <d> --decided-by <a> --role <r> --source <s> --reason <r> --recorded-by <a>` | 记录决策 |
| `argos provenance record-check --json --provenance <id>` | 记录校验结果 |
| `argos provenance verify --json --provenance <id>` | 验证来源完整性 |
| `argos provenance list --json --state <s> --project <p>` | 列出来源记录 |
| `argos provenance status --json --provenance <id>` | 查看来源状态 |
| `argos install-adapters` | 生成适配器文件 |
| `argos mcp` | 启动 MCP 服务器 |
| `argos dogfood cases --json` | 列出 dogfood 用例 |
| `argos dogfood packet --case <c> --workspace <w> --argos-binary <a>` | 生成 dogfood 测试包 |
| `argos dogfood evaluate --case <c> --report <r> --json` | 评估 dogfood 报告 |
| `argos dogfood write cases --json` | 列出写入 dogfood 用例 |
| `argos dogfood write packet --case <c> --workspace <w> --argos-binary <a>` | 生成写入 dogfood 测试包 |
| `argos dogfood write evaluate --case <c> --report <r> --workspace <w> --json` | 评估写入 dogfood 报告 |
```

- [ ] **Step 2: 提交审查/MCP/Adapter/速查区**

```bash
git add README.md
git commit -m "docs: add review, MCP, adapter, and command reference to README"
```

---

### Task 7: 对齐 SKILL.md

**Files:**
- Modify: `skills/capture-knowledge/SKILL.md`

- [ ] **Step 1: 检查 SKILL.md 与 README Agent 区的一致性**

SKILL.md 中需要确认的内容：

1. 命令名称和参数与 README Agent 区一致
2. 写入流程步骤顺序与 README "完整 13 步写入流程"一致
3. provenance 三阶段（design / draft_write / publish）决策顺序正确
4. `argos project add` 包含 `--tech-domain` 和 `--business-domain` 参数

- [ ] **Step 2: 修复发现的不一致**

根据检查结果，使用 Edit 工具修正 SKILL.md 中的不一致项。

- [ ] **Step 3: 提交**

```bash
git add skills/capture-knowledge/SKILL.md
git commit -m "docs: align capture skill with new README reference"
```

---

### Task 8: 最终验证

**Files:**
- No new files

- [ ] **Step 1: 运行全量测试**

```bash
go test ./... -count=1
```

Expected: all packages PASS.

- [ ] **Step 2: CLI 帮助输出验证**

```bash
go run ./cmd/argos
```

Expected: 帮助中包含完整的 `project add` 参数。

- [ ] **Step 3: 提交最终验证结果**

```bash
git add -A
git commit -m "docs: finalize README redesign and documentation unification"
```

---

## 最终审查门

1. 请求 code review: `superpowers:requesting-code-review`
2. 修复 Critical 和 Important 问题
3. 重跑 `go test ./... -count=1`
4. 使用 `superpowers:finishing-a-development-branch` 决定合并方式
