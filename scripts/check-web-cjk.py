#!/usr/bin/env python3
"""检查静态导出的 Web UI(`web/out`) HTML 是否残留韩文。

本仓库(ARTEX 中文版)的核心性质是"用户可见界面全部为中文、不残留韩文"
(继承自 ko 版的 `out` HTML 门禁，判定脚本反转：韩文版查汉字泄漏，中文版查
韩文残留)。这个性质若只靠人工验证，贡献者硬编码韩文字符串或翻译回退产生
回归时，会一直悄悄坏到有人肉眼发现为止。本脚本在合并门禁上拦住这种回归。

检查方式
---------
扫描 `web/out` 下所有 `.html` 文件的**原始内容**(不仅是渲染出的标记，还包括
内联脚本与 React Server Components flight 数据)，只要发现一个韩文字符就以
退出码 1 结束。之所以直接看原始内容，是因为用户可见文本会随 flight 数据
(JSON 形字符串)一起下发到 HTML 里；只剥标签会漏掉这条泄漏路径。

检查范围之外(诚实的局限)
-----------------------
该门禁只看静态导出产物，**不覆盖 MOCK 模式(`NEXT_PUBLIC_MOCK=1`)下浏览器
运行时渲染的内容**。`web/src/lib/mock/data.ts`·`handler.ts` 与各页面的硬编码
示例在水合后经 `useEffect` 通过 `mockHandle()` 进入 React 状态渲染，因此只
存在于运行时 JS bundle 而不在 `out` HTML 里。

这部分靠源码评审守住。渲染类文件(`web/src` 的 `.tsx`·客户端数据)中允许的
韩文仅限：代码注释、`t.has(...)` 屏蔽而实际不会使用的 i18n 回退、locale 为
`ko` 时才可见的分支字面量，或构建韩文版对照时有意保留的内容。

允许·检测范围
-------------
- **汉字(CJK)允许**。汉字块与韩文块在 Unicode 中互不重叠，只检测韩文块，
  汉字自动通过。
- **只检测韩文**：谚文音节(AC00–D7A3)、谚文字母(1100–11FF)、兼容谚文
  (3130–318F)。假名与全角标点不属于"韩文"，不在检测范围内(避免原文保留的
  代码块或证据引用引起误报)。

有意保留的韩文
--------------
若将来确实需要在界面保留韩文(例如语言切换器的 `한국어` 标签)，把该字符加入
下方 `ALLOWED_HANGUL` 并注释原因。这样门禁保持严格，只放行有意的例外。

仅用标准库，不访问网络。
运行: `python3 -I scripts/check-web-cjk.py [out 目录]`
(默认为仓库根的 `web/out`。CI 在 `web` 工作目录以
`python3 -I ../scripts/check-web-cjk.py` 调用。)
"""
import os
import re
import subprocess
import sys

# 仅韩文块：谚文音节(AC00–D7A3)、谚文字母(1100–11FF)、兼容谚文(3130–318F)。
# 不包含汉字、假名或全角标点。
HANGUL = re.compile("[\u1100-\u11ff\u3130-\u318f\uac00-\ud7a3]")

# 有意保留在界面上的韩文(如语言切换器 `한국어`)。目前为空 ——
# 为空时任何韩文都会被门禁拦下。添加例外时请写明字符与原因。
# 例) ALLOWED_HANGUL = {"한", "국", "어"}  # 语言切换器 '한국어' 标签(ko 版对照用)
ALLOWED_HANGUL: set = set()


def find_out_dir() -> str:
    """确定要检查的 `out` 目录。

    有参数就用参数路径，否则用 git 仓库根的 `web/out`。
    """
    if len(sys.argv) > 1:
        return os.path.abspath(sys.argv[1])
    root = subprocess.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True
    ).strip()
    return os.path.join(root, "web", "out")


def main() -> int:
    out_dir = find_out_dir()
    if not os.path.isdir(out_dir):
        print(
            f"错误：找不到待检查目录 —— {out_dir}\n"
            "请先执行 `npm run build:static` 生成静态导出后再运行。"
        )
        return 2

    checked = 0
    offenders = []  # (相对路径, 韩文字符数, 预览)
    for dirpath, _dirnames, filenames in os.walk(out_dir):
        for name in filenames:
            if not name.endswith(".html"):
                continue
            checked += 1
            path = os.path.join(dirpath, name)
            with open(path, encoding="utf-8", errors="ignore") as fh:
                text = fh.read()
            found = [ch for ch in HANGUL.findall(text) if ch not in ALLOWED_HANGUL]
            if found:
                rel = os.path.relpath(path, out_dir)
                kinds = "".join(sorted(set(found)))
                offenders.append((rel, len(found), kinds))

    print(f"已检查 HTML {checked} 个（目录：{out_dir}）")
    if offenders:
        print(f"韩文残留 {len(offenders)} 个文件 —— 用户可见 HTML 中存在韩文：")
        for rel, count, kinds in offenders:
            preview = kinds if len(kinds) <= 30 else kinds[:30] + "…"
            print(f"  out/{rel}: 韩文 {count} 个 · 种类 {preview}")
        print(
            "\n请翻译为中文；若确属有意保留的韩文，"
            "请在 scripts/check-web-cjk.py 的 ALLOWED_HANGUL 中附原因添加。"
        )
        return 1
    if checked == 0:
        print("警告：没有检查到任何 HTML —— 请确认构建是否成功。")
        return 2
    print("韩文残留 0 —— 用户可见 HTML 全部为中文/非韩文。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
