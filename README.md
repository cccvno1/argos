# Argos

Argos is a local-first knowledge substrate for AI coding workflows.

## 工作方式

Argos 让你和 AI Agent 协作管理项目知识。你通过对话表达意图，Agent 在后台操作
Argos。

### 检索知识

Agent 在工作前自动检索项目已有的知识。Argos 根据当前任务和代码上下文，判断每一条
知识的匹配程度：

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

每一步决策都记录在来源追踪（provenance）中：谁在哪个环节批准了什么、基于什么
原因。这形成了一条从"我想记录这个"到"知识已就位"的完整证据链。

### 审查知识

Agent 可以汇总所有待审查的知识变更：

- 哪些草稿等待发布决定
- 哪些已发布知识缺少来源记录
- 哪些来源记录与当前内容不一致

这些汇总供你和 PR reviewer 逐条检查。来源追踪提供证据，但不替代人工 review。

## 安装

```bash
go install ./cmd/argos
```

## 初始化

在项目根目录运行 `argos init` 创建知识目录结构。之后所有知识操作由 AI Agent 在
后台完成，你通过对话引导即可。

---

## Agent 操作参考

以下章节覆盖 Agent 使用 Argos 的完整操作。所有 Agent 命令默认使用 `--json` 输出
格式以便程序化解析。

### 检索知识

Agent 在工作流程中自动检索相关知识。检索路径为查找（find）→ 列表（list）→
读取（read）→ 引用（cite）。

#### argos context

获取当前工作流上下文和推荐的操作步骤。

```bash
argos context --json --project <id> --phase <phase> --task <desc> \
  [--files <path>...]
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--project` | 是 | 项目标识符 |
| `--phase` | 是 | 工作流阶段（如 implementation, review, debugging） |
| `--task` | 是 | 当前任务描述 |
| `--files` | 否 | 与当前任务相关的文件路径，可重复 |

#### argos knowledge find

查找与当前工作相关的知识。返回匹配结果、匹配原因和下一步建议，不含完整正文。

```bash
argos knowledge find --json --project <id> \
  --task <desc> | --query <query> \
  [--phase <phase>] [--files <path>...] [--types <type>...] \
  [--tags <tag>...] [--domains <domain>...] [--status <status>...] \
  [--include-deprecated] [--limit <n>]
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--project` | 是 | 项目标识符 |
| `--task` | 条件 | 当前任务描述；与 `--query` 至少提供一个 |
| `--query` | 条件 | 搜索查询；与 `--task` 至少提供一个 |
| `--phase` | 否 | 工作流阶段 |
| `--files` | 否 | 匹配的文件路径，可重复 |
| `--types` | 否 | 包含的知识类型，可重复 |
| `--tags` | 否 | 包含的标签，可重复 |
| `--domains` | 否 | 包含的领域，可重复 |
| `--status` | 否 | 包含的状态，可重复 |
| `--include-deprecated` | 否 | 包含已废弃的知识条目 |
| `--limit` | 否 | 返回结果上限 |

#### argos knowledge list

列出项目知识清单，不含完整正文。

```bash
argos knowledge list --json --project <id> \
  [--domain <domain>] [--types <type>...] [--include-deprecated]
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--project` | 是 | 项目标识符 |
| `--domain` | 否 | 领域过滤 |
| `--types` | 否 | 知识类型过滤，可重复 |
| `--include-deprecated` | 否 | 包含已废弃的知识条目 |

#### argos knowledge read

读取单条知识的完整内容（含正文）。

```bash
argos knowledge read --json <id>
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `<id>` | 是 | 知识条目 ID（位置参数） |

#### argos knowledge cite

生成知识引用，报告缺失的 ID。

```bash
argos knowledge cite --json <id> [<id>...]
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `<id>`... | 是 | 要引用的知识 ID（位置参数，一个或多个） |

---

### 写入知识

完整写入流程包含项目注册、知识设计、来源追踪和发布。

#### 项目注册

```bash
argos project list --json
```

无额外参数。列出所有已注册项目。

```bash
argos project add --id <id> --name <name> --path <path> \
  [--tech-domain <domain>...] [--business-domain <domain>...]
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--id` | 是 | 项目标识符 |
| `--name` | 是 | 项目名称 |
| `--path` | 是 | 项目源码路径 |
| `--tech-domain` | 否 | 技术领域，可重复 |
| `--business-domain` | 否 | 业务领域，可重复 |

#### 写入流程（13 步）

在用户要求记录经验、决策或文档时，按以下步骤操作：

**步骤 1：确认项目已注册**

```bash
argos project list --json
```

如果目标项目不存在，先注册：

```bash
argos project add --id <id> --name <name> --path <path>
```

**步骤 2：设计知识结构**

```bash
argos knowledge design --json --project <id> --intent <intent> \
  [--future-task <desc>] [--phase <phase>] [--query <query>] \
  [--files <path>...] [--domains <domain>...] [--tags <tag>...] \
  [--draft-path <path>]
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--project` | 是 | 项目标识符 |
| `--intent` | 是 | 知识意图（自然语言描述要记录的内容） |
| `--future-task` | 否 | 此知识将支持的未来任务 |
| `--phase` | 否 | 工作流阶段 |
| `--query` | 否 | 相关知识搜索查询 |
| `--files` | 否 | 相关的文件路径，可重复 |
| `--domains` | 否 | 相关领域，可重复 |
| `--tags` | 否 | 相关标签，可重复 |
| `--draft-path` | 否 | 建议的草稿路径 |

**步骤 3：编写知识设计模板**

将 `knowledge design` 返回的 `knowledge_design_template` 写入
`write_guidance.design_path` 指向的文件。

**步骤 4：启动来源追踪**

```bash
argos provenance start --json --design <design.json> --draft <draft-path> \
  [--created-by <agent>]
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--design` | 是 | 知识设计 JSON 文件路径 |
| `--draft` | 是 | 草稿条目或包路径 |
| `--created-by` | 否 | 来源记录创建者标识 |

**步骤 5：记录设计阶段决策**

```bash
argos provenance record-decision --json \
  --provenance <id> --stage design --decision approved \
  --decided-by <actor> --role knowledge_owner --source conversation \
  --reason "<reason>" --recorded-by <agent>
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--provenance` | 是 | 来源记录 ID 或路径 |
| `--stage` | 是 | 决策阶段（此处为 design） |
| `--decision` | 是 | 决策值（approved / changes_requested / rejected） |
| `--decided-by` | 是 | 决策人标识 |
| `--role` | 是 | 决策人角色 |
| `--source` | 是 | 决策来源 |
| `--reason` | 是 | 决策理由 |
| `--recorded-by` | 是 | 决策记录人标识 |

**步骤 6：记录草稿写入决策**

```bash
argos provenance record-decision --json \
  --provenance <id> --stage draft_write --decision approved \
  --decided-by <actor> --role knowledge_owner --source conversation \
  --reason "<reason>" --recorded-by <agent>
```

参数同步骤 5，`--stage` 为 `draft_write`。

**步骤 7：写入草稿**

在设计决策和草稿写入决策均已记录后，将知识内容写入草稿路径。

**步骤 8：执行检查**

```bash
argos provenance record-check --json --provenance <id>
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--provenance` | 是 | 来源记录 ID 或路径 |

**步骤 9：记录发布决策**

```bash
argos provenance record-decision --json \
  --provenance <id> --stage publish --decision approved \
  --decided-by <actor> --role knowledge_owner --source conversation \
  --reason "<reason>" --recorded-by <agent>
```

参数同步骤 5，`--stage` 为 `publish`。

**步骤 10：验证来源记录**

```bash
argos provenance verify --json --provenance <id>
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--provenance` | 是 | 来源记录 ID 或路径 |

**步骤 11：发布知识**

```bash
argos knowledge publish --provenance <id> [--published-by <agent>]
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--provenance` | 是 | 来源记录 ID 或路径 |
| `--published-by` | 否 | 发布人标识 |

**步骤 12：重建索引**

```bash
argos index
```

无额外参数。重新构建本地知识索引。

**步骤 13：验证可发现性**

```bash
argos knowledge find --json --project <id> --task <desc>
```

确认新发布的知识在检索结果中可见。

#### 草稿检查（非流程步骤，可单独执行）

```bash
argos knowledge check --json --design <design.json> --draft <draft>
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--design` | 是 | 知识设计 JSON 文件路径 |
| `--draft` | 是 | 草稿条目或包路径 |

---

### 审查和审计

#### 来源追踪状态

```bash
argos provenance status --json --provenance <id>
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--provenance` | 是 | 来源记录 ID 或路径 |

查看单条来源记录的详细状态：设计哈希、草稿树哈希、检查结果、各阶段决策记录。

#### 来源追踪列表

```bash
argos provenance list --json \
  [--state <state>] [--project <id>] [--knowledge-id <id>]
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--state` | 否 | 状态过滤，默认 `all` |
| `--project` | 否 | 项目 ID 过滤 |
| `--knowledge-id` | 否 | 知识 ID 过滤 |

#### 知识审计

```bash
argos knowledge audit --json \
  [--project <id>] [--include-published]
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--project` | 否 | 项目 ID 过滤 |
| `--include-published` | 否 | 包含已发布且来源记录健康的知识条目 |

汇总所有待审查的知识变更，包括：等待发布决策的草稿、缺少来源记录的已发布知识、
来源记录与当前内容不一致的知识。

---

### 验证

```bash
argos validate [--inbox] [--path <path>]
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--inbox` | 否 | 验证收件箱草稿 |
| `--path` | 否 | 验证单个条目或包路径 |

---

### MCP 集成

启动本地 MCP 服务器：

```bash
argos mcp
```

MCP 服务器通过 stdio 通信，支持 `tools/list` 协议发现。

#### 可用工具

| 工具名 | 说明 |
| --- | --- |
| `argos_context` | 获取工作流上下文和推荐操作步骤 |
| `argos_standards` | 查找项目中活跃的标准 |
| `argos_find_knowledge` | 查找与当前工作相关的知识 |
| `argos_list_knowledge` | 列出项目知识清单 |
| `argos_read_knowledge` | 按 ID 读取知识条目完整内容 |
| `argos_cite_knowledge` | 生成知识引用 |
| `argos_design_knowledge` | 设计持久化知识结构 |
| `argos_check_knowledge` | 对照审阅过的设计检查草稿 |

#### argos_context

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `project` | 是 | 项目标识符 |
| `phase` | 是 | 工作流阶段 |
| `task` | 是 | 当前任务描述 |
| `files` | 否 | 相关文件路径列表 |

#### argos_standards

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `project` | 是 | 项目标识符 |
| `task_type` | 否 | 当前工作类型 |
| `files` | 否 | 相关文件路径列表 |
| `limit` | 否 | 返回结果上限（1-5） |

#### argos_find_knowledge

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `project` | 是 | 项目标识符 |
| `task` | 条件 | 当前任务描述；与 `query` 至少提供一个 |
| `query` | 条件 | 搜索查询；与 `task` 至少提供一个 |
| `phase` | 否 | 工作流阶段 |
| `files` | 否 | 相关文件路径列表 |
| `types` | 否 | 知识类型列表 |
| `tags` | 否 | 标签列表 |
| `domains` | 否 | 领域列表 |
| `status` | 否 | 状态列表 |
| `include_deprecated` | 否 | 包含已废弃条目 |
| `limit` | 否 | 返回结果上限（1-20） |

#### argos_list_knowledge

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `project` | 是 | 项目标识符 |
| `domain` | 否 | 领域过滤 |
| `types` | 否 | 知识类型列表 |
| `include_deprecated` | 否 | 包含已废弃条目 |

#### argos_read_knowledge

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `id` | 是 | 知识条目 ID |

#### argos_cite_knowledge

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `ids` | 是 | 要引用的知识 ID 列表 |

#### argos_design_knowledge

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `project` | 是 | 项目标识符 |
| `intent` | 是 | 知识意图 |
| `future_task` | 否 | 未来该知识应支持的任务 |
| `phase` | 否 | 工作流阶段 |
| `query` | 否 | 相关知识搜索查询 |
| `files` | 否 | 相关文件路径列表 |
| `domains` | 否 | 相关领域列表 |
| `tags` | 否 | 相关标签列表 |
| `draft_path` | 否 | 建议的草稿路径 |

#### argos_check_knowledge

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `design` | 是 | 知识设计 JSON 文件路径 |
| `draft` | 是 | 草稿文件或目录路径 |

---

### Adapter

```bash
argos install-adapters
```

无额外参数。为所有已注册项目安装生成的适配器文件。适配器为仅支持读取项目指令文件
的工具定义最低合约：保留宿主流控制、优先使用 MCP、回退到 CLI JSON 或 Markdown
源文件。

---

### Dogfood（内部验证）

Discovery 验证：

```bash
argos dogfood cases --json
argos dogfood packet --json --case <handle> \
  --workspace <fixture> --argos-binary <binary>
argos dogfood evaluate --json --case <handle> --report <report.md>
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--case` | 是 | Dogfood 案例 ID 或公开句柄 |
| `--workspace` | 是（packet 命令） | 夹具工作区路径 |
| `--argos-binary` | 是（packet 命令） | Argos 二进制路径 |
| `--report` | 是（evaluate 命令） | Markdown 报告路径 |

写入验证：

```bash
argos dogfood write cases --json
argos dogfood write packet --json --case <handle> \
  --workspace <workspace> --argos-binary <binary>
argos dogfood write evaluate --json --case <handle> \
  --report <report.md> --workspace <workspace>
```

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `--json` | 是 | 输出 JSON 格式 |
| `--case` | 是 | 写入验证案例 ID 或公开句柄 |
| `--workspace` | 是 | 工作区路径 |
| `--argos-binary` | 是（packet 命令） | Argos 二进制路径 |
| `--report` | 是（evaluate 命令） | Markdown 报告路径 |

---

### 命令速查

| 命令 | 说明 |
| --- | --- |
| `argos init` | 初始化 Argos 工作区 |
| `argos validate` | 验证知识文件 |
| `argos index` | 重建本地知识索引 |
| `argos install-adapters` | 安装项目适配器文件 |
| `argos context` | 获取工作流上下文和推荐步骤 |
| `argos project list` | 列出已注册项目 |
| `argos project add` | 注册新项目 |
| `argos knowledge design` | 设计知识结构 |
| `argos knowledge check` | 对照设计检查草稿 |
| `argos knowledge find` | 检索知识 |
| `argos knowledge list` | 列出知识清单 |
| `argos knowledge read` | 读取知识完整内容 |
| `argos knowledge cite` | 生成知识引用 |
| `argos knowledge publish` | 发布草稿知识 |
| `argos knowledge audit` | 审计知识变更状态 |
| `argos provenance start` | 启动来源追踪记录 |
| `argos provenance record-decision` | 记录决策 |
| `argos provenance record-check` | 执行来源检查 |
| `argos provenance verify` | 验证来源记录完整性 |
| `argos provenance status` | 查看来源记录状态 |
| `argos provenance list` | 列出所有来源记录 |
| `argos dogfood cases` | 列出 Discovery 验证案例 |
| `argos dogfood packet` | 生成 Discovery 验证包 |
| `argos dogfood evaluate` | 评估 Discovery 验证结果 |
| `argos dogfood write cases` | 列出写入验证案例 |
| `argos dogfood write packet` | 生成写入验证包 |
| `argos dogfood write evaluate` | 评估写入验证结果 |
| `argos mcp` | 启动 MCP 服务器 |
