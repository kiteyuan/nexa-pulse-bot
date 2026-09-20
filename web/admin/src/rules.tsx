import { useEffect, useState } from "react";
import { api } from "./api";
import type { SettingsBody } from "./types";
import { Desk, Field, Switch, Btn, Icon, useToast } from "./ui";

export function RulesDesk() {
  const toast = useToast();
  const [st, setSt] = useState<SettingsBody | null>(null);
  const [draft, setDraft] = useState<SettingsBody | null>(null);
  useEffect(() => {
    api<SettingsBody>("/api/settings")
      .then((data) => {
        setSt(data);
        setDraft(data);
      })
      .catch((e: Error) => toast(e.message, "err"));
  }, [toast]);

  if (!draft) {
    return <Desk title="规则" />;
  }

  const set = <K extends keyof SettingsBody>(key: K, value: SettingsBody[K]) => setDraft({ ...draft, [key]: value });

  return (
    <Desk
      title="规则"
      actions={
        <Btn
          icon={<Icon name="check" />}
          onClick={() => {
            api("/api/settings", { method: "PUT", body: JSON.stringify(draft) })
              .then(() => {
                setSt({ ...draft, llm_api_key: "" });
                setDraft({ ...draft, llm_api_key: "" });
                toast("已保存");
              })
              .catch((e: Error) => toast(e.message, "err"));
          }}
        >
          保存
        </Btn>
      }
    >
      <div className="rules">
        <section>
          <h2>过滤</h2>
          <div className="row2">
            <Field label="最短字数">
              <input type="number" value={draft.min_length} onChange={(e) => set("min_length", Number(e.target.value))} />
            </Field>
            <Field label="屏蔽词（逗号分隔）">
              <input
                value={draft.block_keywords.join(",")}
                onChange={(e) =>
                  set(
                    "block_keywords",
                    e.target.value
                      .split(/[,，]/)
                      .map((s) => s.trim())
                      .filter(Boolean),
                  )
                }
              />
            </Field>
          </div>
        </section>
        <section>
          <div className="section-head">
            <h2>语言模型</h2>
            <label className="inline-switch">
              <span>{draft.llm_enabled ? "已启用" : "已关闭"}</span>
              <Switch checked={draft.llm_enabled} onChange={(v) => set("llm_enabled", v)} label="启用 LLM" />
            </label>
          </div>
          <div className={`stack ${draft.llm_enabled ? "" : "dim"}`}>
            <Field label="接口地址">
              <input value={draft.llm_base_url} onChange={(e) => set("llm_base_url", e.target.value)} disabled={!draft.llm_enabled} />
            </Field>
            <Field label="API Key">
              <input
                value={draft.llm_api_key}
                onChange={(e) => set("llm_api_key", e.target.value)}
                placeholder={st?.llm_api_key || "未设置"}
                disabled={!draft.llm_enabled}
              />
            </Field>
            <div className="row2">
              <Field label="模型">
                <input value={draft.llm_model} onChange={(e) => set("llm_model", e.target.value)} disabled={!draft.llm_enabled} />
              </Field>
              <Field label="温度">
                <input
                  type="number"
                  step="0.1"
                  value={draft.llm_temperature}
                  onChange={(e) => set("llm_temperature", Number(e.target.value))}
                  disabled={!draft.llm_enabled}
                />
              </Field>
            </div>
            <Field label="翻译目标">
              <input
                value={draft.translate_to}
                onChange={(e) => set("translate_to", e.target.value)}
                placeholder="off / zh / en / ja / ko"
                disabled={!draft.llm_enabled}
              />
            </Field>
          </div>
        </section>
        <section>
          <h2>节奏</h2>
          <div className="row2">
            <Field label="处理间隔（秒）">
              <input
                type="number"
                value={draft.poll_interval_seconds}
                onChange={(e) => set("poll_interval_seconds", Number(e.target.value))}
              />
            </Field>
            <Field label="采集间隔（秒）">
              <input
                type="number"
                value={draft.collect_interval_seconds}
                onChange={(e) => set("collect_interval_seconds", Number(e.target.value))}
              />
            </Field>
          </div>
        </section>
      </div>
    </Desk>
  );
}
