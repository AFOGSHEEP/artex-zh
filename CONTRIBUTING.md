# 贡献指南 (Contributing)

中文 · [English](CONTRIBUTING.en.md)

感谢你关注 ARTEX 中文版（`artex-zh`）。本文档整理开始贡献前需要了解的范围、方针与流程。提交贡献前请务必先读[使用范围与法律责任](#使用范围与法律责任)和[本地化方针](#本地化方针)。

- 报告 Bug 或建议功能 → 使用 [issue 模板](https://github.com/AFOGSHEEP/artex-zh/issues/new/choose)。
- 发现翻译/本地化问题 → 使用「翻译 / 本地化问题」issue 模板。
- 发现安全漏洞 → **不要提公开 issue**，按 [SECURITY.md](SECURITY.md) 的流程报告。
- 所有参与者须遵守[行为准则（CODE_OF_CONDUCT.md）](CODE_OF_CONDUCT.md)。

---

## 使用范围与法律责任

ARTEX 是 LLM 多智能体**自主**执行渗透测试的攻击性安全工具。贡献者与用户受同样的范围约束：

- 验证代码时，只在**自己拥有或获得书面明确授权的目标**、或**本地隔离环境**（如 Docker 里的 OWASP Juice Shop、DVWA 等刻意构造且本人拥有的漏洞靶场）上运行工具。
- 不接受对授权范围外的真实/生产/远程系统执行扫描、探测、利用的代码，或助长此类使用的变更。
- 在中国大陆，未经授权侵入或干扰他人计算机信息系统的行为违反《刑法》相关条款与《网络安全法》；涉及个人信息的适用《个人信息保护法》。详见 [README](README.md#-安全与滥用警告请务必先读)。

贡献的代码与文档被如何使用的法律责任，由执行它的用户本人承担。本仓库按「现状（AS IS）」提供。

---

## 本地化方针

本仓库（中文版）的直接上游是 [jiwoochris/artex-ko](https://github.com/jiwoochris/artex-ko)，再上游是原版 [Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX)。中文版的定位是：**在保持 agent 判断性能原样的同时，把用户可见的产出全部恢复为简体中文**。偏离这一方针的翻译贡献可能损害性能，不予接受。

- **不要翻译 agent 的内部推理提示词（行为指令正文）。** 必须保持经过基准的原文（中文）；该正文在 `agent/promptcatalog.go` 与数据库种子（`agent_prompts`）里。翻译会给 agent 判断带来漂移。
- **只强制用户可见产出使用简体中文。** 检测结果（`report_finding`）、事实摘要（`record_fact`）、最终报告、对话回复属于此类。该强制由 `agent/prompt.go` 的 `langDirective()` 代码固定尾巴附加到每个角色的 system 提示词末尾；要改输出语言就改这个函数（并同步 `agent/prompt_test.go` 的不变量断言）。
- **命令、payload、代码、URL、日志原文一律不翻译。** 分析需要原文，保持原样。
- **韩文资源保留作对照。** UI 字符串的韩文版保留在 `web/messages/ko.json`，用于与 artex-ko 对照；中文在 `web/messages/zh.json`（默认 locale 为 `zh`）。
- 新增 UI 字符串时不要硬编码，请添加到消息文件的 key。
- **输出语言强制是提示词引导而非硬上限。** `langDirective()` 只是指示输出语言为简体中文；忠实度取决于模型能力、角色与上下文。验证本地化变更请使用能力足够的模型（前沿级）——小型低成本模型可能把报告/摘要回退成其他语言，不要仅凭小模型的输出判断翻译是否生效。OpenAI 系模型的 `max_tokens` 陷阱见 [README 的「模型选择与输出语言」](README.md#模型选择与输出语言)。
- **追赶上游变更的流程见维护者文档。** 上游（artex-ko / 原版 ARTEX）更新时，区分保留资产与翻译对象、检查翻译对称与漂移的 runbook 在 [MAINTAINING.md](MAINTAINING.md)。

---

## 开发环境

项目由 **Go 后端**（单二进制内嵌前端）+ **Next.js 前端**组成。

主要功能的设计意图整理在 `docs/` 的设计文档中。涉及漏洞与流量证据关联的功能（报告 agent 的自动绑定、`report_finding` 的 `traffic_refs` 等）时，请先读[漏洞多流量证据设计文档](docs/finding-traffic-evidence-zh.md)。

### 版本要求

- Go 1.26+（按 `go.mod`）
- Node.js 22+（按 release 工作流）
- Docker 与 Docker Compose（本地运行/验证）

### 后端 (Go)

本地装有 Go 时，在仓库根目录执行：

```bash
go build ./...
go vet ./agent/
go test ./agent/
```

没有 Go 也可以用 Docker 等价验证；模块/构建缓存放 named volume 可加速重复运行：

```bash
docker run --rm -v "$PWD":/src -w /src \
  -v artexzh-gomod:/go/pkg/mod -v artexzh-gocache:/root/.cache/go-build \
  golang:1.26 sh -c 'go build ./... && go vet ./agent/ && go test ./agent/'
```

#### DB 集成测试（需要 postgres）

上面的 `go test ./agent/` 会**静默跳过**需要连接 PostgreSQL 的 **DB 集成测试**。`agent`、`config`、`db`、`evidence`、`llmrec`、`server` 六个包里有需要真实数据库的测试；当既没有环境变量 `ARTEX_PG_DSN`、配置文件也没有 `database` 项时，这些测试以 `--- SKIP` 跳过、包以 `ok` 结束。因此改了这六个包却无 DSN 验证时，可能出现**本地通过（ok）但 PR 的 `go-db` 任务失败**。

本地跑这些测试需启动 PostgreSQL 并传入 `ARTEX_PG_DSN`。下面用与 CI 相同的 `postgres:16-alpine` 在隔离网络中运行，并复用上面的 named volume：

```bash
# 1) 起隔离网络与空 postgres（与 CI 相同的镜像与账号）。
docker network create artexzh-db 2>/dev/null || true
docker run -d --name artexzh-pg --network artexzh-db \
  -e POSTGRES_USER=artex -e POSTGRES_PASSWORD=artex -e POSTGRES_DB=artex \
  postgres:16-alpine
until docker exec artexzh-pg pg_isready -U artex -d artex >/dev/null 2>&1; do sleep 1; done

# 2) 传 DSN 跑 DB 集成包（DSN 的 host 是容器名）。
#    只跑改动的包时，把 ./agent/ 换成 config/db/evidence/llmrec/server。
docker run --rm --network artexzh-db -v "$PWD":/src -w /src \
  -v artexzh-gomod:/go/pkg/mod -v artexzh-gocache:/root/.cache/go-build \
  -e ARTEX_PG_DSN='postgres://artex:artex@artexzh-pg:5432/artex?sslmode=disable' \
  golang:1.26 sh -c 'go test ./agent/ -count=1'

# 3) 清理。
docker rm -f artexzh-pg && docker network rm artexzh-db
```

CI 的 `go-db` 任务会把这六个包**各自隔离在独立的 postgres 上**、合并前强制运行（`.github/workflows/ci.yml`）。改了 DB 集成包，建议 PR 前按上述方式直接验证。

### 前端 (web)

```bash
cd web
npm ci
npm run dev          # 开发服务器
npm run build        # 生产构建
npm run build:static # 静态导出构建（合并门禁 · 含 TypeScript 类型检查）
npm run check        # Biome lint/格式检查（信息用 · 预存欠账，暂非门禁）
npm run check:fix    # 自动修复
```

提交前的格式与 lint 由 Biome 管理：`lint-staged` 会对暂存文件自动执行 `biome check --write`。

### 整体运行（Docker Compose）

```bash
cp .env.example .env     # 设置 POSTGRES_PASSWORD
docker compose up -d postgres   # 上游 artex 镜像已下架，仅启动数据库；后端按 README 源码编译
```

---

## 贡献流程

1. 先**开 issue**。大改动先对齐方向；小修（错字、链接、明显 bug）可直接 PR。
2. **Fork** 仓库并建主题分支：`feat/...`、`fix/...`、`docs/...`、`i18n/...`。
3. 完成变更并**亲自运行对应范围的验证**。Go 变更须通过上面的 `build`/`vet`/`test`；web 变更须通过 `npm run build:static`（合并门禁，含 TypeScript 类型检查）。`npm run check`（Biome）因上游遗留 lint 欠账暂非门禁，`web.yml` 只作信息步骤——不必全绿，但**不要引入新错误**（提交时 `lint-staged` 会自动对暂存文件执行 `biome check --write`）。改了文档（`.md`）请运行 `python3 -I scripts/check-doc-links.py`，检查仓库内链接、图片引用与锚点（`#标题`）链接没有损坏；锚点按 GitHub 规则从标题生成 slug 比对——只改标题不修指向它的锚点链接会在这里被拦下（CI 的 `docs` 工作流以合并门禁强制同一检查）。该检查也在仓库根的 [`.pre-commit-config.yaml`](.pre-commit-config.yaml) 中作为 `docs` 钩子，`pre-commit install` 后提交时自动运行（只用 Python 标准库、不联网，无需 Docker）。
4. **提 PR**：按 [PR 模板](.github/PULL_REQUEST_TEMPLATE.md)写清改了什么、为什么、如何验证；改了 UI 附截图。
5. 用户可见的变更（功能、本地化、文档、检测规则等）在[变更日志（CHANGELOG.md）](CHANGELOG.md)的 `[Unreleased]` 节加一行；纯内部重构或仅测试的变更可省略。

### 提交信息

沿用现有提交历史的惯例：`type(scope): 描述`，描述用中文。

- `type`：`feat` · `fix` · `docs` · `chore` · `refactor` · `test` · `i18n` 等
- `scope`：变更区域（`agent` · `web` · `server` 等），可省略

示例：

```
feat(agent): 用户可见输出强制简体中文 (langDirective)
docs: 撰写中文 README，韩文原版保留为 README.ko.md
i18n(web): 仪表盘导航标签中文文案
```

---

## 检测规则·检测测试贡献

本仓库在 [`detections/`](detections/) 中维护用于**防御·检测** ARTEX 类自主 AI 攻击的规则：可部署的 [Sigma](https://sigmahq.io) 规则（[`detections/sigma/`](detections/sigma/)）、网络侧 [Suricata](https://suricata.io) 规则（[`detections/suricata/`](detections/suricata/)）、[MITRE ATT&CK](https://attack.mitre.org/) 覆盖层（[`detections/attack/`](detections/attack/)），以及可复现地证明这些规则确实触发的测试（[`detections/tests/`](detections/tests/)）。新增或修改检测规则请遵守以下契约；八个测试套件把其中大部分机械化强制，只改规则不更新测试/层会导致测试失败。

- **所有指标锚定在可观测事实上。** 规则使用的字符串、User-Agent、行为阈值必须能从本仓库源码实际确认，不做推测；规则内写明依据的源文件（如 `artex-enrich/1.0` 指标来自 `enrich/enrich.go`）。指标一致性测试（[`detections/tests/indicators/`](detections/tests/indicators/)）检查每个指标在源码与规则两侧仍然存在——上游再同步改了源字符串就必须同步改规则，否则测试失败。改动机器可读指标清单（[`detections/indicators/artex_indicators.csv`](detections/indicators/artex_indicators.csv)）时，须同步更新 MISP 事件（[`detections/indicators/artex_indicators.misp.json`](detections/indicators/artex_indicators.misp.json)）；MISP 导出测试（[`detections/tests/misp/`](detections/tests/misp/)）强制两文件逐行一致且为 pymisp 可加载的合法 MISP 文档。
- **诚实写明局限。** Sigma 规则写在 `description`、Suricata 规则写在注释里：说明规则抓不到的情形与误报可能。若是通用狩猎线索而非 ARTEX 专属特征（如破坏性命令），请明确标注，避免单次命中就把攻击者断言为 ARTEX。
- **通过静态校验。** Sigma 规则以 SigmaHQ 校验器基线 0 issue 通过（`sigma check --validation-config detections/tests/sigma_lint/validators.yml`）。普通 `sigma check` 只跑 pySigma 核心校验器；标题大小写、字段/logsource 分类、参考链接等 SigmaHQ 惯例只有该基线会拦截。四项例外是不适用于独立规则集的 SigmaHQ monorepo 惯例，理由记录在 [`detections/tests/sigma_lint/validators.yml`](detections/tests/sigma_lint/validators.yml)。Suricata 规则须能被 `suricata -T` 干净加载。
- **附带可复现测试。** 在 [`detections/tests/`](detections/tests/) 下用测试证明规则会触发（或结构有效）：输入每次确定性生成、不把二进制入库；与引擎版本无关的属性（触发存在、无误报）精确断言；随版本波动的数值按下限断言并另行记录基准。新增/修改 Sigma 关联规则时，后端可移植性测试（[`detections/tests/sigma_backends/`](detections/tests/sigma_backends/)）会验证其在多个后端可转换——请与 [`detections/README.zh.md`](detections/README.zh.md) 的后端支持说明保持一致。
- **同步更新 ATT&CK 层。** 增改规则的 `attack.*` 标签时，同步更新 [`detections/attack/artex_navigator_layer.json`](detections/attack/artex_navigator_layer.json) 的技术与评分；一致性测试强制规则↔层双向一致，只改一边即失败。
- **不收录可被读作攻击指南的内容。** 本仓库的检测资料只保持防御·检测定位；执行利用的方法、绕过检测的技巧等攻击性叙述不予接受。

八个测试套件有 Docker 即可运行，产物不入库。任一脚本在任一断言失败时以非零码退出，可直接接入 CI 或 pre-commit：

```bash
detections/tests/sigma/run.sh           # Sigma：sigma check + 后端转换 + 指标存活
detections/tests/sigma_match/run.sh     # Sigma：原子规则对恶意样本触发、正常样本静默
detections/tests/sigma_lint/run.sh      # Sigma：SigmaHQ 全量惯例校验器 + 文档化基线
detections/tests/sigma_backends/run.sh  # Sigma 可移植性：关联规则在多后端可转换
detections/tests/suricata/run.sh        # Suricata：合成 pcap → suricata -r → 断言告警数
detections/tests/attack/run.sh          # ATT&CK：层 ↔ 规则双向一致
detections/tests/indicators/run.sh      # 指标：规则锚定指标 ↔ 上游源码双向一致
detections/tests/misp/run.sh            # MISP：指标 CSV ↔ MISP 事件同步 + pymisp 有效性
```

一键运行八个套件用 [`detections/tests/run-all.sh`](detections/tests/run-all.sh)：按 CI 顺序依次执行，前面的套件失败也跑完全部，然后输出各套件 PASS/FAIL 摘要，任一失败即非零退出。把它直接挂成 pre-commit 钩子的示例配置在仓库根的 [`.pre-commit-config.yaml`](.pre-commit-config.yaml)：`pip install pre-commit && pre-commit install` 后，只有在改动检测规则或规则锚定的上游源码的提交上（与 CI 相同范围）运行，在推送前拦住规则/测试不一致。同一配置文件还包含检查文档内链、图片与锚点的 `docs` 钩子（即贡献流程第 3 步的 `check-doc-links.py`）。

这八个测试由仓库 CI（[`.github/workflows/detections.yml`](.github/workflows/detections.yml)）在每次 `detections/` 有变更的推送/PR 上运行；指标一致性测试在指标指向的上游源文件（`enrich/`、`selfupdate/`、`guard/`、`db/`、`cmd/artex/main.go）变更时也会运行，捕捉上游再同步改掉 UA/标记/默认端口导致规则悄悄过期的情形。只改规则不更新测试/层、破坏 SigmaHQ 惯例、或与源码不一致的变更，都会在合并前被 CI 拦下。

规则索引与各规则的依据/局限见 [`detections/README.zh.md`](detections/README.zh.md)，测试的断言项与运行方法见 [`detections/tests/README.ko.md`](detections/tests/README.ko.md)。

---

## 许可

本项目以 **GNU Affero General Public License v3.0（AGPL-3.0）** 发布。提交贡献即表示你同意该贡献同样以 **AGPL-3.0 公开**；尤其当你修改本项目并经由网络（如在线服务）向用户提供时，必须向这些用户公开对应的完整源代码。完整条款见 [LICENSE](LICENSE)。
