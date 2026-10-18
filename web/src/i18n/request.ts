import { getRequestConfig } from "next-intl/server";

import { type Locale, resolveLocale } from "./config";

// locale 各自的消息加载器。用显式 import 映射，避免打包器把 messages/ 整个目录
// 拉进 context 模块（也排除了仅存在于本地的 zh.sources.json）。
const loaders: Record<Locale, () => Promise<{ default: Record<string, unknown> }>> = {
  zh: () => import("../../messages/zh.json"),
  ko: () => import("../../messages/ko.json"),
};

// next-intl 请求配置。本项目不使用 i18n 路径路由（segment·middleware），
// locale 在这里直接决定。不引用 requestLocale，因此不会触发动态渲染；
// 中文在 messages/zh.json，韩语保留在 messages/ko.json（上游韩文版对照用）。
export default getRequestConfig(async () => {
  const locale = resolveLocale();
  const messages = (await loaders[locale]()).default;
  return { locale, messages };
});
