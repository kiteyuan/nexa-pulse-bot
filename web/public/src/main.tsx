import { StrictMode, useEffect, useMemo, useState, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";

type Theme = { id: number; name: string; slug: string };
type Item = {
  id: number;
  title: string;
  content: string;
  source: string;
  link?: string;
  created_at: string;
  media_paths: string[];
  themes?: Theme[];
};

type CardKind = "image" | "link" | "text";
type Section = { key: string; name: string; items: Item[]; total: number };
type Route = { view: "home" } | { view: "theme"; slug: string };

const HOME_BATCH = 5;
const THEME_PER = 48;
const LIST_MAX_ITEMS = 8;
const LIST_MIN_ITEMS = 4;

function randomListCount() {
  return LIST_MIN_ITEMS + Math.floor(Math.random() * (LIST_MAX_ITEMS - LIST_MIN_ITEMS + 1));
}

function formatTime(raw: string) {
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) {
    return "";
  }
  return d.toLocaleString("zh-CN", { month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" });
}

function excerpt(text: string, max = 110) {
  const t = text.replace(/\s+/g, " ").trim();
  if (t.length <= max) {
    return t;
  }
  return `${t.slice(0, max).trim()}…`;
}

function linkHost(url?: string) {
  if (!url) {
    return "";
  }
  try {
    return new URL(url).hostname.replace(/^www\./, "");
  } catch {
    return "";
  }
}

function cardKind(item: Item): CardKind {
  if (item.media_paths?.length) {
    return "image";
  }
  if (item.link) {
    return "link";
  }
  return "text";
}

function parseRoute(hash: string): Route {
  const raw = hash.replace(/^#/, "").replace(/^\//, "");
  const m = raw.match(/^t\/([^/?#]+)/);
  if (m?.[1]) {
    return { view: "theme", slug: decodeURIComponent(m[1]) };
  }
  return { view: "home" };
}

function useRoute(): Route {
  const [route, setRoute] = useState(() => parseRoute(window.location.hash));
  useEffect(() => {
    const onHash = () => setRoute(parseRoute(window.location.hash));
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);
  return route;
}

function goHome() {
  window.location.hash = "";
}

function goTheme(slug: string) {
  window.location.hash = `#/t/${encodeURIComponent(slug)}`;
}

function buildSections(themes: Theme[], items: Item[], limit?: number): Section[] {
  const bySlug = new Map<string, Item[]>();
  const uncategorized: Item[] = [];
  for (const item of items) {
    const tags = item.themes ?? [];
    if (tags.length === 0) {
      uncategorized.push(item);
      continue;
    }
    for (const t of tags) {
      const list = bySlug.get(t.slug) ?? [];
      list.push(item);
      bySlug.set(t.slug, list);
    }
  }
  const take = (list: Item[]) => (limit == null ? list : list.slice(0, limit));
  const ordered: Section[] = [];
  for (const t of themes) {
    const all = bySlug.get(t.slug) ?? [];
    if (all.length === 0) {
      continue;
    }
    ordered.push({ key: t.slug, name: t.name, items: take(all), total: all.length });
  }
  if (uncategorized.length > 0) {
    ordered.push({
      key: "_none",
      name: "未分类",
      items: take(uncategorized),
      total: uncategorized.length,
    });
  }
  return ordered;
}

function App() {
  const route = useRoute();
  const [themes, setThemes] = useState<Theme[]>([]);
  const [items, setItems] = useState<Item[]>([]);
  const [current, setCurrent] = useState<Item | null>(null);
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let stop = false;
    Promise.all([
      fetch("/api/themes").then((r) => r.json()) as Promise<Theme[]>,
      fetch("/api/items?limit=100").then((r) => r.json()) as Promise<Item[]>,
    ])
      .then(([t, list]) => {
        if (stop) {
          return;
        }
        setThemes(t ?? []);
        setItems(list ?? []);
        setErr("");
      })
      .catch(() => {
        if (!stop) {
          setErr("内容加载失败");
        }
      })
      .finally(() => {
        if (!stop) {
          setLoading(false);
        }
      });
    return () => {
      stop = true;
    };
  }, []);

  useEffect(() => {
    setCurrent(null);
    document.querySelector(".board")?.scrollTo(0, 0);
  }, [route]);

  useEffect(() => {
    if (!current) {
      return;
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setCurrent(null);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [current]);

  const homeSections = useMemo(() => buildSections(themes, items), [themes, items]);

  const themeSection = useMemo(() => {
    if (route.view !== "theme") {
      return null;
    }
    return buildSections(themes, items, THEME_PER).find((s) => s.key === route.slug) ?? null;
  }, [route, themes, items]);

  if (loading) {
    return (
      <div className="loading" role="status" aria-label="加载中">
        <span className="spinner" />
      </div>
    );
  }

  if (err) {
    return (
      <div className="page">
        <header className="hero home-hero">
          <div className="brand">
            <h1>NexaPulse</h1>
          </div>
        </header>
        <div className="board">
          <p className="banner">{err}</p>
        </div>
      </div>
    );
  }

  if (route.view === "theme") {
    return (
      <>
        <ThemePage section={themeSection} onOpen={setCurrent} />
        {current && <Detail item={current} onClose={() => setCurrent(null)} />}
      </>
    );
  }

  return (
    <>
      <HomePage sections={homeSections} onOpen={setCurrent} />
      {current && <Detail item={current} onClose={() => setCurrent(null)} />}
    </>
  );
}

function hotColumnCount() {
  if (window.innerWidth <= 560) {
    return 1;
  }
  if (window.innerWidth <= 900) {
    return 2;
  }
  return 3;
}

function useMobileHome() {
  const [mobile, setMobile] = useState(() => window.innerWidth <= 560);
  useEffect(() => {
    const mq = window.matchMedia("(max-width: 560px)");
    const apply = () => setMobile(mq.matches);
    apply();
    mq.addEventListener("change", apply);
    return () => mq.removeEventListener("change", apply);
  }, []);
  return mobile;
}

/** 列表最高约 8 行，用来把下一张卡片放进当前最短的列。 */
function packColumns(sections: Section[], cols: number): Section[][] {
  const buckets: Section[][] = Array.from({ length: cols }, () => []);
  const height = Array(cols).fill(0);
  for (const sec of sections) {
    let at = 0;
    for (let i = 1; i < cols; i++) {
      if (height[i] < height[at]) {
        at = i;
      }
    }
    buckets[at].push(sec);
    height[at] += 1 + Math.min(sec.items.length, 8);
  }
  return buckets;
}

function HomePage({ sections, onOpen }: { sections: Section[]; onOpen: (item: Item) => void }) {
  const mobile = useMobileHome();
  const [cols, setCols] = useState(hotColumnCount);
  useEffect(() => {
    const apply = () => setCols(hotColumnCount());
    window.addEventListener("resize", apply);
    return () => window.removeEventListener("resize", apply);
  }, []);
  const columns = useMemo(() => packColumns(sections, cols), [sections, cols]);

  return (
    <div className="page home">
      <header className="hero home-hero">
        <div className="brand">
          <img className="brand-mark" src="/icon.png" alt="" width={40} height={40} />
          <h1>NexaPulse</h1>
        </div>
        <a
          className="repo"
          href="https://github.com/kiteyuan/nexa-pulse-bot"
          target="_blank"
          rel="noreferrer"
          aria-label="GitHub"
        >
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <path
              fill="currentColor"
              d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"
            />
          </svg>
        </a>
      </header>

      {sections.length === 0 && <p className="banner">还没有内容</p>}

      <div className="board">
        <div className="hot-fall">
          {columns.map((col, i) => (
            <div className="hot-col" key={i}>
              {col.map((sec) => (
                <HotPanel key={sec.key} section={sec} onOpen={onOpen} compact={mobile} />
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function HotPanel({
  section,
  onOpen,
  compact,
}: {
  section: Section;
  onOpen: (item: Item) => void;
  compact: boolean;
}) {
  const [desktopLimit] = useState(randomListCount);
  const limit = compact ? HOME_BATCH : desktopLimit;
  const visible = section.items.slice(0, Math.min(limit, section.items.length));

  return (
    <section className="hot">
      <header className="hot-head">
        <h2>{section.name}</h2>
        <button type="button" className="more" onClick={() => goTheme(section.key)}>
          更多
        </button>
      </header>
      <ol className="hot-list">
        {visible.map((item, i) => (
          <li key={`${section.key}-${item.id}`}>
            <button type="button" className="hot-row" onClick={() => onOpen(item)}>
              <span className={`rank r${Math.min(i + 1, 4)}`}>{i + 1}</span>
              <span className="hot-title">{item.title}</span>
            </button>
          </li>
        ))}
      </ol>
    </section>
  );
}

function ThemePage({ section, onOpen }: { section: Section | null; onOpen: (item: Item) => void }) {
  if (!section) {
    return (
      <div className="page">
        <header className="hero theme-hero">
          <button type="button" className="back" onClick={goHome}>
            ← 全部栏目
          </button>
          <h1>栏目不存在</h1>
        </header>
        <div className="board">
          <p className="banner">该栏目暂无内容，或链接已失效。</p>
        </div>
      </div>
    );
  }

  return (
    <div className="page theme">
      <header className="hero theme-hero">
        <button type="button" className="back" onClick={goHome}>
          ← 全部栏目
        </button>
        <h1>{section.name}</h1>
        <p>{section.total} 条精选</p>
      </header>

      <div className="board">
        <div className="fall">
          {section.items.map((item) => (
            <Card key={item.id} item={item} onOpen={() => onOpen(item)} />
          ))}
        </div>
      </div>
    </div>
  );
}

function Card({ item, onOpen }: { item: Item; onOpen: () => void }) {
  const kind = cardKind(item);
  const time = formatTime(item.created_at);
  const host = linkHost(item.link);

  let body: ReactNode;
  if (kind === "image") {
    body = (
      <>
        <img className="cover" src={item.media_paths[0]} alt="" loading="lazy" />
        <div className="card-body">
          <strong>{item.title}</strong>
          <span className="meta">
            {item.source || "来源"}
            {time ? ` · ${time}` : ""}
          </span>
        </div>
      </>
    );
  } else if (kind === "link") {
    body = (
      <div className="card-body">
        <em className="host">{host || item.source || "链接"}</em>
        <strong>{item.title}</strong>
        {item.content && <p className="snip">{excerpt(item.content)}</p>}
        <span className="meta">
          {item.source || host}
          {time ? ` · ${time}` : ""}
        </span>
      </div>
    );
  } else {
    body = (
      <div className="card-body">
        <strong>{item.title}</strong>
        {item.content && <p className="snip">{excerpt(item.content, 90)}</p>}
        <span className="meta">
          {item.source || "来源"}
          {time ? ` · ${time}` : ""}
        </span>
      </div>
    );
  }

  return (
    <button type="button" className={`card ${kind}`} onClick={onOpen}>
      {body}
    </button>
  );
}

function Detail({ item, onClose }: { item: Item; onClose: () => void }) {
  useEffect(() => {
    const { body, documentElement: html } = document;
    const prev = {
      overflow: body.style.overflow,
      htmlOverflow: html.style.overflow,
    };
    html.style.overflow = "hidden";
    body.style.overflow = "hidden";
    return () => {
      body.style.overflow = prev.overflow;
      html.style.overflow = prev.htmlOverflow;
    };
  }, []);

  return (
    <div className="veil" role="presentation" onClick={onClose}>
      <article className="sheet" role="dialog" aria-label={item.title} onClick={(e) => e.stopPropagation()}>
        <header className="sheet-head">
          <p className="meta">
            {item.source || "来源"}
            {item.themes?.length ? ` · ${item.themes.map((t) => t.name).join("、")}` : ""}
            {item.created_at ? ` · ${formatTime(item.created_at)}` : ""}
          </p>
          <button type="button" className="x" onClick={onClose} aria-label="关闭">
            ×
          </button>
        </header>
        <h2>{item.title}</h2>
        {item.link && (
          <p className="sheet-link">
            <a href={item.link} target="_blank" rel="noreferrer">
              阅读原文
            </a>
          </p>
        )}
        {item.media_paths?.map((p) => (
          <img key={p} src={p} alt="" />
        ))}
        <div className="body">{item.content}</div>
      </article>
    </div>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
