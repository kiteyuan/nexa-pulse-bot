import { useCallback, useEffect, useState } from "react";
import QRCode from "qrcode";
import { api } from "./api";
import type { Account, Channel, LoginState } from "./types";
import {
  Btn,
  Desk,
  Empty,
  Field,
  Icon,
  IconBtn,
  Menu,
  MenuItem,
  Seg,
  Switch,
  Tags,
  Veil,
  cleanSourceName,
  useConfirm,
  useToast,
} from "./ui";

export function TelegramDesk() {
  const toast = useToast();
  const confirm = useConfirm();
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [accountId, setAccountId] = useState(0);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [q, setQ] = useState("");
  const [filter, setFilter] = useState<"all" | "on" | "off">("all");
  const [addOpen, setAddOpen] = useState(false);
  const [loginFor, setLoginFor] = useState<Account | null>(null);
  const [syncing, setSyncing] = useState(false);

  const loadAccounts = useCallback(async () => {
    const rows = (await api<Account[]>("/api/accounts")) ?? [];
    setAccounts(rows);
    setAccountId((cur) => (cur && rows.some((a) => a.id === cur) ? cur : rows[0]?.id ?? 0));
  }, []);

  const loadChannels = useCallback(async (id: number) => {
    if (!id) {
      setChannels([]);
      return;
    }
    setChannels((await api<Channel[]>(`/api/channels?account_id=${id}`)) ?? []);
  }, []);

  useEffect(() => {
    void loadAccounts().catch((e: Error) => toast(e.message, "err"));
  }, [loadAccounts, toast]);

  useEffect(() => {
    void loadChannels(accountId).catch((e: Error) => toast(e.message, "err"));
  }, [accountId, loadChannels, toast]);

  const view = channels.filter((c) => {
    if (filter === "on" && c.disabled) {
      return false;
    }
    if (filter === "off" && !c.disabled) {
      return false;
    }
    return `${c.title} ${c.username}`.toLowerCase().includes(q.trim().toLowerCase());
  });
  const live = channels.filter((c) => !c.disabled).length;
  const current = accounts.find((a) => a.id === accountId);

  const sync = () => {
    if (!accountId || syncing) {
      return;
    }
    setSyncing(true);
    api("/api/channels/sync", { method: "POST", body: JSON.stringify({ account_id: accountId }) })
      .then(() => loadChannels(accountId))
      .then(() => toast("已同步"))
      .catch((e: Error) => toast(e.message, "err"))
      .finally(() => setSyncing(false));
  };

  const removeAccount = async (account: Account) => {
    if (!(await confirm(`删除账号「${account.name}」？`, "其下会话配置会一并删除。", "删除"))) {
      return;
    }
    api(`/api/accounts/${account.id}`, { method: "DELETE" })
      .then(() => loadAccounts())
      .catch((e: Error) => toast(e.message, "err"));
  };

  return (
    <Desk
      title="Telegram"
      actions={
        <>
          {current && accounts.length > 1 && (
            <Menu
              label={
                <>
                  <i className={`dot ${current.status === "online" ? "ok" : ""}`} />
                  <span className="menu-label">{current.name}</span>
                </>
              }
            >
              {(close) =>
                accounts.map((a) => (
                  <MenuItem
                    key={a.id}
                    onClick={() => {
                      setAccountId(a.id);
                      close();
                    }}
                  >
                    <i className={`dot ${a.status === "online" ? "ok" : ""}`} />
                    <span>{a.name}{a.id === accountId ? " · 当前" : ""}</span>
                  </MenuItem>
                ))
              }
            </Menu>
          )}
          {current && accounts.length <= 1 && (
            <span className="chip">
              <i className={`dot ${current.status === "online" ? "ok" : ""}`} />
              {current.name}
            </span>
          )}
          {current && (
            <Btn ghost icon={<Icon name="login" />} onClick={() => setLoginFor(current)}>
              {current.status === "online" ? "重登" : "登录"}
            </Btn>
          )}
          <Btn ghost icon={<Icon name="plus" />} onClick={() => setAddOpen(true)}>
            账号
          </Btn>
          {current && (
            <IconBtn label="删除账号" danger onClick={() => void removeAccount(current)}>
              <Icon name="trash" />
            </IconBtn>
          )}
          <Btn icon={<Icon name="sync" />} disabled={!current || syncing} onClick={sync}>
            {syncing ? "同步中" : "同步"}
          </Btn>
        </>
      }
    >
      <section className="canvas solo">
        {!current ? (
          <Empty
            title="还没有 Telegram 账号"
            action={
              <Btn icon={<Icon name="plus" />} onClick={() => setAddOpen(true)}>
                添加账号
              </Btn>
            }
          />
        ) : (
          <>
            <div className="canvas-tools">
              <label className="search">
                <Icon name="search" />
                <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="搜索会话" />
              </label>
              <Seg
                value={filter}
                onChange={setFilter}
                items={[
                  { id: "all", label: `全部 ${channels.length}` },
                  { id: "on", label: `监听 ${live}` },
                  { id: "off", label: `静默 ${channels.length - live}` },
                ]}
              />
              {current.status !== "online" && <span className="meta warn">未登录，登录后才能同步</span>}
            </div>
            <div className="canvas-body">
              {view.length === 0 ? (
                <Empty
                  title={channels.length === 0 ? "同步后会话会出现在这里" : "没有匹配"}
                  action={
                    channels.length === 0 ? (
                      <Btn icon={<Icon name="sync" />} disabled={syncing || current.status !== "online"} onClick={sync}>
                        同步
                      </Btn>
                    ) : undefined
                  }
                />
              ) : (
                <ul className="list">
                  {view.map((c) => (
                    <li className="list-row" key={c.id}>
                      <Switch
                        checked={!c.disabled}
                        label={c.disabled ? "开始监听" : "停止监听"}
                        onChange={(on) => {
                          api(`/api/channels/${c.id}`, { method: "PATCH", body: JSON.stringify({ disabled: !on }) })
                            .then(() => loadChannels(accountId))
                            .catch((e: Error) => toast(e.message, "err"));
                        }}
                      />
                      <div className="list-main">
                        <strong>{cleanSourceName(c.title) || c.username || "未命名"}</strong>
                        <div className="list-meta">
                          {c.username && <cite>@{c.username}</cite>}
                          <Tags items={c.themes} />
                        </div>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </>
        )}
      </section>

      {addOpen && (
        <Veil title="添加账号" onClose={() => setAddOpen(false)}>
          <form
            className="stack"
            onSubmit={(e) => {
              e.preventDefault();
              const fd = new FormData(e.currentTarget);
              api("/api/accounts", {
                method: "POST",
                body: JSON.stringify({
                  name: fd.get("name"),
                  api_id: Number(fd.get("api_id")),
                  api_hash: fd.get("api_hash"),
                  phone: fd.get("phone"),
                }),
              })
                .then(() => {
                  setAddOpen(false);
                  return loadAccounts();
                })
                .catch((err: Error) => toast(err.message, "err"));
            }}
          >
            <Field label="名称">
              <input name="name" required autoFocus placeholder="便于辨认，如工作号" />
            </Field>
            <Field label="api_id">
              <input name="api_id" required inputMode="numeric" />
            </Field>
            <Field label="api_hash">
              <input name="api_hash" required />
            </Field>
            <Field label="手机号">
              <input name="phone" placeholder="可选" />
            </Field>
            <div className="sheet-actions">
              <Btn type="submit" icon={<Icon name="plus" />}>
                添加
              </Btn>
            </div>
          </form>
        </Veil>
      )}

      {loginFor && <LoginSheet account={loginFor} onClose={() => setLoginFor(null)} onOnline={loadAccounts} />}
    </Desk>
  );
}

function LoginSheet({ account, onClose, onOnline }: { account: Account; onClose: () => void; onOnline: () => void }) {
  const toast = useToast();
  const [password, setPassword] = useState("");
  const [login, setLogin] = useState<LoginState | null>(null);
  const active = login?.status === "connecting" || login?.status === "waiting" || login?.status === "migrating";

  useEffect(() => {
    if (!active) {
      return;
    }
    let stop = false;
    const tick = window.setInterval(() => {
      api<LoginState>(`/api/accounts/${account.id}/login`)
        .then((st) => {
          if (stop) {
            return;
          }
          setLogin(st);
          if (st.status === "online") {
            toast("已登录");
            onOnline();
            onClose();
          }
        })
        .catch((e: Error) => {
          if (!stop) {
            toast(e.message, "err");
          }
        });
    }, 1500);
    return () => {
      stop = true;
      window.clearInterval(tick);
    };
  }, [active, account.id, onClose, onOnline, toast]);

  const statusHint =
    login?.status === "connecting"
      ? "连接中…"
      : login?.status === "migrating"
        ? "写入会话…"
        : login?.status === "waiting" && !login.url
          ? "申请二维码…"
          : null;

  return (
    <Veil title={`登录 · ${account.name}`} onClose={onClose} wide>
      <p className="lead">手机 Telegram → 设置 → 设备 → 链接桌面设备</p>
      <Field label="两步验证密码">
        <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="未开启则留空" />
      </Field>
      <div className="sheet-actions">
        <Btn
          icon={<Icon name="login" />}
          onClick={() => {
            api(`/api/accounts/${account.id}/login`, { method: "POST", body: JSON.stringify({ password }) })
              .then((st) => setLogin(st as LoginState))
              .catch((e: Error) => toast(e.message, "err"));
          }}
        >
          生成二维码
        </Btn>
      </div>
      {statusHint && <p className="quiet">{statusHint}</p>}
      {login?.url && login.status === "waiting" && <LoginQR url={login.url} />}
      {login?.status === "error" && <p className="fail">{login.error || "登录失败"}</p>}
    </Veil>
  );
}

function LoginQR({ url }: { url: string }) {
  const [src, setSrc] = useState("");
  useEffect(() => {
    let stop = false;
    void QRCode.toDataURL(url, { width: 280, margin: 1, errorCorrectionLevel: "M" }).then((data) => {
      if (!stop) {
        setSrc(data);
      }
    });
    return () => {
      stop = true;
    };
  }, [url]);
  if (!src) {
    return <p className="quiet">绘制中…</p>;
  }
  return <img className="qr" width={280} height={280} alt="登录二维码" src={src} />;
}
