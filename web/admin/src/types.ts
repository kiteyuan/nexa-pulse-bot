export type Page = "telegram" | "rss" | "columns" | "review" | "rules" | "journal";

export type Account = { id: number; name: string; status: string };
export type LoginState = { url: string; status: string; error: string };
export type ThemeTag = { id: number; name: string; slug: string };
export type ThemeSource = { source_kind: string; source_id: number; source_name: string };
export type Theme = ThemeTag & { sources?: ThemeSource[] };
export type Channel = {
  id: number;
  title: string;
  username: string;
  disabled: boolean;
  themes?: ThemeTag[];
};
export type Feed = { id: number; name: string; url: string; disabled: boolean; themes?: ThemeTag[] };
export type InboxTab = "pending" | "published" | "rejected";
export type InboxItem = {
  id: number;
  title: string;
  content: string;
  source: string;
  llm_status: string;
  error_message: string;
  created_at: string;
  link?: string;
  themes?: ThemeTag[];
};
export type InboxPayload = {
  counts: { pending: number; published: number; rejected: number };
  items: InboxItem[];
};
export type SettingsBody = {
  llm_enabled: boolean;
  llm_base_url: string;
  llm_api_key: string;
  llm_model: string;
  llm_temperature: number;
  translate_to: string;
  min_length: number;
  block_keywords: string[];
  poll_interval_seconds: number;
  collect_interval_seconds: number;
};
export type LogRow = { id: number; level: string; source: string; message: string; created_at: string };

export const pages: { id: Page; label: string; group?: string }[] = [
  { id: "telegram", label: "Telegram", group: "采集" },
  { id: "rss", label: "RSS", group: "采集" },
  { id: "columns", label: "栏目", group: "编排" },
  { id: "review", label: "审核", group: "编排" },
  { id: "rules", label: "规则", group: "系统" },
  { id: "journal", label: "日志", group: "系统" },
];

export function when(raw?: string) {
  if (!raw) {
    return "";
  }
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) {
    return "";
  }
  return d.toLocaleString("zh-CN", { month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" });
}
