import { useEffect, useMemo, useState } from "react";
import { api } from "./api";
import type { Channel, Feed, Theme } from "./types";
import {
  Btn,
  Desk,
  Empty,
  Field,
  Icon,
  IconBtn,
  Menu,
  MenuItem,
  Veil,
  cleanSourceName,
  useConfirm,
  useToast,
} from "./ui";

type PoolItem = {
  key: string;
  kind: "telegram" | "rss";
  sid: number;
  name: string;
  mark: "TG" | "RSS";
};

export function ColumnsDesk() {
  const toast = useToast();
  const confirm = useConfirm();
  const [themes, setThemes] = useState<Theme[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [feeds, setFeeds] = useState<Feed[]>([]);
  const [id, setId] = useState(0);
  const [q, setQ] = useState("");
  const [creating, setCreating] = useState(false);
  const [renaming, setRenaming] = useState(false);

  const load = () =>
    Promise.all([api<Theme[]>("/api/themes"), api<Channel[]>("/api/channels"), api<Feed[]>("/api/feeds")])
      .then(([t, c, f]) => {
        setThemes(t ?? []);
        setChannels(c ?? []);
        setFeeds(f ?? []);
        setId((cur) => (cur && t.some((x) => x.id === cur) ? cur : t[0]?.id ?? 0));
      })
      .catch((e: Error) => toast(e.message, "err"));

  useEffect(() => void load(), []);

  const theme = themes.find((t) => t.id === id);
  const bound = useMemo(() => new Set((theme?.sources ?? []).map((s) => `${s.source_kind}:${s.source_id}`)), [theme]);
  const pool: PoolItem[] = [
    ...channels
      .filter((c) => !c.disabled)
      .map((c) => ({
        key: `telegram:${c.id}`,
        kind: "telegram" as const,
        sid: c.id,
        name: cleanSourceName(c.title || c.username),
        mark: "TG" as const,
      })),
    ...feeds
      .filter((f) => !f.disabled)
      .map((f) => ({ key: `rss:${f.id}`, kind: "rss" as const, sid: f.id, name: f.name, mark: "RSS" as const })),
  ].filter((s) => !bound.has(s.key) && s.name.toLowerCase().includes(q.trim().toLowerCase()));

  const removeTheme = async (t: Theme) => {
    if (!(await confirm(`删除栏目「${t.name}」？`, "来源绑定会解开，已入库文章仍保留。", "删除"))) {
      return;
    }
    api(`/api/themes/${t.id}`, { method: "DELETE" })
      .then(() => load())
      .catch((e: Error) => toast(e.message, "err"));
  };

  const index = themes.findIndex((t) => t.id === id);
  const moveTheme = (dir: "up" | "down") => {
    if (!theme) {
      return;
    }
    api(`/api/themes/${theme.id}/move`, { method: "POST", body: JSON.stringify({ dir }) })
      .then(() => load())
      .catch((e: Error) => toast(e.message, "err"));
  };

  return (
    <Desk
      title="栏目"
      actions={
        <>
          {theme && themes.length > 1 && (
            <>
              <IconBtn label="在用户端提前" disabled={index <= 0} onClick={() => moveTheme("up")}>
                <Icon name="up" />
              </IconBtn>
              <IconBtn label="在用户端靠后" disabled={index < 0 || index >= themes.length - 1} onClick={() => moveTheme("down")}>
                <Icon name="down" />
              </IconBtn>
            </>
          )}
          {theme && themes.length > 1 && (
            <Menu label={<span className="menu-label">{theme.name}</span>}>
              {(close) =>
                themes.map((t) => (
                  <MenuItem
                    key={t.id}
                    onClick={() => {
                      setId(t.id);
                      close();
                    }}
                  >
                    <span>{t.name}{t.id === id ? " · 当前" : ""}</span>
                    <em className="menu-count">{t.sources?.length ?? 0}</em>
                  </MenuItem>
                ))
              }
            </Menu>
          )}
          {theme && themes.length <= 1 && <span className="chip">{theme.name}</span>}
          <Btn ghost icon={<Icon name="plus" />} onClick={() => setCreating(true)}>
            新建
          </Btn>
          {theme && (
            <IconBtn label="重命名" onClick={() => setRenaming(true)}>
              <Icon name="edit" />
            </IconBtn>
          )}
          {theme && (
            <IconBtn label="删除栏目" danger onClick={() => void removeTheme(theme)}>
              <Icon name="trash" />
            </IconBtn>
          )}
        </>
      }
    >
      <section className="canvas solo">
        {!theme ? (
          <Empty
            title="还没有栏目"
            action={
              <Btn icon={<Icon name="plus" />} onClick={() => setCreating(true)}>
                新建栏目
              </Btn>
            }
          />
        ) : (
          <div className="curate">
            <div className="curate-pane">
              <header className="curate-head">
                <h3>已绑定</h3>
                <span>{theme.sources?.length ?? 0}</span>
              </header>
              <div className="curate-scroll">
                {(theme.sources ?? []).length === 0 ? (
                  <p className="quiet pad">从右侧加入来源</p>
                ) : (
                  <ul className="curate-list">
                    {(theme.sources ?? []).map((s) => (
                      <li key={`${s.source_kind}-${s.source_id}`}>
                        <button
                          type="button"
                          className="curate-row"
                          onClick={() =>
                            api(`/api/themes/${theme.id}/sources?kind=${s.source_kind}&source_id=${s.source_id}`, {
                              method: "DELETE",
                            })
                              .then(() => load())
                              .catch((e: Error) => toast(e.message, "err"))
                          }
                        >
                          <b>{s.source_kind === "telegram" ? "TG" : "RSS"}</b>
                          <span>{cleanSourceName(s.source_name)}</span>
                          <Icon name="minus" />
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
            <div className="curate-pane">
              <header className="curate-head">
                <h3>可加入</h3>
                <span>{pool.length}</span>
              </header>
              <label className="search block">
                <Icon name="search" />
                <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="搜索来源" />
              </label>
              <div className="curate-scroll">
                {pool.length === 0 ? (
                  <p className="quiet pad">{q.trim() ? "没有匹配" : "没有可加入的来源"}</p>
                ) : (
                  <ul className="curate-list">
                    {pool.map((s) => (
                      <li key={s.key}>
                        <button
                          type="button"
                          className="curate-row add"
                          onClick={() =>
                            api(`/api/themes/${theme.id}/sources`, {
                              method: "POST",
                              body: JSON.stringify({ source_kind: s.kind, source_id: s.sid }),
                            })
                              .then(() => load())
                              .catch((e: Error) => toast(e.message, "err"))
                          }
                        >
                          <b>{s.mark}</b>
                          <span>{s.name}</span>
                          <Icon name="plus" />
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
          </div>
        )}
      </section>

      {creating && (
        <Veil title="新建栏目" onClose={() => setCreating(false)}>
          <form
            className="stack"
            onSubmit={(e) => {
              e.preventDefault();
              const name = new FormData(e.currentTarget).get("name")?.toString().trim() ?? "";
              if (!name) {
                return;
              }
              api<Theme>("/api/themes", { method: "POST", body: JSON.stringify({ name }) })
                .then((created) => {
                  setCreating(false);
                  return load().then(() => setId(created.id));
                })
                .catch((err: Error) => toast(err.message, "err"));
            }}
          >
            <Field label="名称">
              <input name="name" required autoFocus placeholder="如 科技" />
            </Field>
            <div className="sheet-actions">
              <Btn type="submit" icon={<Icon name="plus" />}>
                创建
              </Btn>
            </div>
          </form>
        </Veil>
      )}

      {renaming && theme && (
        <Veil title="重命名栏目" onClose={() => setRenaming(false)}>
          <form
            className="stack"
            onSubmit={(e) => {
              e.preventDefault();
              const name = new FormData(e.currentTarget).get("name")?.toString().trim() ?? "";
              if (!name) {
                return;
              }
              api<Theme>(`/api/themes/${theme.id}`, { method: "PATCH", body: JSON.stringify({ name }) })
                .then(() => {
                  setRenaming(false);
                  return load();
                })
                .catch((err: Error) => toast(err.message, "err"));
            }}
          >
            <Field label="名称">
              <input name="name" required autoFocus defaultValue={theme.name} />
            </Field>
            <div className="sheet-actions">
              <Btn type="submit" icon={<Icon name="check" />}>
                保存
              </Btn>
            </div>
          </form>
        </Veil>
      )}
    </Desk>
  );
}
