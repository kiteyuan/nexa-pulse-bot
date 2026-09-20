import { useEffect, useState } from "react";
import { getToken, setToken } from "./api";
import { ColumnsDesk } from "./columns";
import { JournalDesk } from "./journal";
import { ReviewDesk } from "./review";
import { RssDesk } from "./rss";
import { RulesDesk } from "./rules";
import { TelegramDesk } from "./telegram";
import { pages, type Page } from "./types";
import { Feedback, Glyph } from "./ui";

function parsePage(hash: string): Page {
  const id = hash.replace(/^#\/?/, "") as Page;
  return pages.some((p) => p.id === id) ? id : "telegram";
}

function navBlocks() {
  const blocks: { group?: string; items: typeof pages }[] = [];
  for (const p of pages) {
    const last = blocks[blocks.length - 1];
    if (last && last.group === p.group) {
      last.items.push(p);
    } else {
      blocks.push({ group: p.group, items: [p] });
    }
  }
  return blocks;
}

export function App() {
  const [token, setTok] = useState(getToken());
  const [page, setPage] = useState<Page>(() => parsePage(window.location.hash));
  useEffect(() => {
    const sync = () => setPage(parsePage(window.location.hash));
    if (!window.location.hash) {
      window.location.hash = "telegram";
    }
    window.addEventListener("hashchange", sync);
    return () => window.removeEventListener("hashchange", sync);
  }, []);
  const go = (id: Page) => {
    window.location.hash = id;
    setPage(id);
  };
  if (!token) {
    return <Gate onIn={(v) => { setToken(v); setTok(v); }} />;
  }

  return (
    <Feedback>
      <div className="shell">
        <aside className="rail">
          <div className="mark">Nexa</div>
          <nav>
            {navBlocks().map((block) => (
              <div key={block.group ?? "root"} className="rail-block">
                {block.group && <div className="rail-group">{block.group}</div>}
                {block.items.map((p) => (
                  <button key={p.id} type="button" className={page === p.id ? "on" : ""} onClick={() => go(p.id)}>
                    <Glyph name={p.id} />
                    {p.label}
                  </button>
                ))}
              </div>
            ))}
          </nav>
          <button
            type="button"
            className="out"
            onClick={() => {
              setToken("");
              setTok("");
            }}
          >
            退出
          </button>
        </aside>
        <div className="stage">
          {page === "telegram" && <TelegramDesk />}
          {page === "rss" && <RssDesk />}
          {page === "columns" && <ColumnsDesk />}
          {page === "review" && <ReviewDesk />}
          {page === "rules" && <RulesDesk />}
          {page === "journal" && <JournalDesk />}
        </div>
      </div>
    </Feedback>
  );
}

function Gate({ onIn }: { onIn: (token: string) => void }) {
  return (
    <div className="lock">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onIn(new FormData(e.currentTarget).get("token")?.toString() ?? "");
        }}
      >
        <h1>Nexa</h1>
        <label>
          令牌
          <input name="token" type="password" required autoComplete="current-password" autoFocus />
        </label>
        <button type="submit" className="btn">
          进入
        </button>
      </form>
    </div>
  );
}
