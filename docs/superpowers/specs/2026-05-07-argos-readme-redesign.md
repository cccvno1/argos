# Argos README 重新设计

日期: 2026-05-07

## 背景

Argos 的 README 经过多次迭代，存在以下问题：
1. 命令参数不一致（`context` 缺参数、`project add` 不同章节写法不同）
2. 人类叙事和 Agent 命令参考混在一起，结构混乱
3. 部分描述与实际 CLI 行为脱节
4. SKILL.md 和 README 之间的重复内容未对齐

## 设计原则

参照 [superpowers](https://github.com/obra/superpowers) 的文档风格：

- **人类区**：纯叙事性工作流描述，不出现任何命令
- **Agent 区**：完整命令参考，按场景组织，每条命令标注所有参数和必需/可选

## README 结构

```
# Argos
定位一句话

## 工作方式

### 检索知识
叙事描述 Agent 如何检索和引用知识

### 写入知识
叙事描述 design → provenance → publish 的协作流程

### 审查知识
叙事描述 audit/status 如何配合团队 review

## 安装
go install ./cmd/argos

## 初始化
argos init

---

## Agent 操作参考

### 工作流概览
检索 / 写入 / 审查 三者的关系和边界

### 检索知识
find / list / read / cite 命令 + 完整参数

### 写入知识
design → provenance(start/decisions/check/verify) → publish → index 完整链路

### 审查和审计
audit / status / provenance list 命令 + 完整参数

### MCP 集成
tools/list + 工具列表 + 参数说明

### Adapter
install-adapters 用法

### 命令速查表
所有命令 + 参数一览
```

## 同时修改的文件

1. `README.md` — 按上述结构重写
2. `skills/capture-knowledge/SKILL.md` — 确保与 README Agent 区一致
3. `internal/cli/cli.go` — `printUsage` 补充缺失参数
4. 归档 4 个过时 specs 到 `docs/superpowers/specs/archived/`

## 同步更新的关键修复点

1. `argos context --json --project <project>` → 补全 `--phase <phase> --task <task>`
2. `argos project add` → Agent/Internal 参考统一包含 `--tech-domain` 和 `--business-domain`
3. `argos knowledge design` → 补全所有可选参数
4. `argos knowledge find` → 显示 `--task` 和 `--query` 两种用法
5. CLI `printUsage` → 同步修正

## 归档文件列表

移动到 `docs/superpowers/specs/archived/`：

- `2026-05-02-argos-agent-knowledge-authoring-protocol-design.md`
- `2026-05-02-argos-authoring-v2-contract-and-harness-design.md`
- `2026-05-03-argos-author-inspect-authoring-packet-design.md`
- `2026-05-03-argos-authoring-dogfood-productization-design.md`
