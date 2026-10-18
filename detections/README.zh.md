# ARTEX 检测规则

中文 · [English](README.md) · [한국어](README.ko.md)

> 本目录把[防御·检测指南（../docs/defense-zh.md）](../docs/defense-zh.md)第 4 节「检测规则」中的伪规则，转换为可直接部署的厂商中立 [Sigma](https://sigmahq.io) 格式（翻译成各自的 SIEM/EDR 查询语言即可使用）。这里收录的所有指标都**不是推测**，而是从本仓库源码实际确认的字符串或行为。所有规则仅用于保护自己拥有或获得书面授权系统的**防御·检测目的**。

## 原子（atomic）规则

- **`sigma/artex_enrich_user_agent.yml`**：ARTEX 资产补全（`enrich/enrich.go`）发出的入站 `artex-enrich/1.0` User-Agent。目标侧可观测的辅助指标。`level: high`。
- **`sigma/artex_selfupdate_egress.yml`**：自更新例程（`selfupdate/github.go`）发出的出站 `artex-selfupdate` User-Agent。主机/取证视角的 egress 指标。`level: medium`。
- **`sigma/artex_guard_audit_framing.yml`**：工具调用被拦截时写入审计日志的平台管控标记（`guard/guard.go`）。主机/取证指标。`level: high`。
- **`sigma/destructive_command_hunting.yml`**：ARTEX guard 默认拦截清单（`db/db.go` 种子）的直接镜像：破坏性 shell/数据库命令。不是 ARTEX 专属特征，而是通用狩猎线索。`level: medium`。
- **`sigma/artex_recording_proxy_ca.yml`**：记录代理按 `_ca/mitmproxy-ca-cert.pem` 布局生成的 MITM CA 证书文件（`traffic/traffic.go`）。主机/取证工件；文件名本身与独立运行的 mitmproxy 相同，按狩猎线索对待。`level: medium`。

## 关联（correlation）规则：行为型

静态字符串可以改，行为更难隐藏。[`sigma/correlation/`](sigma/correlation/) 的 Sigma **关联**规则承载防御指南（4.1~4.2 节、4.4 节）的行为层。每条关联规则以 `id` 引用上述原子规则，因此解析引用时需要转换整个 `sigma/` 树而不是单个关联文件（见下文）。

- **`sigma/correlation/artex_enrich_scan_velocity.yml`**：同一来源在短时间窗内倾泻的一批 `artex-enrich/1.0` 探测（补全以并发 4 无限速运行）。捕捉单条规则漏掉的速度。`event_count`、`level: high`。
- **`sigma/correlation/artex_enrich_fanout.yml`**：同一来源把补全 UA 打到多个**不同**主机。以机器速度铺开整个资产清单的形态——线索是广度（breadth）而不只是量（volume）。`value_count`、`level: high`。
- **`sigma/correlation/artex_guard_block_burst.yml`**：同一主机上平台管控标记反复出现。不是文档引用了标记字符串，而是有一个正在运行的 ARTEX 实例在碰自己的 guard。`event_count`、`level: high`。
- **`sigma/correlation/artex_guard_marker_then_destructive.yml`**：同一主机在时间窗内同时出现 guard 标记与破坏性命令（防御指南 §4.2，多阶段）。把 ARTEX 专属标记与本身通用的破坏命令信号结合以提高特异度。`temporal`、`level: high`。

阈值与时间窗是保守默认值，请按各自基线调整。§4.2 的纯 Web 多阶段场景（枚举 → 探测 → 认证）无法还原为单一 ARTEX 专属 UA，仍需按环境定制基线规则；可直接上手的通用行为型 Sigma 基线模板放在[防御指南 §4.2](../docs/defense-zh.md#42-siem-关联规则)。因无法以 ARTEX 源码锚定，未纳入本目录受测规则树。

## 网络规则（Suricata）

Sigma 面向主机与日志遥测。网络线路上可观测的 ARTEX 专属 User-Agent 共两种，均已收入 [`suricata/`](suricata/) 的 [Suricata](https://suricata.io) 规则：补全探测的 `artex-enrich/1.0`（`enrich/enrich.go`）对应一条存在特征与一条高速枚举变体（sid 1000001、1000002）；norma SDK WebFetch 工具在攻击阶段发出的 `norma/0.4`（`github.com/Autumn-27/norma/tool/webfetch.go`）对应一条存在特征（sid 1000003）。其余 worker 工具（Bash 执行的 `curl`、`nmap` 等）使用各自 UA、没有 ARTEX 专属指纹，因此网络层有意只窄口径覆盖这两个 UA。适用范围与 TLS 注意事项、用 `suricata -T` 与参考 pcap 验证的方法见 [`suricata/README.ko.md`](suricata/README.ko.md)。

## ATT&CK 覆盖

规则标注的技术整理为 [MITRE ATT&CK](https://attack.mitre.org/) Navigator 层 [`attack/artex_navigator_layer.json`](attack/)：跨六个战术（侦察、命令控制、执行、影响、凭据访问、收集）的八项技术，每项都基于规则的 `attack.*` 标签并按检测强度（ARTEX 专属特征还是通用狩猎线索）打分。用 [ATT&CK Navigator](https://mitre-attack.github.io/attack-navigator/) 打开即可查看哪些规则覆盖哪些 ARTEX 行为。打分口径、技术↔规则对应与诚实的范围说明（覆盖不等于完备）见 [`attack/README.ko.md`](attack/README.ko.md)。[一致性测试](tests/attack/run.sh)保证层与规则集不互相漂移。

## 入侵指标清单（机器可读）

面向只要原子指标本身、不要检测逻辑的防御者：[`indicators/artex_indicators.csv`](indicators/) 把 ARTEX 发出的专属指纹收进一个 CSV，包含补全/自更新 User-Agent、guard 审计标记、服务端/代理默认端点、记录代理 CA 证书与 PostgreSQL 探索图 schema 指纹，每行附来源源码文件与（如适用）建立其上的规则，可直接导入威胁情报平台、SIEM 查找表或主机分诊清单。同一批指标另提供可直接导入的 [MISP](https://www.misp-project.org/) 事件（[`indicators/artex_indicators.misp.json`](indicators/)），用 MISP 或经其导出 STIX 的防御者无需手工映射 CSV 列。有规则背书的指纹标了 `to_ids`；主机取证用的端口与 schema 指纹未标。通用狩猎线索（破坏命令）与 norma SDK 共享的 `norma/0.4` WebFetch UA（Suricata sid 1000003 捕获的是网络签名而非 ARTEX 专属字符串）为避免误报被有意排除在导入清单之外。列结构、MISP 类型映射、诚实的注意事项以及保持 CSV 与 MISP 事件同步的一致性测试见 [`indicators/README.ko.md`](indicators/README.ko.md)。

## 主机分诊（triage）

上述规则面向使用 SIEM、网络传感器或威胁情报平台的防御者。另一类响应者——没有 SIEM、站在一台可疑主机 shell 前的人——可以使用 [`triage/artex_host_triage.py`](triage/)：一个只读脚本，仅凭本地状态回答「这台机器上跑没跑过 ARTEX」。它检查同一批指纹，并额外覆盖 **CSV 有意不带 Sigma 规则的三项主机/数据库指标**（服务端监听端口、记录代理端点、PostgreSQL 探索 schema）——它们不经日志或网络暴露，只能在主机上直接确认。脚本还会从 `/proc` 或 `--proc-from` 转储中检查记录器向子进程注入环境变量的痕迹，即运行中的进程是否同时携带代理变量与 mitmproxy CA 信任变量（`agent/worker.go`）。每项发现都是与对应指标行同等局限的分诊线索。详见 [`triage/README.ko.md`](triage/README.ko.md)；内置 `--self-test` 进入下文合并门禁。

## 测试

每条规则都配有 Docker 即可运行的复现测试，位于 [`tests/`](tests/)：

- **Suricata**（[`tests/suricata/run.sh`](tests/suricata/run.sh)）：先验证整个规则文件能以 `suricata -T --init-errors-fatal` 加载（哪怕是碰不到任何抓包的规则，解析失败也会暴露），再用 scapy 合成确定性 pcap 并以 `suricata -r` 回放——断言存在特征规则每次探测命中一次、速度规则过阈值触发、良性 UA 抓包零告警。二进制 pcap 不入库，每次运行重新生成。
- **Sigma**（[`tests/sigma/run.sh`](tests/sigma/run.sh)）：把下文「验证与转换」的 `sigma check` 与 `sigma convert` 变成可执行测试：0 错误、整个树编译为后端查询、每个原子指标字符串在查询中存活、关联规则单独转换会失败（证明关联规则确实依赖其引用的原子规则）。
- **Sigma 实时事件匹配**（[`tests/sigma_match/run.sh`](tests/sigma_match/run.sh)）：上一个测试证明规则有效且可编译，这个证明原子与关联规则**真的会触发**。每条原子规则断言典型恶意样本事件触发、正常样本不触发（例如 `.mitmproxy/` 下独立 CA 文件不会触发要求 `_ca/` 目录的记录代理规则）；每条关联规则断言在阈值于时间窗内被单组满足的良性时间线上触发，在阈值不足、超窗、分组拆分、缺腿时间线上保持静默。解析全部交给 pySigma，测试只断言编译后的条件树与聚合规格。
- **ATT&CK 层**（[`tests/attack/run.sh`](tests/attack/run.sh)）：覆盖层与规则集保持一致——打分的技术/战术必须恰好等于规则集的 `attack.*` 标签，每项技术必须指向真实存在的规则文件；只改一边不改另一边即失败。
- **指标真源（source-of-truth）**（[`tests/indicators/run.sh`](tests/indicators/run.sh)）：确认规则锚定的指标仍是上游源码发出的原字符串（`enrich/enrich.go` 的 `artex-enrich/1.0`、`selfupdate/` 的 `artex-selfupdate`、`guard/guard.go` 的 guard 标记、`db/db.go` 的破坏命令 token），捕捉其他三个测试都漏掉的漂移——所有规则编译并触发的同时，上游同步却改掉了 UA 或标记。同一测试还重读机器可读的 [`indicators/artex_indicators.csv`](indicators/artex_indicators.csv)，断言每行发布内容仍有源码与规则背书，防御者导入的产物也不会过期。最后断言所有被读取的上游源文件都进入 CI 工作流的 `push`/`pull_request` 路径过滤，使只改一个锚定源的 PR 无法跳过该测试、漂移无法穿过合并门禁——把「不是推测、而是源码确认」的承诺变成硬约束。
- **MISP 导出一致性**（[`tests/misp/run.sh`](tests/misp/run.sh)）：证明 MISP 事件（[`indicators/artex_indicators.misp.json`](indicators/artex_indicators.misp.json)）是合法 MISP 文档——由 [pymisp](https://github.com/MISP/PyMISP) 实际加载（其对象模型拒绝不存在的属性类型），并逐行与 CSV 同步：相同取值、每指标预期的 MISP 类型/分类、以及忠实反映 CSV 口径的 `to_ids`/`disable_correlation` 标志（有规则背书=可处置故 `to_ids` 开；主机取证端口=分诊线索故 `to_ids` 关且停用关联）。该事件与 CSV 并排手工维护，增删改 CSV 行时同一提交必须同步改 MISP 事件，否则测试持续失败。
- **Sigma 后端可移植性**（[`tests/sigma_backends/run.sh`](tests/sigma_backends/run.sh)）：证明规则不止能转出一个 Splunk 示例——整树（原子+关联）编译到 Splunk、Elasticsearch `eql`、Grafana Loki；五条原子规则在不支持 Sigma 关联的后端（Elasticsearch `lucene`、Microsoft `kusto`）仍可编译。给下文的后端支持表提供可复跑的验证。
- **SigmaHQ 惯例 lint**（[`tests/sigma_lint/run.sh`](tests/sigma_lint/run.sh)）：以 [`tests/sigma_lint/validators.yml`](tests/sigma_lint/validators.yml) 记录的基线运行 SigmaHQ 全量校验器（`pySigma-validators-sigmahq` 插件，普通 `sigma check` 不加载）并断言 0 issue；同时确认校验器全量运行、仅保留有意排除并已文档化的四项检查，规则一旦引入新的惯例问题（标题大小写错误、字段脱离分类体系）即构建失败。

八项规则测试之外，还有两个非规则门禁在同一 CI 工作流与 [`tests/run-all.sh`](tests/run-all.sh) 中运行：一是测试装置同步检查（run-all.sh、CI 与各套件目录以相同顺序调用相同套件），二是[主机分诊工具](triage/)的 `--self-test`（[`tests/triage-selftest.sh`](tests/triage-selftest.sh)）——构造合成主机断言所有分诊检查触发、干净主机零发现。

任一脚本在任一断言失败时以非零码退出。详见 [`tests/README.ko.md`](tests/README.ko.md)。

## 如何诚实地解读这些规则

- **静态指标可被修改。** 运营者可以把 UA 改成别的值，因此**查不到** `artex-enrich/1.0` 或 `artex-selfupdate` **不代表安全**。持久的信号是*行为*：同一来源从侦察 → 枚举 → 探测 → 认证/注入推进、随响应自适应、不间断运转（见防御指南第 1、2、4.1~4.2 节）；上述 `sigma/correlation/` 规则承载了可部署的关联（速度、扇出、guard 拦截爆发、guard 标记+破坏命令多阶段），纯 Web 多阶段场景仍需环境定制基线。
- **破坏命令规则是通用狩猎。** 它镜像 ARTEX guard 拦截清单，但正当管理员也会执行相同命令：命中按线索处理，按环境配置白名单，不要仅凭它断言 ARTEX。
- **端口指标属于主机取证而非 Sigma。** 服务端默认 `:8787` 与记录代理 `127.0.0.1:8788`（`cmd/artex/main.go`）更适合在可疑主机上用 `ss`/`netstat` 确认——因此没有做成嘈杂的网络规则，而是写进防御指南并收录于[指标 CSV](indicators/)；[主机分诊脚本](triage/)正是为有 shell 没 SIEM 的响应者代跑这些主机本地检查（端口、记录代理工件、日志标记、PostgreSQL schema）。

## 验证与转换

规则以 [sigma-cli](https://github.com/SigmaHQ/sigma-cli)（pySigma）验证。复现方式：

```sh
python3 -m venv .venv && . .venv/bin/activate
pip install sigma-cli

# 结构 + 最佳实践校验（预期：0 错误、0 issue）
sigma check detections/sigma/

# 按本独立规则集的文档化基线跑 SigmaHQ 全量惯例检查（预期：0 issue）；
# 普通的 `sigma check` 不会加载这些校验器
pip install pySigma-validators-sigmahq
sigma check --validation-config detections/tests/sigma_lint/validators.yml detections/sigma/

# 编译为目标查询语言，例如 Splunk
sigma plugin install splunk
sigma convert -t splunk --without-pipeline detections/sigma/artex_enrich_user_agent.yml

# 转换整个树，以便解析关联规则以 id 引用的原子规则
sigma convert -t splunk --without-pipeline detections/sigma/
```

基线强制全部 SigmaHQ 惯例，仅排除与本独立规则集无关的 SigmaHQ monorepo 文件组织/分类体系四项检查；每项排除及理由记录于 [`tests/sigma_lint/validators.yml`](tests/sigma_lint/validators.yml) 并由上述 lint 测试强制。

### Sigma 后端可移植性

`sigma/correlation/` 规则以 `id` 引用原子基规则，只有支持 Sigma 关联转换的后端才能转换，且支持情况因后端而异，`-t` 的选择因此重要。下表按固定基准（`sigma-cli` 3.1.0 与兼容的最新后端）实测，[`tests/sigma_backends/run.sh`](tests/sigma_backends/run.sh) 可复现：

- **整树（原子+关联）可转换**：Splunk（`-t splunk`）、Elasticsearch EQL（`-t eql`）、Grafana Loki（`-t loki`）。直接转换 `detections/sigma/` 即连同关联查询一起得到。
- **仅原子规则（暂不支持关联）**：Elasticsearch Lucene（`-t lucene`）、OpenSearch（`-t opensearch_lucene`）、面向 Sentinel/Defender XDR 的 Microsoft `kusto` 后端（`-t kusto`）。在这些后端转换五条原子规则，关联时间窗在产品内原生表达（如 Sentinel 计划分析的 `summarize ... by bin(TimeGenerated, 30m)`）；传入整个目录会以 "Backend does not support correlation rules" 停止。

```sh
# 仅原子规则，例如 Microsoft Sentinel / Defender（kusto 后端）
sigma plugin install kusto
sigma convert -t kusto --without-pipeline \
  detections/sigma/artex_enrich_user_agent.yml \
  detections/sigma/artex_selfupdate_egress.yml \
  detections/sigma/artex_guard_audit_framing.yml \
  detections/sigma/artex_recording_proxy_ca.yml \
  detections/sigma/destructive_command_hunting.yml
```

固定版本上的已知边界：Elasticsearch ES|QL 目标（`-t esql`）拒绝 guard 标记规则（`String value expressions are not supported`），在该目标下转换其余三条原子规则即可；IBM QRadar 插件（`ibm-qradar-aql`）与固定版 pySigma 不兼容、需 `--force-install`，故不在测试覆盖内。已安装后端可用 `sigma list targets` 查看。

上例使用 `--without-pipeline`，直接输出规则体中的通用字段名（`cs-user-agent`、`cs-host`、`CommandLine`）。要对齐产品 schema，去掉该旗标并用 `-p` 应用处理管道（见 `sigma list pipelines`）。注意产品管道只映射字段名，还可能要求规则通用 `logsource` 未指定的目标表——例如 `-p sentinel_asim` 在设置匹配数据的 `query_table` 前会以 "Unable to determine table name" 停止，部署前请把字段与目标表映射到自己的环境。

## 贡献

欢迎检测与加固贡献。新规则须把所有指标锚定在可观测事实上、在 `description` 中说明局限、干净通过 SigmaHQ 校验器基线（`sigma check --validation-config tests/sigma_lint/validators.yml`），且不得包含可被读作攻击指南的内容。参见 [`../CONTRIBUTING.md`](../CONTRIBUTING.md)。
