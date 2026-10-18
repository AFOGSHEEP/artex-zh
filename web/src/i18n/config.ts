// 支持的 locale 与默认值。本仓库是 ARTEX 中文版，默认 locale 为 "zh"，
// 并保留韩语 "ko" 以便与上游韩文版（artex-ko）对照。
export const LOCALES = ["zh", "ko"] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "zh";

// 激活 locale 在构建期决定。为兼容静态导出（next.config 的 output: "export"），
// 不使用 cookies()/headers() 等动态 API，只读环境变量。
// NEXT_PUBLIC_LOCALE 为空或不在支持列表内时回落到默认值(zh)。
export function resolveLocale(): Locale {
  const raw = process.env.NEXT_PUBLIC_LOCALE;
  return LOCALES.includes(raw as Locale) ? (raw as Locale) : DEFAULT_LOCALE;
}
