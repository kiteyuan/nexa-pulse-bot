import { useEffect, useState } from "react";
import { api } from "./api";
import { when, type LogRow } from "./types";
import { Btn, Desk, Empty, Icon, Seg, useToast } from "./ui";

export function JournalDesk() {
  const toast = useToast();
  const [rows, setRows] = useState<LogRow[]>([]);
  const [level, setLevel] = useState<"all" | "ERROR" | "INFO">("all");
  const load = () =>
    api<LogRow[]>("/api/logs?limit=80")
      .then((data) => setRows(data ?? []))
      .catch((e: Error) => toast(e.message, "err"));
  useEffect(() => {
    void load();
  }, []);
  const view = rows.filter((r) => level === "all" || r.level === level);
  return (
    <Desk
      title="日志"
      actions={
        <>
          <Seg
            value={level}
            onChange={setLevel}
            items={[
              { id: "all", label: "全部" },
              { id: "ERROR", label: "错误" },
              { id: "INFO", label: "信息" },
            ]}
          />
          <Btn ghost icon={<Icon name="refresh" />} onClick={() => void load()}>
            刷新
          </Btn>
        </>
      }
    >
      <section className="canvas solo">
        <div className="canvas-body">
          {view.length === 0 ? (
            <Empty title="暂无日志" />
          ) : (
            <ol className="log">
              {view.map((l) => (
                <li key={l.id}>
                  <time>{when(l.created_at)}</time>
                  <b className={l.level === "ERROR" ? "bad" : ""}>{l.level}</b>
                  <em>{l.source}</em>
                  <span>{l.message}</span>
                </li>
              ))}
            </ol>
          )}
        </div>
      </section>
    </Desk>
  );
}
