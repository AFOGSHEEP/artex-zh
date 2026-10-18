# 变更日志

中文 · [English](CHANGELOG.en.md) · [한국어](CHANGELOG.ko.md) · [中文（上游原版）](CHANGELOG.zh.md)

本文档记录 ARTEX 中文版（本仓库）及其直接上游 artex-ko 相对上游 ARTEX 项目的变更，格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

上游 ARTEX 项目的版本发布历史（0.3.x 及更早）与贡献者名单，以原中文完整保留在 [`CHANGELOG.zh.md`](CHANGELOG.zh.md)。各变更的详细内容与依据见仓库提交历史。

## [Unreleased] · 中文版变更（AFOGSHEEP/artex-zh）

### 本地化（i18n）

- **用户可见产出恢复为简体中文。** 在 artex-ko 的实现上反转输出语言：代码固定片段（`agent/prompt.go` 的 `langDirective()`）指示 agent 把所有用户可见产出（漏洞报告、事实摘要、最终总结、聊天回复）用简体中文书写，并保留反镜像条款（不得跟随目标系统/证据的语言）与「技术原文逐字保留」条款（命令、payload、代码、URL、日志/响应原文不翻译）。经过基准的中文推理提示词正文（大脑）保持原样。
- **UI 默认语言切回中文。** `web/src/i18n/config.ts` 的 `DEFAULT_LOCALE` 改为 `zh`；文案继续由 `web/messages/zh.json` 提供，韩文资源 `ko.json` 保留用于与 artex-ko 对照。
- **服务端 API 的用户可见错误与响应恢复为中文。** artex-ko 韩文化的 40+ 个 Go 文件中所有用户可见字符串逐条译回中文（措辞与上游 0.3.x 对齐）；仅作为 agent 大脑输入回灌的文案保持原样以避免基准漂移。聊天命令解析保留韩文前缀（`의도`/`힌트`）以兼容 artex-ko 的使用习惯。
- **前端解析契约同步。** `web/src/components/transcript.tsx` 中解析审批摘要的正则从韩文格式（`도구 X 승인`）同步为中文格式（`工具 X 审批`）；`web/src/app/(main)/system/skills/page.tsx` 判断技能重名 409 的标记从 `이미 존재` 同步为 `技能已存在`。
- **本地化不变量测试全部反转。** artex-ko 新增的 `*_localized_test.go` 断言由「必须含韩文、不得含汉字」反转为「必须含中文、不得残留韩文」；期望值与代码实际输出精确一致（经 git 历史对齐推导 + 人工核对）。
- **构建门禁反转。** `scripts/check-web-cjk.py` 由「静态导出 HTML 零汉字」反转为「零韩文残留」。
- **Mock 演示数据、安装/更新/重置脚本、`config.example.json` 注释、CI 任务名与 Issue 模板全部中文化。**
- **文档体系。** 新的中文 `README.md`（主文档），artex-ko 的韩文 README 保留为 `README.ko.md`；防御指南新增中文版 [`docs/defense-zh.md`](docs/defense-zh.md)，检测规则库新增中文说明 [`detections/README.zh.md`](detections/README.zh.md)。

### 仓库维护

- **一键更新源指向本仓库。** `selfupdate` 的 Release 查询从上游 `Autumn-27/artex`（已 404）改为 `AFOGSHEEP/artex-zh`；在本仓库发布首个 Release 前一键更新会提示「尚未发布任何正式版本」。
- **安装文档适配上游镜像下架。** Docker 快速路径标注不可用，推荐源码编译单二进制。

## [Unreleased] · 韩文版变更（上游 jiwoochris/artex-ko，本仓库继承）

### 本地化（i18n）

- **强制用户可见输出为韩文。** 保持经过基准的 agent 行为指令正文（大脑）原样，用代码固定片段（`langDirective`）指示所有用户可见产出（漏洞报告、事实摘要、最终总结、聊天回复）以韩文书写；命令、payload、代码、日志原文保留。
- **Web UI 韩文化。** 在 Next App Router 引入 `next-intl`，字符串拆分到 `web/messages/ko.json` 与 `web/messages/zh.json`；仪表盘、漏洞、对话、通知发送、拦截、LLM 配置等界面文案全部韩文化。
- **服务端 API 的用户可见错误与响应韩文化。** 浏览器可见的 HTTP 错误/响应文案替换为韩文；作为 agent 大脑输入回灌的文案保持原文以避免基准漂移。
- **文档整理为韩文。** 韩文 `README.md` 与英文 `README.en.md` 并存，原中文保留在 `README.zh.md`。

### 防御·检测资料

- **新增防御·检测指南。** 整理自主 AI 攻击与传统扫描器的差异、防守方可观测的指纹（IoC）、入口与加固、检测规则与事件响应的韩文指南（[`docs/defense-ko.md`](docs/defense-ko.md)）及英文版（[`docs/defense-en.md`](docs/defense-en.md)）。
- **提供可部署的检测规则。** 主机·日志层为 [Sigma](https://sigmahq.io) 原子·关联规则（[`detections/sigma/`](detections/sigma/)），网络层为针对 enrich 探针与 norma SDK WebFetch User-Agent 的 [Suricata](https://suricata.io) 规则（[`detections/suricata/`](detections/suricata/)）。
- **ATT&CK 覆盖可视化。** 规则标注的技术整理为 MITRE ATT&CK Navigator 层（[`detections/attack/`](detections/attack/)）。
- **以标准格式提供机器可读 IoC。** ARTEX 特有指纹汇总为 CSV（[`detections/indicators/artex_indicators.csv`](detections/indicators/artex_indicators.csv)）与可直接导入威胁情报平台的 MISP 事件（[`detections/indicators/artex_indicators.misp.json`](detections/indicators/artex_indicators.misp.json)）。
- **配套可复现的检测测试。** 八类测试（Sigma 结构与编译校验、Sigma 实时事件匹配、后端可移植性、SigmaHQ 惯例 lint、Suricata 加载与触发、ATT&CK 层一致性、指标-源一致、MISP 导出 ↔ CSV 同步）、一键批跑 runner 与 pre-commit 示例，并接入 CI 合并门禁。Sigma 实时事件匹配同时验证规则对恶意样本事件实际告警、对正常事件保持静默。

### 仓库维护

- **加入安全·滥用警告与当地法律告知。**（韩国《信息通信网法》《个人信息保护法》）
- **以韩文 UI 截图更新了界面预览。**
- **维护者 Runbook 与贡献指南。** 上游同步与翻译漂移防护的 [`MAINTAINING.md`](MAINTAINING.md)、检测规则贡献契约 [`CONTRIBUTING.md`](CONTRIBUTING.md)。
- **推送·PR 合并门禁 CI。** 所有推送与 PR 运行 Go 构建、`go vet`、单元测试（[`ci.yml`](.github/workflows/ci.yml)）、UI 静态构建（[`web.yml`](.github/workflows/web.yml)）、文档站内链接与图片引用完整性（[`docs.yml`](.github/workflows/docs.yml)）；需要数据库的集成测试按包隔离配套 PostgreSQL 服务验证。文档链接检查用不依赖外网的确定性脚本（[`scripts/check-doc-links.py`](scripts/check-doc-links.py)），并按 GitHub slug 规则核对锚点链接。
- **外链存活周期巡检。** 非阻塞工作流（[`external-links.yml`](.github/workflows/external-links.yml)）每周一及手动触发运行（[`scripts/check-external-links.py`](scripts/check-external-links.py)）。
- **贡献与治理基础设施。** Bug/功能/翻译 Issue 模板、PR 模板、安全漏洞报告政策（[`SECURITY.md`](SECURITY.md)）、行为准则（[`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md)）。
- **完整的英文文档层。** `README.en.md`、防御指南、变更日志、安全政策、行为准则、贡献指南、维护 Runbook、流量证据设计文档及英文 Issue 模板；中英文档在页眉互相链接。

---

上游 ARTEX 项目的版本发布历史与贡献者名单见 [`CHANGELOG.zh.md`](CHANGELOG.zh.md)（原中文保留）。
