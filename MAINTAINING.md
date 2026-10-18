# 上游同步与翻译漂移防护（维护者指南）

中文 · [English](MAINTAINING.en.md)

本文档整理**维护者**在追赶上游变更的同时维护中文本地化的流程。上游有两级：直接上游 [jiwoochris/artex-ko](https://github.com/jiwoochris/artex-ko)（韩文版），及其上游原版 [Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX)（已下架，以其最终源码为准）。贡献范围、法律责任、本地化方针见 [CONTRIBUTING.md](CONTRIBUTING.md)，用户侧说明见 [README.md](README.md)；本文只关注方针**如何实际执行**。

本地化的核心目标一句话：**保持 agent 判断性能原样，只把用户可见产出恢复为简体中文**。上游每次更新都容易扰动这条边界，以下流程与检查用于拦住翻译漂移。

---

## 1. 本地化结构一览

本仓库以 artex-ko 的源码快照为基线，在其上叠加中文化提交。追赶上游即「查看与上游的差异，区分保留与翻译对象后落盘」。

产物分三类：

- **保持原文的资产**（第 2 节）。翻译会破坏性能或上游对照。
- **代码固定的输出语言强制。** `agent/prompt.go` 的 `langDirective()` 给每个角色的 system 提示词末尾附加「用户可见输出用简体中文撰写」的指令。
- **译为中文的用户可见字符串。** UI 在 `web/messages/zh.json`，服务端用户可见文案在各 Go 文件的命名常量里。

---

## 2. 保持原文的资产（禁止翻译）

以下资产不翻译，保持原文（中文或英文）；上游变更触及它们时**原样落盘**。

- **agent 内部推理提示词（大脑正文）。** `agent/promptcatalog.go` 与数据库种子 `agent_prompts` 中的行为指令正文；必须保持经过基准的中文原文，翻译会引入判断漂移。
- **兼具展示与 agent 输入双重身份的字符串。** 既出现在活动时间线、又回灌为 planner/reporter 输入上下文的文案（任务中止原因、拦截阻断消息、流量证据辅助说明等）——一条记录两种用途，保持原文。逐案判定依据记录在 artex-ko 维护者的「大脑边界记录」（其 `work/DECISIONS-FOR-JIWOO.md`，不在本仓库内）。
- **韩文对照资源。** `web/messages/ko.json` 与 `README.ko.md`、`docs/defense-ko.md` 等 artex-ko 原文档保留原样，便于与直接上游对照；中文填充 `web/messages/zh.json` 与中文文档。
- **命令、payload、代码、URL、标识符、日志原文。** 分析需要原文，不翻译。Go 代码注释优先级最低，保留原样。

---

## 3. 上游追踪基线

需要配置上游 remote（直接上游）：

```bash
git remote add upstream https://github.com/jiwoochris/artex-ko
git remote -v   # 确认 upstream 存在
```

- **基线 = 2026-10-09 确认的 artex-ko 源码快照**（本仓库初始导入提交，含其时点的全部本地化状态）。

含义是「截至该快照的上游变更已全部溶入本仓库」。每次新落盘上游变更后，按第 7 节更新该基线。

---

## 4. 落盘上游变更的流程

### 4.1 拉取上游并查看差异

```bash
git fetch upstream
git log --oneline <基线>..upstream/main   # 未落盘提交列表
```

`git fetch` 只更新远程跟踪分支，不影响工作树与 `HEAD`。未落盘提交为 0 即已同步。

### 4.2 变更文件分类

```bash
git log --name-status --oneline <基线>..upstream/main
```

分类标准：

- `agent/promptcatalog.go`、`agent_prompts` 种子及第 2 节的展示兼输入字符串变更 → **原样落盘，不翻译**。
- Go 后端逻辑（`db/`、`llmrec/`、`server/` 等）变更 → 逻辑照常落盘；**新增的用户可见文案**（`writeErr` 等）确认为中文常量（上游若带来韩文/英文措辞，译为中文）。
- UI（`web/src/**`）新增界面字符串 → 不硬编码，以相同 key 同时写入 `web/messages/zh.json`（默认）与 `web/messages/ko.json`（对照）。
- **检测规则锚定的上游指标**（`enrich/enrich.go` 的探测 UA、`selfupdate/` 的自更新 UA、`guard/guard.go` 的审计标记、`db/db.go` 的破坏命令 deny 清单、`cmd/artex/main.go` 的默认监听与记录代理端口）变更 → 同步更新 `detections/` 的 Sigma/Suricata 规则、ATT&CK 层与 `detections/indicators/artex_indicators.csv` 的取值。这些指标不是翻译对象而是**检测依据**，上游改值后规则会静默过期；5.4 的指标一致性测试会自动抓这种漂移。

### 4.3 落盘

按功能合并或择优移植后，把 4.2 挑出的新字符串译为中文。合并过程容易造成 `zh.json`/`ko.json` key 不齐或原文漏进用户可见位置，落盘后立即跑第 5 节检查。

---

## 5. 翻译对称与漂移检查

上游落盘或翻译工作之后，核对以下各项。

### 5.1 zh ↔ ko 消息对称与韩文残留

`zh.json` 与 `ko.json` 的 key 完全一致，且 `zh.json` 值中不得残留韩文：

```bash
python3 - <<'PY'
import json, re
zh = json.load(open('web/messages/zh.json'))
ko = json.load(open('web/messages/ko.json'))
def flatten(d, p=''):
    out = {}
    if isinstance(d, dict):
        for k, v in d.items(): out.update(flatten(v, p + '/' + k))
    elif isinstance(d, list):
        for i, v in enumerate(d): out.update(flatten(v, p + '/' + str(i)))
    else: out[p] = d
    return out
fz, fk = flatten(zh), flatten(ko)
hangul = re.compile('[\uac00-\ud7af]')
print('zh leaf keys :', len(fz))
print('ko leaf keys :', len(fk))
print('key symdiff  :', len(set(fz) ^ set(fk)))                          # 必须为 0
print('zh vals w/Hangul:', sum(1 for v in fz.values() if isinstance(v, str) and hangul.search(v)))  # 必须为 0
PY
```

基线值（本仓库导入时）：`zh leaf keys = 2952`，`ko leaf keys = 2952`，`key symdiff = 0`，`zh vals w/Hangul = 0`。key 数可随上游落盘增长，但 zh 与 ko 必须一致，后两项恒为 0。

### 5.2 大脑资产原文保全

大脑正文保持中文原文；下列检查的**中文行数若掉到 0 附近，反而是大脑被误译污染的信号**：

```bash
python3 -c "import re; han=re.compile(r'[㐀-鿿]'); t=open('agent/promptcatalog.go').read(); print('promptcatalog.go CJK lines =', sum(1 for l in t.splitlines() if han.search(l)))"
```

该数值大幅下降时，检查大脑正文是否被翻译。

### 5.3 构建产物不残留韩文

UI 静态导出后，预渲染 HTML 中出现韩文即翻译缺失：

```bash
cd web && npm ci && NEXT_EXPORT=1 npm run build   # 生成 out/
python3 -I ../scripts/check-web-cjk.py            # 断言 out/ HTML 韩文为 0
```

### 5.4 检测指标与上游源码仍一致

`detections/` 规则锚定源码实际发出的字符串（探测 UA、自更新 UA、审计标记、破坏命令 deny 清单）。上游再同步改值后，翻译检查全部通过、已部署规则却静默失配。以下测试双向核对每个指标在源码与规则两侧仍然存在，再同步后必须运行：

```bash
detections/tests/indicators/run.sh   # Docker 隔离运行，RESULT: PASS 即一致
```

失败时输出哪个指标、朝哪个方向失配（源码变了还是规则变了），按 4.2 最后一条把规则/层对齐新值。该测试也由仓库 CI（[`.github/workflows/detections.yml`](.github/workflows/detections.yml)）在规则树或被锚定源文件变更的推送/PR 上自动运行，把再同步漂移拦在合并门禁。

新增指标并**锚定新的源文件**时（类似 `cmd/artex/main.go` 的端口指标），务必把该文件加进上述工作流的 `push`/`pull_request` `paths` 过滤；漏加则只改该源的 PR 不会触发指标测试，漂移会静默通过门禁。这个同步关系本身也被指标测试的第五项检查（"CI triggers this test when any pinned source changes"）强制。

### 5.5 升级检测测试工具钉版本时

检测测试以固定版本运行 `sigma-cli`、SigmaHQ 校验器插件（`pySigma-validators-sigmahq`）与 Suricata 镜像（各 `run.sh` 默认值，可用环境变量覆盖）。升级钉版本可能带来**工具侧漂移**：SigmaHQ 校验器升级常新增惯例检查，`detections/tests/sigma_lint/run.sh` 会把新 issue 染红——此时按新惯例修规则，或对不适用于独立规则集的惯例在 [`detections/tests/sigma_lint/validators.yml`](detections/tests/sigma_lint/validators.yml) 记录理由后加入排除。后端插件支持变化时 `sigma_backends` 测试会给出同样信号。

---

## 6. 构建与测试收尾验证

落盘/翻译完成后按 [CONTRIBUTING.md 的开发环境](CONTRIBUTING.md#开发环境)验证后端与前端。本地无 Go 时用 Docker 等价运行：

```bash
docker run --rm -v "$PWD":/src -w /src \
  -v artexzh-gomod:/go/pkg/mod -v artexzh-gocache:/root/.cache/go-build \
  golang:1.26 sh -c 'go build ./... && go vet ./... && go test ./... -count=1'
```

翻译用户可见文案时，同步维护断言该文案的回归测试（`*_localized_test.go`：含中文、无韩文），让上游变更再带回外文时能被测试拦住。翻译验证务必使用能力足够的（前沿级）模型——小模型输出可能回退原文，不能只凭其输出判断翻译是否生效。

---

## 7. 基线更新记录

落盘上游变更并完成验证后，把第 3 节的**基线值更新为新上游提交**，并在同一或紧随的提交中带上本文变更，让下一位维护者只看本文一处就知道「同步到哪了」。

提交信息遵循 [CONTRIBUTING.md 的提交规范](CONTRIBUTING.md#提交信息)，例如：

```
chore(upstream): 落盘上游 artex-ko 快照更新（xxx）+ 新增 UI 字符串中文化
```

---

## 8. 审阅·验证守则（常见陷阱）

### 8.1 查仓库 CI 状态必须显式指定仓库

本仓库继承自上游 fork 链；本地 `git remote` 可能同时挂有 `origin`（AFOGSHEEP/artex-zh）与 `upstream`（jiwoochris/artex-ko）。此时 `gh` 不指定仓库会**默认选错上游**，显示对方（没有我们工作流的）运行结果，导致把上游 CI 绿了误当本仓库通过、或把我们自己的 `ci.yml`/`detections.yml` 误判为 "HTTP 404 … not found"。

```bash
gh run list -R AFOGSHEEP/artex-zh --workflow ci.yml --limit 5
gh run list -R AFOGSHEEP/artex-zh --workflow detections.yml --limit 5

gh repo set-default AFOGSHEEP/artex-zh
gh repo set-default --view   # 确认默认仓库
```

（`gh repo set-default` 是本地 gh 配置，不入库；新机器/新检出需重设。）

### 8.2 文档外链要用浏览器式 GET 验证

防御指南（[`docs/defense-zh.md`](docs/defense-zh.md) 等）第 7 节引用官方渠道（cert.org.cn、cnvd.org.cn、cnnvd.org.cn、12377.cn、cisa.gov 等）。用 `curl -I`（HEAD）或默认 User-Agent 检查容易**把好链接误判成失效**：政府/安全机构站点常拒绝 HEAD、拦截默认 curl UA、或多次重定向。因此外链检查要**浏览器 UA、GET、跟随重定向**：

```bash
UA='Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0 Safari/537.36'
for u in https://www.cert.org.cn https://www.cnvd.org.cn https://www.cnnvd.org.cn; do
  curl -sS -L -A "$UA" -o /dev/null -w "$u -> %{http_code} %{url_effective}\n" "$u"
done
```

最终状态码 200 即有效；出现 400/403 先怀疑是**检查方式被访问策略拦下**而非链接失效，逐一排除 HEAD、默认 UA、未跟随重定向等因素。（原韩文版记录的 boho.or.kr、fsec.or.kr、pipc.go.kr 同理。）

该人工流程由 `scripts/check-external-links.py` 自动化：收集所有被跟踪 `.md` 中代码块/行内代码之外的外链（排除保留字/占位主机），以浏览器 UA、GET、重定向跟随检查，网络错误/5xx/429 重试以区分瞬时抖动与真故障。结果分四类：OK（2xx/3xx）、RESTRICTED（401/403/405/429，主机存活仅检查方式被拦）、ALLOWED（记录于 `scripts/external-links-allowlist.txt`、我们无法修复的上游继承死链）、DOWN（404/410/5xx/连接错误，大概率真失效）。

- 无网络预览检查对象：`python3 -I scripts/check-external-links.py --list`
- 发布/周期检查（新增 DOWN 即非零退出）：`python3 -I scripts/check-external-links.py --strict`

外链存活是 flaky 的，**不进合并门禁**；由非阻塞工作流 [`external-links`](.github/workflows/external-links.yml) 每周一及手动运行 `--strict`，allowlist 之外新出现 DOWN 时才染红。上游保留文件继承的死链（如 `CHANGELOG.zh.md` 致谢的已消失账号）记入 allowlist 并附理由。

---

## 9. 发布流水线

推送版本标签（`v*`）会触发 [`.github/workflows/release.yml`](.github/workflows/release.yml)：产出五平台二进制，条件满足时再构建多架构 Docker 镜像。本仓库尚未发布过任何 Release（一键更新在此前会提示「尚未发布任何正式版本」）；切断首个版本属于发布决策之后的事。

### 9.1 如何发版

```bash
git tag v0.3.15-zh.1
git push origin v0.3.15-zh.1
```

### 9.2 流水线内容

工作流分五个 job：**frontend**（静态导出一次，产物作为 `web-dist` artifact）；**binaries**（交叉编译 linux amd64/arm64、darwin amd64/arm64、windows amd64 并打 zip，linux amd64 跑 `artex -h` 冒烟）；**release**（汇总 zip 生成 `SHA256SUMS` 并创建 GitHub Release 附件）；**docker-gate**（检查 `DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN` secrets 并作为下一 job 条件）；**docker**（仅当 secrets 存在时基于交叉编译产物构建多架构镜像推 Docker Hub，否则跳过且不红）。

### 9.3 构建前提与仓库结构的吻合

流水线三个前提与当前仓库结构一致（binaries job 已在本地复现验证）：前端产物经 `server/webui/dist` 由 `//go:embed all:webui/dist` 内嵌（binaries 以 `ARTEX_SKIP_FRONTEND=1` 复用，不重复构建前端）；`build.sh --target <os>/<arch>` 产出 `dist/artex-<os>-<arch>/artex` 与含启动脚本、`skills/`、`config.example.json`、`README.md` 的 zip；Docker job 经 `bin-linux-<arch>` artifact 与 Dockerfile 的 `COPY dist/${TARGETARCH}/artex` 对接（`.dockerignore` 不排除 `dist/`）。

### 9.4 待决事项：Docker 镜像命名空间

docker job 当前仍指向上游的 `autumn27/artex`（已下架）。本仓库以哪个命名空间发布镜像待定；决定之前不配置 Docker Hub secrets，发布仅产出二进制 zip 与校验和（docker job 跳过）。

### 9.5 不打标签的本地预验证

```bash
# 1) 前端静态导出（对应 frontend job）
cd web && npm ci && npm run build:static && cd ..
# 2) 模拟 binaries job 接收 artifact 的位置
rm -rf server/webui/dist && mkdir -p server/webui/dist && cp -a web/out/. server/webui/dist/
# 3) 以与 binaries job 相同的环境构建单目标
docker run --rm -v "$PWD":/app -w /app \
  -e ARTEX_SKIP_FRONTEND=1 -e ARTEX_SKIP_NPM_CI=1 \
  -e ARTEX_COMPRESS=0 -e ARTEX_PACKAGE=1 -e ARTEX_PACKAGE_DIR=dist \
  -e ARTEX_BUILD_VERSION=v0.0.0-local \
  golang:1.26 bash -c 'apt-get update && apt-get install -y zip && ./build.sh --target linux/amd64'
# 4) 检查产物：dist/artex-linux-amd64/artex · dist/*.zip · dist/SHA256SUMS
```

`dist/artex-linux-amd64/artex` 为静态链接 ELF，`-h` 打印用法后以退出码 0 结束（binaries job 冒烟测试验证的行为）。构建产物（`dist/`、`server/webui/dist/`）不入库（`.gitignore` 已排除）。
