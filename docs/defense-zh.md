# 自主 AI 攻击防御·检测指南

> 本文档面向防守方，帮助理解 ARTEX 这类**自主 AI 渗透 agent** 的运作方式，并建设**检测与阻断**能力。它不是攻击方法教程。所有内容仅用于保护自己拥有或获得书面明确授权的系统；未经授权对他人系统进行检测/攻击本身就是违法犯罪（见[上级 README 的安全·滥用警告](../README.md)）。
>
> English: **[Defense and Detection Guide (defense-en.md)](defense-en.md)** · 한국어: [defense-ko.md](defense-ko.md)

自主 AI 攻击工具把过去由一名人类手工驱动的渗透测试，变成 LLM agent **自行拆解目标、执行真实工具、累积发现**的 7×24 自动化过程。防守者面对的对手从「一名熟练攻击者」变成「不知疲倦的 agent 集群」。本指南梳理这一变化对检测与响应提出了什么要求。

---

## 1. 自主 AI 攻击与传统扫描器有何不同

传统漏洞扫描器按预定项清单顺序发包然后结束；ARTEX 类自主 agent 的结构不同（见[系统技术架构](../README.md#系统技术架构)），以下要素结合后**无需人工介入即可走完多阶段攻击链**：

- **角色分工的多 agent。** 拆解目标的 `goals`、判定下一步方向并派生意图（intent）的 `planner`、各领一条意图用真实工具执行的多个 `worker`、供人类介入的 `mainagent`。planner 是唯一的意图生成者，worker 们并行执行。
- **双图累积状态。** 「有什么」（资产图）与「测到哪了」（探索图）分开维护并以锚点相连，因此攻击**渐进加深**、从不同角度重访同一资产、在先前观察之上搭建下一步。
- **事件驱动闭环。** 图一变 planner 就被唤醒派发意图，worker 的写回又触发下一轮，直到目标被证明为止。
- **串行攻击链稳定推进。** planner 把「发现注入点 → 获取凭据 → 横向移动 → 提权」这类依赖顺序记入共享 todolist，只对前置已满足的下一步派意图，攻击链在无状态会话下不错序地走完。
- **LLM 按上下文变形 payload。** 执行的工具与 payload 不是代码里的固定常量，而是 LLM 依据情境生成的值，每个请求的形态都略有不同。

### 为什么更难检测

- **固定特征匹配容易失效。** payload 随上下文变化，精确匹配已知恶意字符串的 WAF 特征容易被绕过。
- **可以慢速、像人一样断续推进。** agent 在轮次间休息、随发现调整方向，只看「短时间大量请求」的速率检测会漏掉。
- **侦察与渗透连成一条流。** 人类「看侦察结果、几天后手工打」的时间间隙消失，从首次接触到数据带出的时间大幅缩短。

### 但仍然可检测：行为难以隐藏

静态指纹（User-Agent、特定 payload 字符串）运营者想改就能改；而**自主 agent 的行为模式**是攻击的本质，很难改掉。第 2~4 节正是以行为视角为主线：

- 同一来源以**一致的会话**连续执行**多阶段、不同性质的请求**（侦察 → 枚举 → 认证尝试 → 利用）。
- 在人类会休息的时段**不间断推进**的探索。
- 从失败响应中提取线索、**系统性变形下一个请求**——不是随机 fuzz，而是自适应推进。

---

## 2. 防守方可观测的指纹（IoC·特征）

分两个视角看：**(a) 目标（被攻击方）视角**——指向我方系统的 ARTEX 流量中能看到什么；**(b) 运营者/取证视角**——在真正运行过 ARTEX（或被当作跳板）的主机上能看到什么。两者不要混淆：目标侧可见的静态指纹有限，行为指纹才是核心。

### (a) 目标视角：指向我方系统的流量

- **资产补全（enrich）探测的 User-Agent `artex-enrich/1.0`。** ARTEX 自动补全资产（DNS/HTTP 验证）时用该 UA 直接向目标发 `GET`（`enrich/enrich.go`）。该路径与 LLM 无关、由 ARTEX 自行发起，特点是**不跟随重定向、关闭 keep-alive、只读响应开头提取 `<title>`**，默认并发 4。因此若看到 `artex-enrich/1.0` UA 的**短连接、单发 GET、读完标题即断**的查询同时打到多个资产，强烈提示 ARTEX 系补全流量。但运营者可以改 UA，**缺失不等于安全。**
- **主攻击流量呈现所执行工具的默认指纹。** worker 用真实工具（Bash 执行的外部工具、HTTP）发请求；ARTEX 只是在子进程环境变量注入 `HTTP_PROXY` 与代理 CA 路径让流量过自带记录代理，**并不给攻击流量强加 ARTEX 专属 User-Agent**。目标看到的 UA/头是**当时所执行工具的默认值**；运营者没定制则留下常见自动化工具指纹，定制了也可能伪装成正常浏览器。因此**不要依赖单一 UA 匹配，要与行为检测结合**。
- **worker 内置 WebFetch 工具留下 `norma/0.4` User-Agent。** 与 Bash 执行的外部工具不同，ARTEX 通过 norma SDK（`github.com/Autumn-27/norma`）直接发起的 HTTP 查询（WebFetch 工具）在攻击阶段携带 SDK 默认 UA `norma/0.4`。明文 HTTP（或 TLS 终结点）下可在网络侧观测，[Suricata 规则 sid 1000003](../detections/README.zh.md) 匹配该前缀（`norma/`）。该 UA 并非 ARTEX 专属（其他基于 norma SDK 的工具同样使用），与 `artex-enrich/1.0` 一样只是辅助线索；Bash 执行的工具各有自己的 UA，**缺失不等于安全。**
- **没有内置限速。** ARTEX 本身没有针对目标流量的速率限制，请求节奏由 LLM 驱动的外部工具决定；但每任务默认 3 个 worker 并行，同一目标上**多条意图同时推进**。既可能表现为「慢速单会话」，也可能表现为「多角度并行」，单一固定阈值难以命中。
- **行为特征（最重要）。** 下列模式的**同时出现**指向自主 agent：
  - 同一来源（或少量轮换来源）在**短时间内连锁**完成侦察 → 目录/端点枚举 → 参数探测 → 认证/注入尝试。
  - 对同一端点**根据响应码/长度系统性变形**的连续请求（自适应，非随机 fuzz）。
  - 超出人类运营时段**长时间不间断**的会话。
  - 401/403/429 之后不停手、持续尝试**绕过变体**的韧性。

### (b) 运营者/取证视角：运行过 ARTEX 的主机

用于入侵调查中在跳板/中继主机上寻找 ARTEX 安装运行的痕迹。

- **默认监听端口 `:8787`。** ARTEX 服务端默认 HTTP 监听地址（`cmd/artex/main.go`，可用 `--addr` 修改）。内网主机在该端口开着管理 UI（仪表盘/任务/资产图）即应怀疑 ARTEX 实例。
- **记录型 MITM 代理 `127.0.0.1:8788`。** 拦截并记录 worker Bash/HTTP 执行的本地代理默认地址（`--proxy` 默认值，仅回环）。它生成自有 CA 解密并记录 TLS（`mitmproxy-ca-cert.pem`），因此主机上是否存在 **ARTEX 安装的受信 CA 证书**、是否向子进程注入 `HTTP_PROXY` 与代理 CA 路径环境变量，都是线索。注入进入 ARTEX 生成的所有 worker 工具且变量名硬编码于源码（`agent/worker.go`），**运行中的进程同时携带代理变量与 mitmproxy CA 信任变量**是比端口号更特异的指纹。[主机分诊工具](../detections/triage/README.md)从 `/proc`（取证镜像则为环境变量转储）核对该组合。
- **自更新回调 `artex-selfupdate`。** 自更新查询 GitHub Releases 时使用的 User-Agent（`selfupdate/`）。出网日志中若看到该 UA 访问代码托管主机，提示存在 ARTEX 二进制。
- **PostgreSQL 中的双图。** 存在 `exploration_nodes`、`assets`、`companies`、`activity` 等表与 `agent_prompts` 种子数据的数据库是 ARTEX 数据存储的特征。
- **基于数据库的正则审批规则与审计日志。** 评估工具调用的 intercept 规则存于数据库并按优先级以正则求值（`intercept/`）；被拦截的调用在审计日志（`GET /api/audit`）中以 `【ARTEX 平台管控·非目标防御】` 开头的管控措辞记录——受损主机审计记录中出现该字符串可佐证 ARTEX 运行。
- **破坏性命令狩猎指标。** ARTEX 自带 guard 内置拦截的命令模式，正是自主 agent **可能尝试**的命令集合的镜像。主机命令审计可把以下作为狩猎指标：`rm -rf`、`mkfs`、`dd of=/dev/`、`shred`/`wipe`、SQL `DROP DATABASE`/`DROP TABLE`/`TRUNCATE`、MongoDB `drop`/`dropDatabase`、Redis `FLUSHALL`/`FLUSHDB`、`curl`/`wget` 的 `-X DELETE`，以及 `curl … | nc …` 类外带管道。注意最后的外带管道不是破坏数据而是往外偷数据的泄露信号，性质不同：ARTEX guard 也仅对这一类默认关闭内置（正常检测用途的反弹 shell/数据传输管道误报频繁），可部署的破坏性命令狩猎规则（[`destructive_command_hunting.yml`](../detections/sigma/destructive_command_hunting.yml)）也只覆盖破坏范围而不含它——请把外带管道作为独立狩猎指标按环境调优，而不是直接封禁。

> 小结：**目标侧防御押在行为指纹上，静态 UA（`artex-enrich/1.0`、`norma/0.4` 等）只作辅助线索**；运营者/取证指纹（`:8787`、`127.0.0.1:8788`、`artex-selfupdate`、数据库 schema、审计日志措辞）用于**调查被入侵的中继主机**。

### 为什么封 IP 是弱的一线防御

安全事件曝光后，社区常流传「共享攻击 IP 列表、防火墙封掉」。与分享者的善意相反，**不建议把来源不明的非官方 IP 列表直接变成封禁规则**，对自主 AI 攻击尤其如此：

- **来源无法核验。** 个人张贴的非官方列表无法确认采集依据，混入与事件无关的 IP 也无从过滤。
- **很快过时。** 自主 agent 通过 VPN、云实例、失陷中继频繁更换出口 IP（见上文「少量轮换来源」）；昨天观测到的攻击 IP 今天很可能已弃用，封了列表攻击者换个 IP 就回来。
- **误伤风险大。** 列表混入共享网段、CDN、正常云 IP 时，一封就把正常客户流量或内部服务一起断掉；与自动封禁（第 6 节）结合时误伤扩散更快。

封 IP 并非无用——**从应急机构或可信威胁情报获取官方 IoC，核验有效期与误伤风险后应用**才有意义。但封一行 IP 只是追赶轮换出口的临时措施；持久有效的是难以改变的**行为**（第 2~4 节行为检测）与**攻击面收缩**（第 3、5 节加固）。「指纹能改，行为难藏」这一前提在这里同样成立。

---

## 3. 攻击者瞄准的入口与加固

自主 agent 瞄准与人类攻击者**相同的弱点**，只是更快、更执着。以下是防守视角的优先加固点。

### 3.1 外部暴露面与已知（n-day）漏洞

自主 agent 最先、也最稳定瞄准的不是刁钻 0-day，而是**暴露在外网且未修复的已知漏洞**：边界设备（VPN/防火墙）、对外开放的管理与运维控制台、应用服务器/中间件/框架（如被广泛滥用的 WebLogic、Struts 系），以及**非主营业务挂载的合作/招聘/员工辅助系统**。自主 agent 用资产图自动枚举暴露面后，把有公开 PoC 的 n-day **赶在补丁落地前**同时对数百资产投放——这个「找到暴露弱点并代入」的速度是与人类攻击者拉开差距的地方。

- **收缩攻击面。** 持续维护对外暴露资产/管理控制台/辅助系统的清单，没必要对外的收进内网、VPN、白名单之后。
- **快速修复已知漏洞。** 边界设备、Web 服务器、应用服务器、中间件的公开漏洞是自主 agent 的头号目标；以 [CISA KEV（已知被利用漏洞目录）](https://www.cisa.gov/known-exploited-vulnerabilities-catalog)等持续更新的官方清单为优先级输入（国内可与 CNVD/CNNVD 通报交叉核对），把有公开 PoC 组件的修补间隔压到最短。以动态官方清单为准、而不是把特定 CVE 号写死在文档里，agent 换用新 n-day 时才不会掉队。
- **收紧必须暴露的接口。** 不得不对外的管理/运维接口加上来源限制（IP 白名单）、MFA、VPN，让未认证枚举本身无法进行。
- **辅助系统与主业务同标准管理。** 合作/招聘/员工辅助系统与主业务同等打补丁、同等级监控——自主 agent 钻入的入口经常是这些辅助路径。认证流程本身的加固见 3.2。

检测视角：针对已知漏洞路径（URL/参数）的请求从外部短间隔连锁出现，即 n-day 扫描信号；与第 2 节行为指纹、第 4 节同源多阶段关联规则结合时最有效。

### 3.2 辅助认证与身份核验流程

附加服务、合作与招聘渠道等**以不同于主业务的路径接入的认证/身份核验流程**往往校验更松，是绕过的重点目标；自主 agent 会自动枚举这些路径、读响应差异、系统性地找绕过条件。

- 身份核验与认证强度**与主业务统一**，全面排查可绕过的辅助路径。
- 认证状态迁移（未登录 → 登录、普通 → 提权）在**服务端重新校验**，不信任客户端提交的信任标记（Cookie/头/参数）。
- 审计身份核验令牌、一次性验证码的**有效期、可重用性、可猜测性**。

### 3.3 API 认证与授权（IDOR、越权）

- 所有对象访问强制**服务端归属与权限校验**（阻断仅换 ID 即可打开他人资源的 IDOR）。
- 排查水平/垂直越权路径。自主 agent 会机械化递增标识符做批量尝试，**单点手工测试漏掉的洞**很快被它找到。

### 3.4 凭据填充（credential stuffing）

拿泄露账密列表撞库的攻击被自主 agent 用**速度与分散度**放大。

- 登录/身份核验端点施加**自适应限速**（按 IP、账号、设备、行为）。
- 敏感操作强制 **MFA**：撞对密码也过不了第二关。
- 用**泄露凭据检测**（已知泄露库比对、异常登录位置/速度）先发制人。
- 对登录失败/成功**分布变化**（突发的低速广撒尝试）告警。

### 3.5 会话、令牌与密钥管理

- 最小化会话令牌的**范围、寿命与刷新**，敏感迁移处重新认证。
- API key、内部令牌不得出现在**响应、日志、报错信息**里（自主 agent 会积极收集报错里的线索）。

---

## 4. 检测规则与日志模式（实战）

以不绑定具体产品的**伪规则**记录，请翻译成自己的 WAF/IPS/SIEM 语法。其中基于静态指纹的部分已提供可直接部署的 [Sigma 规则（`detections/sigma/`）](../detections/README.zh.md)；核心的行为/关联检测（4.1、4.2）虽无法还原成单条规则，但其中能以 ARTEX 源码佐证的行为指标（补全查询速率、补全目标数、guard 拦截爆发、guard 标记与破坏性命令同主机并发）也提供了可部署的 [Sigma 关联规则（`detections/sigma/correlation/`）](../detections/README.zh.md)。其纯 Web 多阶段关联（枚举 → 探测 → 认证）无法还原为单一 ARTEX 专属 UA，需要按环境定制基线规则——4.2 给出了可直接部署的通用行为型 Sigma 基线模板，请按自己的 SIEM 与基线调优后作为起点。网络层明文 HTTP（或 TLS 终结点）可观测的 ARTEX User-Agent 共两种，均已提供 [Suricata 规则（`detections/suricata/`）](../detections/suricata/README.md)：补全探测的 `artex-enrich/1.0`（存在特征 sid 1000001、高速枚举变体 sid 1000002），以及 norma SDK WebFetch 在攻击阶段发出的 `norma/0.4`（sid 1000003）。

### 4.1 WAF/IPS（行为模式）

- 同一来源以**单一会话**连续出现**性质各异的请求群**（静态资源占比低，枚举/参数探测/认证尝试占比高）时加分。
- 对**根据响应码与响应长度变形的连续请求**（高熵但非随机的自适应模式）加权。
- `artex-enrich/1.0` 等已知自动化 UA 直接标**高危**，但不要把 UA 缺失解读为安全。

### 4.2 SIEM 关联规则

- **同源多阶段关联**：同一 IP/ASN/会话在短窗口内先后出现 (a) 目录/端点枚举、(b) 参数探测、(c) 认证/注入尝试 → 「疑似自主攻击」告警。
- **时段异常**：偏离业务正常流量分布、**长时间不间断**的单一会话。
- **失败后持续**：吃到 403/429 仍不停手、继续**绕过变体**的来源。

下面是可直接部署的**同源多阶段关联**基线模板。由于攻击流量不带 ARTEX 专属 UA，该模板与 `detections/sigma/` 中锚定 ARTEX 源码的规则不同，是**通用行为型规则**：不看特定工具指纹，只看「同一来源在短窗口内完成枚举、探测、认证三件事」这一行为。文件内含三条子规则与一条 temporal 关联规则（同一客户端在时间窗内三者全部满足才触发）。

```yaml
# ── 通用行为型模板（非 ARTEX 专属特征）──
# 对应防御指南 4.2 节的同源多阶段 Web 模式（枚举 → 探测 → 认证）。
# ARTEX 攻击流量不带 ARTEX 指纹，因此与 detections/sigma/ 的规则不同，
# 本模板不锚定 ARTEX 源码，是通用行为型起点。字段名（SigmaHQ Web 服务器分类）
# 与阈值/时间窗务必按自己的日志与基线调整。
# 自包含：三条子规则 + 同一客户端在窗口内三者全部满足才触发的 temporal 关联规则。
title: Web Endpoint Enumeration Burst From One Source
id: f03c360c-dc33-4a8a-afa8-821b1ff5c4e3
status: experimental
description: |
    Stage 1 of the same-source multi-stage pattern in the ARTEX defense guide section 4.2: a
    burst of endpoint or directory enumeration from a single client, seen as a high rate of 404
    and 400 responses in a short window. This is generic behaviour, not an ARTEX-specific
    signature; tune the count and window to your own baseline. On its own this leg is low signal
    and earns weight only inside the correlation below.
references:
    - https://github.com/AFOGSHEEP/artex-zh/blob/main/docs/defense-zh.md
    - https://github.com/AFOGSHEEP/artex-zh/blob/main/docs/defense-en.md
author: ARTEX defense guide (generic template)
date: 2026-10-07
tags:
    - attack.reconnaissance
    - attack.t1595
logsource:
    category: webserver
detection:
    enum_misses:
        sc-status:
            - 404
            - 400
    condition: enum_misses
falsepositives:
    - Broken links, authorised vulnerability scanners, or misconfigured clients that generate
      many 404 responses.
level: low
---
title: Web Parameter Or Path Injection Probe From One Source
id: f9296e55-6b7a-4030-b5f7-5f7b146233be
status: experimental
description: |
    Stage 2 of the same-source multi-stage pattern: parameter or path probing, matched here as
    query strings carrying common injection or traversal markers. This leg is unavoidably
    signature-like and noisy on its own, so it is scored low and earns weight only inside the
    correlation below. Extend the marker list to your own probe corpus and WAF categories; it is
    a coarse proxy for the broader "adapts requests to responses" behaviour the guide describes.
references:
    - https://github.com/AFOGSHEEP/artex-zh/blob/main/docs/defense-zh.md
    - https://github.com/AFOGSHEEP/artex-zh/blob/main/docs/defense-en.md
author: ARTEX defense guide (generic template)
date: 2026-10-07
tags:
    - attack.initial-access
    - attack.t1190
logsource:
    category: webserver
detection:
    probe_markers:
        cs-uri-query|contains:
            - '../'
            - "' or "
            - ' union select '
            - '<script'
            - '; drop '
    condition: probe_markers
falsepositives:
    - Legitimate request payloads that resemble probe markers; tune the marker list to your
      application.
level: low
---
title: Authentication Or Identity-Verification Attempt From One Source
id: 2614cacb-7455-46ad-9af2-7b9633f12d7b
status: experimental
description: |
    Stage 3 of the same-source multi-stage pattern: requests to login, authentication, or
    identity-verification endpoints, or 401 and 403 responses. Map the paths and your own
    authentication-event fields to your application; auxiliary, affiliate, and broker channels
    often expose weaker identity-verification endpoints than the main service and belong here
    too. This leg is broad by design and is only meaningful inside the correlation below.
references:
    - https://github.com/AFOGSHEEP/artex-zh/blob/main/docs/defense-zh.md
    - https://github.com/AFOGSHEEP/artex-zh/blob/main/docs/defense-en.md
author: ARTEX defense guide (generic template)
date: 2026-10-07
tags:
    - attack.credential-access
    - attack.t1110
logsource:
    category: webserver
detection:
    auth_path:
        cs-uri-stem|contains:
            - '/login'
            - '/auth'
            - '/verify'
            - '/otp'
    auth_deny:
        sc-status:
            - 401
            - 403
    condition: auth_path or auth_deny
falsepositives:
    - Ordinary users signing in; this leg is broad and only meaningful inside the correlation.
level: low
---
title: Same-Source Multi-Stage Web Attack (Enumeration, Probe, Auth)
id: 9b7c7b86-702f-42b8-be99-3e60a188ec5b
status: experimental
description: |
    The behaviour-based core of ARTEX defense guide section 4.2 as a deployable template: one
    client runs endpoint enumeration, parameter or path probing, and an authentication or
    identity-verification attempt within the same short window. This is the pattern an autonomous
    agent drives at machine speed and keeps driving past 403 and 429 responses. It is UA-free and
    carries no ARTEX fingerprint, so it is a GENERIC behavioural rule, not one of the
    ARTEX-source-grounded rules under detections/sigma/. Normalise the client field (c-ip, or a
    session identifier if you have one) and tune the window to your baseline. If three legs are
    too strict and miss cases, relax to any two of the three.
references:
    - https://github.com/AFOGSHEEP/artex-zh/blob/main/docs/defense-zh.md
    - https://github.com/AFOGSHEEP/artex-zh/blob/main/docs/defense-en.md
author: ARTEX defense guide (generic template)
date: 2026-10-07
tags:
    - attack.initial-access
    - attack.t1190
correlation:
    type: temporal
    rules:
        - f03c360c-dc33-4a8a-afa8-821b1ff5c4e3
        - f9296e55-6b7a-4030-b5f7-5f7b146233be
        - 2614cacb-7455-46ad-9af2-7b9633f12d7b
    group-by:
        - c-ip
    timespan: 10m
falsepositives:
    - An authorised vulnerability scan or QA run from a single source; allow-list its address.
level: high
```

使用该模板的注意事项：

- 该代码块用检测包相同的工具链验证过：`sigma check` 以 SigmaHQ 全量规范跑通（0 错误 0 issue），`sigma convert -t splunk` 可生成查询（三条子规则按 `c-ip` 在 10 分钟窗内全部满足才触发）。但因无法以 ARTEX 源码锚定，未放入 `detections/` 的受测规则树——那是「只收录从源码确认、而非推测」的原则。
- 该模板是关联（correlation）规则，`sigma convert` 能否导出整个模板取决于后端对 Sigma 关联转换的支持。固定版本（`sigma-cli` 3.1.0）实测：整模板在 Splunk（`-t splunk`）、Elasticsearch EQL（`-t eql`）、Grafana Loki（`-t loki`）可转换；Microsoft `kusto`（Sentinel/Defender）与 Elasticsearch Lucene（`-t lucene`）不支持关联规则转换（`Backend does not support correlation rules`），此时只转换三条子规则，「10 分钟窗·同 `c-ip`」的关联在产品内自行表达（如 Sentinel 计划分析规则的 `summarize ... by bin(TimeGenerated, 10m), <客户端>`）。实测支持矩阵见 [Sigma 后端可移植性](../detections/README.zh.md#sigma-后端可移植性)。
- 第二阶段（探测）依赖注入/穿越标记列表，是粗信号、单独误报多，因此三条子规则 `level` 低，只在三者同源同时出现的关联规则里升为高告警。
- 客户端按 `c-ip` 分组；在代理/CDN 后面请换成 `X-Forwarded-For` 还原的真实客户端地址或会话标识。三阶段全要求若过严漏报，可放宽为满足任意两条即触发。

### 4.3 认证日志

- 按账号/IP 的**登录失败率突变**、跨大量账号的**低速分散尝试**（撞库特征）、失败到成功的**异常迁移速度**。
- 对身份核验/一次性验证码端点的**枚举式访问**。

### 4.4 出网（egress）与取证

- 内部主机以 `artex-selfupdate` UA 访问代码托管主机的请求。
- 内部绑定 `:8787`（管理 UI）或 `127.0.0.1:8788`（记录代理）的进程。
- `artex-enrich/1.0` UA 短时间内查询大量外部资产的 DNS/HTTP 补全模式。

---

## 5. 加固清单

供防御组织快速自查：

- [ ] 登录、身份核验与敏感 API 已启用**自适应限速**（IP/账号/设备/行为）。
- [ ] 敏感操作强制 **MFA**。
- [ ] 所有对象访问都有**服务端归属/权限校验**（阻断 IDOR）。
- [ ] 认证状态迁移在**服务端重校验**，不信任客户端信任标记。
- [ ] **辅助/合作/招聘渠道**的身份核验强度已与主业务统一。
- [ ] 已运行泄露凭据**检测与比对**。
- [ ] WAF 以**行为模式**运营，不只依赖固定特征。
- [ ] **不直接套用户间流传的非官方 IP 封禁列表**；官方 IoC 从可信来源获取并核验有效期与误伤风险后再落地。
- [ ] SIEM 已配置**同源多阶段关联规则**。
- [ ] 认证、访问、出网**日志保留足够时长**（自主攻击节奏快，事后溯源依赖这些材料）。
- [ ] 维护对外资产/管理控制台/辅助系统**清单以收缩攻击面**，边界设备/应用服务器/中间件的**已知（n-day）漏洞快速修复**。
- [ ] 网络**分段**已收窄横向移动与提权的爆炸半径。
- [ ] 密钥/令牌**不出现在响应、日志、报错信息**中。
- [ ] 已准备**自动封禁/隔离**响应（只等人工审批跟不上自主攻击速度）。

---

## 6. 事件响应要点

自主 AI 攻击的特征是**速度**：人类一年量级的渗透与外带，agent 可在短得多的时间内完成；响应设计必须以此为前提。

- **自动处置为先。** 可疑来源隔离、会话吊销、限速骤降等动作要能在人工审批前**自动触发**；全部绑人工审批就跟不上攻击速度。
- **预先定义留存日志。** 认证日志、访问日志（含可合规保留的请求体）、出网日志、DNS 查询——自主攻击留痕积累很快，事后还原攻击链全靠这些。
- **按资产维度追攻击范围。** 攻击沿资产图蔓延，必须还原从首个入口资产到横向/提权的**完整路径**才能防再次入侵。

### 6.1 疑似主机/流量分诊（triage）流程

怀疑与 ARTEX 有关时按顺序检查以下内容。与第 2 节两个视角对应，分诊也分**(a) 我的服务是否成为目标**与**(b) 某主机上是否跑过 ARTEX**。任一阶段都不要凭单一命中下定论，以多指标与行为信号是否共同出现为准；静态指标全无但行为信号可见时，继续调查。

**(a) 目标侧：我的服务是否被 ARTEX 盯上**

1. 在访问/认证日志中检索补全探测 UA `artex-enrich/1.0`：是否有不跟随重定向的单发 `GET` 以短间隔同时打到多个资产（第 2 节 (a)）。运营者可改 UA，没查到也继续下一步。
2. 找同一来源（或少量轮换来源）的**多阶段连锁**：侦察到端点枚举、参数探测、认证/注入尝试短间隔衔接，随响应码/长度自适应，401/403/429 后仍不停手。该行为信号比静态 UA 更持久（第 2 节 (a) 行为特征）。
3. 有 SIEM 的，用 [Sigma 关联规则](../detections/README.zh.md)（补全查询速率/目标数、guard 拦截爆发、guard 标记与破坏性命令并发）挂住该行为，命中来源按「自动处置为先」原则进入隔离/吊销名单。

**(b) 主机取证：某主机上是否跑过 ARTEX**

在疑似主机上核对以下内容，指标依据见第 2 节 (b) 与机器可读的[入侵指标清单](../detections/indicators/artex_indicators.csv)。以下五项只读检查中，前四项（监听端口、出网日志、审计日志、状态/记录存储）由[主机分诊脚本](../detections/triage/)（`detections/triage/artex_host_triage.py`）一次性代跑；第五项命令审计因破坏性命令并非 ARTEX 专属指纹、正当管理员也会使用（只是狩猎线索）而不作为自动指标，需在主机命令历史中人工比对。没有 SIEM、只有 shell 时先跑脚本，各项命中按下文说明仅作线索。

1. **监听端口。** 在主机上直接确认默认服务端口 `:8787` 与回环记录代理 `127.0.0.1:8788` 是否开放：
   ```sh
   ss -ltnp | grep -E ':8787|:8788'   # 无 ss 时用 netstat -ltnp
   ```
   两个端口可用 `--addr`/`--proxy` 改，查询为空时也要结合全部开放端口与内网管理 UI 综合看。
2. **出网日志。** 出网日志中是否有以自更新 UA `artex-selfupdate` 访问代码托管主机（GitHub Releases）的请求（`selfupdate/`）——提示该主机运行过 ARTEX 二进制。
3. **审计日志。** 审计记录中出现管控标记 `【ARTEX 平台管控·非目标防御】`（`guard/guard.go`，每次拦截的工具调用都带该措辞）可佐证 ARTEX 运行。
4. **状态与记录存储。** ARTEX 把探索图放在 PostgreSQL（`exploration_nodes`、`assets`、`companies`、`activity` 表与 `agent_prompts` 种子），并把状态与记录写在可执行文件旁的数据目录（`--data` 默认值，见 `cmd/artex/main.go`）。数据目录下有按任务存放产物的 `tasks/`、存放会话记录的 `transcripts/`；记录代理的产物单独收集在其下 `traffic/` 子目录（`server/manager.go` 在数据目录下为记录代理开 `traffic/`）。因此受信 CA 证书在 `traffic/_ca/mitmproxy-ca-cert.pem`，流量索引在 `traffic/_index/index.sqlite`，记录的请求/响应体在 `traffic/_blobs/`；三者与 `tasks/`、`transcripts/` 同时出现，说明记录代理确实运行过。
5. **命令审计。** 用第 2 节 (b) 末尾的破坏性命令狩猎指标（`rm -rf`、`DROP DATABASE`、`FLUSHALL`、外带管道等）比对主机命令历史；正当管理员也可能使用相同命令，仅作线索。

静态指标（端口、UA、标记）可被运营者修改或清除，**缺失不等于安全**；把 (a) 的行为信号与 (b) 的主机痕迹放在一起判断，才是自主 AI 攻击分诊的核心。

---

## 7. 官方渠道：入侵指标、安全通报与报告义务

> 本节由韩文原版（[defense-ko.md](defense-ko.md) 第 7 节，KISA/FSI/个保委等韩国渠道）改写为中国大陆对应的官方渠道与法定义务；面向其他司法辖区的读者请替换为自己所在地区的对应机构。

防御组织应从官方渠道获取入侵指标与安全通报，并在事件发生时履行法定报告义务。请以下列官方来源为一级基准，替代用户间流传的非官方列表。

### 7.1 获取入侵指标与安全通报的渠道

- **[国家互联网应急中心 CNCERT/CC](https://www.cert.org.cn)** 发布网络安全事件通报、预警与处置建议，运营突发事件处置协调。
- **[CNVD（国家信息安全漏洞共享平台）](https://www.cnvd.org.cn)** 与 **[CNNVD（中国国家信息安全漏洞库）](https://www.cnnvd.org.cn)** 收录与通报漏洞信息，可作为 n-day 修补优先级的国内交叉输入（与 3.1 节的 CISA KEV 对照使用）。
- **[12377（中央网信办违法和不良信息举报中心）](https://www.12377.cn)** 与属地网信部门是违法网络活动的举报渠道。
- 行业性 ISAC（如金融行业的信息共享组织）与主管部门通报渠道，按所属行业接入。

第 5 节加固清单中「从可信来源获取官方 IoC」所指即此类渠道；即便官方 IoC 也应先核验有效期与误伤风险再落地（理由同第 2 节「为什么封 IP 是弱的一线防御」）。

### 7.2 法定报告义务

自主攻击蔓延快，请把法定报告步骤预先写入第 6 节的响应流程。以下为要点，具体适用范围、时限与要件以各机关最新规定为准：

- **网络运营者安全事件**：依据《网络安全法》及配套规程，发生危害网络安全的事件时应立即启动应急预案并按规定向有关主管部门报告；关键信息基础设施运营者另有更严格的报告要求（《关键信息基础设施安全保护条例》）。
- **个人信息泄露**：依据《个人信息保护法》第 57 条，发生或可能发生个人信息泄露、篡改、丢失时应立即采取补救措施，并通知履行个人信息保护职责的部门和个人；监管部门认为可能造成危害的，有权要求其通知个人。
- **行业监管**：金融、能源等行业另有向行业主管/监管机构报告的规定，请一并遵守。

实际提交入口：网络安全事件可通过 CNCERT/CC 渠口与属地通信管理局/网信部门；个人信息问题通过网信部门12377 及个人信息保护履行职责部门渠道。报告时限紧凑，请在第 6 节响应流程中预先写好责任人与联系方式。

---

## 参考

- 原始项目：[Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX)（AGPL-3.0）；本文档属于其韩文本地化仓库 [jiwoochris/artex-ko](https://github.com/jiwoochris/artex-ko) 防御资料的中文版。
- 上级 [README 的安全·滥用警告与使用范围、法律告知](../README.md)。
- 本版本面向中文用户；涉及个人信息的适用《个人信息保护法》《网络安全法》等。无论何地，未经授权的检测在多数司法辖区均属违法，务必先取得书面授权与约定范围。
- 通用 Web 安全加固标准参考：[OWASP Top 10](https://owasp.org/www-project-top-ten/)、[OWASP ASVS（应用安全验证标准）](https://owasp.org/www-project-application-security-verification-standard/)、[OWASP API Security Top 10](https://api-security.owasp.org/)。

> 本指南持续增补以服务防御与检测能力建设。检测规则与加固项的改进建议欢迎通过仓库 issue 提出。
