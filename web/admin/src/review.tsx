import { useCallback, useEffect, useState } from "react";
import { api } from "./api";
import { when, type InboxItem, type InboxPayload, type InboxTab } from "./types";
import { Btn, Desk, Empty, Icon, Seg, Tags, useToast } from "./ui";

export function ReviewDesk() {
  const toast = useToast();
  const [tab, setTab] = useState<InboxTab>("pending");
  const [counts, setCounts] = useState({ pending: 0, published: 0, rejected: 0 });
  const [items, setItems] = useState<InboxItem[]>([]);
  const [current, setCurrent] = useState<number | null>(null);
  const load = useCallback(
    (next: InboxTab) =>
      api<InboxPayload>(`/api/inbox?tab=${next}&limit=60`)
        .then((data) => {
          setCounts(data.counts);
          const list = data.items ?? [];
          setItems(list);
          setCurrent((cur) => (cur && list.some((x) => x.id === cur) ? cur : list[0]?.id ?? null));
        })
        .catch((e: Error) => toast(e.message, "err")),
    [toast],
  );
  useEffect(() => {
    void load(tab);
  }, [tab, load]);

  const item = items.find((x) => x.id === current) ?? null;

  const decide = (id: number, action: "approve" | "reject") => {
    const idx = items.findIndex((x) => x.id === id);
    api(`/api/inbox/${id}`, { method: "PATCH", body: JSON.stringify({ action }) })
      .then(() => {
        const rest = items.filter((x) => x.id !== id);
        setItems(rest);
        const pick = rest[idx] ?? rest[idx - 1] ?? null;
        setCurrent(pick?.id ?? null);
        setCounts((c) => {
          const next = { ...c };
          if (tab === "pending") {
            next.pending = Math.max(0, next.pending - 1);
          }
          if (tab === "published") {
            next.published = Math.max(0, next.published - 1);
          }
          if (tab === "rejected") {
            next.rejected = Math.max(0, next.rejected - 1);
          }
          if (action === "approve") {
            next.published += 1;
          } else {
            next.rejected += 1;
          }
          return next;
        });
      })
      .catch((e: Error) => toast(e.message, "err"));
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return;
      }
      const idx = items.findIndex((x) => x.id === current);
      if (e.key === "j" || e.key === "ArrowDown") {
        e.preventDefault();
        setCurrent(items[Math.min(items.length - 1, idx + 1)]?.id ?? current);
      }
      if (e.key === "k" || e.key === "ArrowUp") {
        e.preventDefault();
        setCurrent(items[Math.max(0, idx - 1)]?.id ?? current);
      }
      if (!item) {
        return;
      }
      if (e.key === "a" && tab !== "published") {
        decide(item.id, "approve");
      }
      if ((e.key === "x" || e.key === "r") && tab !== "rejected") {
        decide(item.id, "reject");
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [items, current, tab, item]);

  const emptyHint =
    tab === "pending" ? "暂无待审" : tab === "published" ? "暂无在架内容" : "暂无弃稿";

  return (
    <Desk
      title="审核"
      actions={
        <Seg
          value={tab}
          onChange={setTab}
          items={[
            { id: "pending", label: `待审 ${counts.pending}` },
            { id: "published", label: `在架 ${counts.published}` },
            { id: "rejected", label: `弃稿 ${counts.rejected}` },
          ]}
        />
      }
    >
      {items.length === 0 ? (
        <Empty title={emptyHint} />
      ) : (
        <div className="desk-split review">
          <aside className="queue">
            {items.map((row) => (
              <button
                key={row.id}
                type="button"
                className={`queue-item ${row.id === current ? "on" : ""}`}
                onClick={() => setCurrent(row.id)}
              >
                <strong>{row.title}</strong>
                <span>
                  {row.source || "未知来源"} · {when(row.created_at)}
                </span>
              </button>
            ))}
          </aside>
          <article className="reader">
            {item ? (
              <>
                <header className="reader-top">
                  <div className="reader-meta">
                    <span>{item.source || "未知来源"}</span>
                    <span>{when(item.created_at)}</span>
                    <Tags items={item.themes} />
                  </div>
                  <h2>{item.title}</h2>
                  <div className="reader-actions">
                    {tab !== "published" && (
                      <Btn icon={<Icon name="check" />} onClick={() => decide(item.id, "approve")}>
                        通过 <kbd>A</kbd>
                      </Btn>
                    )}
                    {tab !== "rejected" && (
                      <Btn ghost icon={<Icon name="close" />} onClick={() => decide(item.id, "reject")}>
                        {tab === "published" ? "下架" : "弃稿"} <kbd>X</kbd>
                      </Btn>
                    )}
                    {item.link && (
                      <a className="btn ghost" href={item.link} target="_blank" rel="noreferrer">
                        <Icon name="external" />
                        <span>原文</span>
                      </a>
                    )}
                  </div>
                </header>
                <div className="reader-body">
                  {item.error_message && tab === "rejected" && <p className="fail">{item.error_message}</p>}
                  <div className="prose">{item.content}</div>
                </div>
              </>
            ) : (
              <Empty title="从左侧选择一条" />
            )}
          </article>
        </div>
      )}
    </Desk>
  );
}
