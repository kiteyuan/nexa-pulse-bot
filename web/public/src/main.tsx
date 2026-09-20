import { StrictMode, useEffect, useMemo, useRef, useState, type ReactNode, type UIEvent } from "react";
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
    window.scrollTo(0, 0);
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

  if (loading || err) {
    return (
      <div className="page">
        <header className="hero">
          <h1>NexaPulse</h1>
        </header>
        <p className="banner">{loading ? "加载中…" : err}</p>
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

function HomePage({ sections, onOpen }: { sections: Section[]; onOpen: (item: Item) => void }) {
  return (
    <div className="page home">
      <header className="hero">
        <h1>NexaPulse</h1>
        <p>各栏目精选速览</p>
      </header>

      {sections.length === 0 && <p className="banner">还没有内容</p>}

      <div className="hot-fall">
        {sections.map((sec) => (
          <HotPanel key={sec.key} section={sec} onOpen={onOpen} />
        ))}
      </div>
    </div>
  );
}

function HotPanel({ section, onOpen }: { section: Section; onOpen: (item: Item) => void }) {
  const listRef = useRef<HTMLOListElement>(null);
  const [shown, setShown] = useState(() => Math.min(HOME_BATCH, section.items.length));
  const visible = section.items.slice(0, shown);
  const hasMore = shown < section.items.length;

  useEffect(() => {
    setShown(Math.min(HOME_BATCH, section.items.length));
  }, [section.key, section.items.length]);

  useEffect(() => {
    const el = listRef.current;
    if (!el || !hasMore) {
      return;
    }
    if (el.scrollHeight <= el.clientHeight + 4) {
      setShown((n) => Math.min(n + HOME_BATCH, section.items.length));
    }
  }, [shown, hasMore, section.items.length, visible.length]);

  const onScroll = (e: UIEvent<HTMLOListElement>) => {
    if (!hasMore) {
      return;
    }
    const el = e.currentTarget;
    if (el.scrollHeight - el.scrollTop - el.clientHeight <= 32) {
      setShown((n) => Math.min(n + HOME_BATCH, section.items.length));
    }
  };

  return (
    <section className="hot">
      <header className="hot-head">
        <h2>{section.name}</h2>
        <button type="button" className="more" onClick={() => goTheme(section.key)}>
          更多
        </button>
      </header>
      <ol ref={listRef} className="hot-list" onScroll={onScroll}>
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
        <p className="banner">该栏目暂无内容，或链接已失效。</p>
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

      <div className="fall">
        {section.items.map((item) => (
          <Card key={item.id} item={item} onOpen={() => onOpen(item)} />
        ))}
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
