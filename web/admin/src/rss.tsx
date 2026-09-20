import { useEffect, useState } from "react";
import { api } from "./api";
import type { Feed } from "./types";
import { Btn, Desk, Empty, Field, Icon, IconBtn, Switch, Tags, Veil, useConfirm, useToast } from "./ui";

export function RssDesk() {
  const toast = useToast();
  const confirm = useConfirm();
  const [rows, setRows] = useState<Feed[]>([]);
  const [q, setQ] = useState("");
  const [draft, setDraft] = useState<Partial<Feed> | null>(null);

  const load = () =>
    api<Feed[]>("/api/feeds")
      .then((data) => setRows(data ?? []))
      .catch((e: Error) => toast(e.message, "err"));
  useEffect(() => void load(), []);

  const view = rows.filter((f) => `${f.name} ${f.url}`.toLowerCase().includes(q.trim().toLowerCase()));
  const live = rows.filter((f) => !f.disabled).length;
  const editing = draft && draft.id != null;

  return (
    <Desk
      title="RSS"
      actions={
        <Btn icon={<Icon name="plus" />} onClick={() => setDraft({ name: "", url: "" })}>
          添加
        </Btn>
      }
    >
      <section className="canvas solo">
        {rows.length > 0 && (
          <div className="canvas-tools">
            <label className="search">
              <Icon name="search" />
              <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="搜索名称或地址" />
            </label>
            <span className="meta">
              {live}/{rows.length} 采集中
            </span>
          </div>
        )}
        <div className="canvas-body">
          {view.length === 0 ? (
            <Empty
              title={rows.length === 0 ? "还没有订阅源" : "没有匹配"}
              action={
                rows.length === 0 ? (
                  <Btn icon={<Icon name="plus" />} onClick={() => setDraft({ name: "", url: "" })}>
                    添加
                  </Btn>
                ) : undefined
              }
            />
          ) : (
            <ul className="list">
              {view.map((f) => (
                <li className="list-row" key={f.id}>
                  <Switch
                    checked={!f.disabled}
                    label={f.disabled ? "开始采集" : "暂停采集"}
                    onChange={(on) => {
                      api(`/api/feeds/${f.id}`, { method: "PATCH", body: JSON.stringify({ disabled: !on }) })
                        .then(() => load())
                        .catch((e: Error) => toast(e.message, "err"));
                    }}
                  />
                  <button type="button" className="list-main hit" onClick={() => setDraft(f)}>
                    <strong>{f.name}</strong>
                    <div className="list-meta">
                      <cite>{f.url}</cite>
                      <Tags items={f.themes} />
                    </div>
                  </button>
                  <div className="row-actions">
                    <IconBtn label="编辑" onClick={() => setDraft(f)}>
                      <Icon name="edit" />
                    </IconBtn>
                    <IconBtn
                      label="删除"
                      danger
                      onClick={async () => {
                        if (!(await confirm(`删除订阅「${f.name}」？`, undefined, "删除"))) {
                          return;
                        }
                        api(`/api/feeds/${f.id}`, { method: "DELETE" })
                          .then(() => load())
                          .catch((e: Error) => toast(e.message, "err"));
                      }}
                    >
                      <Icon name="trash" />
                    </IconBtn>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>

      {draft && (
        <Veil title={editing ? "编辑订阅" : "添加订阅"} onClose={() => setDraft(null)}>
          <form
            className="stack"
            onSubmit={(e) => {
              e.preventDefault();
              const fd = new FormData(e.currentTarget);
              const name = String(fd.get("name") ?? "").trim();
              const url = String(fd.get("url") ?? "").trim();
              const req = editing
                ? api(`/api/feeds/${draft.id}`, { method: "PATCH", body: JSON.stringify({ name, url }) })
                : api("/api/feeds", { method: "POST", body: JSON.stringify({ name, url }) });
              req
                .then(() => {
                  setDraft(null);
                  return load();
                })
                .catch((err: Error) => toast(err.message, "err"));
            }}
          >
            <Field label="显示名称">
              <input name="name" required autoFocus defaultValue={draft.name ?? ""} placeholder="如 Hacker News" />
            </Field>
            <Field label="订阅地址">
              <input name="url" required defaultValue={draft.url ?? ""} placeholder="https://" />
            </Field>
            <div className="sheet-actions">
              <Btn type="submit" icon={editing ? <Icon name="check" /> : <Icon name="plus" />}>
                {editing ? "保存" : "添加"}
              </Btn>
            </div>
          </form>
        </Veil>
      )}
    </Desk>
  );
}
