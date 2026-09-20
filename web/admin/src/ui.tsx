import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useId,
  useRef,
  useState,
  type ButtonHTMLAttributes,
  type ReactNode,
} from "react";
import type { ThemeTag } from "./types";

type Toast = { id: number; text: string; kind: "ok" | "err" };
type ConfirmState = {
  title: string;
  body?: string;
  ok: string;
  resolve: (yes: boolean) => void;
};

const ToastFn = createContext<(text: string, kind?: "ok" | "err") => void>(() => undefined);
const ConfirmFn = createContext<(title: string, body?: string, ok?: string) => Promise<boolean>>(async () => false);

export function useToast() {
  return useContext(ToastFn);
}

export function useConfirm() {
  return useContext(ConfirmFn);
}

export function Feedback({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [confirm, setConfirm] = useState<ConfirmState | null>(null);
  const push = useCallback((text: string, kind: "ok" | "err" = "ok") => {
    const id = Date.now() + Math.random();
    setToasts((cur) => [...cur.slice(-3), { id, text, kind }]);
    window.setTimeout(() => setToasts((cur) => cur.filter((t) => t.id !== id)), 2400);
  }, []);
  const ask = useCallback((title: string, body?: string, ok = "确定") => {
    return new Promise<boolean>((resolve) => setConfirm({ title, body, ok, resolve }));
  }, []);
  return (
    <ToastFn.Provider value={push}>
      <ConfirmFn.Provider value={ask}>
        {children}
        <div className="toasts" aria-live="polite">
          {toasts.map((t) => (
            <div className={`toast ${t.kind}`} key={t.id}>
              {t.text}
            </div>
          ))}
        </div>
        {confirm && (
          <div
            className="veil"
            role="presentation"
            onClick={() => {
              confirm.resolve(false);
              setConfirm(null);
            }}
          >
            <div className="sheet" role="dialog" onClick={(e) => e.stopPropagation()}>
              <h3>{confirm.title}</h3>
              {confirm.body && <p>{confirm.body}</p>}
              <div className="sheet-actions">
                <Btn ghost onClick={() => { confirm.resolve(false); setConfirm(null); }}>
                  取消
                </Btn>
                <Btn danger onClick={() => { confirm.resolve(true); setConfirm(null); }}>
                  {confirm.ok}
                </Btn>
              </div>
            </div>
          </div>
        )}
      </ConfirmFn.Provider>
    </ToastFn.Provider>
  );
}

export function Veil({
  title,
  onClose,
  children,
  wide,
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
  wide?: boolean;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);
  return (
    <div className="veil" role="presentation" onClick={onClose}>
      <div className={`sheet ${wide ? "wide" : ""}`} role="dialog" aria-label={title} onClick={(e) => e.stopPropagation()}>
        <header className="sheet-head">
          <h3>{title}</h3>
          <IconBtn label="关闭" onClick={onClose}>
            <Icon name="close" />
          </IconBtn>
        </header>
        {children}
      </div>
    </div>
  );
}

export function Empty({ title, action }: { title: string; action?: ReactNode }) {
  return (
    <div className="blank">
      <p>{title}</p>
      {action}
    </div>
  );
}

export function Desk({
  title,
  actions,
  children,
}: {
  title: string;
  actions?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="desk">
      <header className="desk-bar">
        <h1>{title}</h1>
        {actions && <div className="bar-actions">{actions}</div>}
      </header>
      {children}
    </div>
  );
}

type BtnProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  ghost?: boolean;
  danger?: boolean;
  icon?: ReactNode;
};

export function Btn({ ghost, danger, icon, children, className = "", ...rest }: BtnProps) {
  const kind = danger ? "danger" : ghost ? "ghost" : "solid";
  return (
    <button type="button" className={`btn ${kind} ${className}`.trim()} {...rest}>
      {icon}
      {children && <span>{children}</span>}
    </button>
  );
}

export function IconBtn({
  label,
  danger,
  children,
  className = "",
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & { label: string; danger?: boolean; children: ReactNode }) {
  return (
    <button
      type="button"
      className={`icon-btn ${danger ? "danger" : ""} ${className}`.trim()}
      aria-label={label}
      title={label}
      {...rest}
    >
      {children}
    </button>
  );
}

export function Seg<T extends string>({
  value,
  onChange,
  items,
}: {
  value: T;
  onChange: (v: T) => void;
  items: { id: T; label: string }[];
}) {
  return (
    <div className="seg" role="tablist">
      {items.map((it) => (
        <button
          key={it.id}
          type="button"
          role="tab"
          aria-selected={value === it.id}
          className={value === it.id ? "on" : ""}
          onClick={() => onChange(it.id)}
        >
          {it.label}
        </button>
      ))}
    </div>
  );
}

export function Switch({ checked, onChange, label }: { checked: boolean; onChange: (v: boolean) => void; label: string }) {
  return (
    <button
      type="button"
      className={`switch ${checked ? "on" : ""}`}
      role="switch"
      aria-checked={checked}
      aria-label={label}
      onClick={() => onChange(!checked)}
    >
      <span />
    </button>
  );
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="field">
      <span>{label}</span>
      {children}
    </label>
  );
}

export function Tags({ items }: { items?: ThemeTag[] }) {
  if (!items?.length) {
    return null;
  }
  return (
    <div className="tags">
      {items.map((t) => (
        <span key={t.id}>{t.name}</span>
      ))}
    </div>
  );
}

export function Menu({
  label,
  children,
}: {
  label: ReactNode;
  children: (close: () => void) => ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const menuId = useId();
  useEffect(() => {
    if (!open) {
      return;
    }
    const onDoc = (e: MouseEvent) => {
      if (!root.current?.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", onDoc);
    window.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      window.removeEventListener("keydown", onKey);
    };
  }, [open]);
  const close = useCallback(() => setOpen(false), []);
  return (
    <div className={`menu ${open ? "open" : ""}`} ref={root}>
      <button
        type="button"
        className="menu-trigger"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={menuId}
        onClick={() => setOpen((v) => !v)}
      >
        {label}
        <Icon name="chevron" />
      </button>
      {open && (
        <div className="menu-pop" role="menu" id={menuId}>
          {children(close)}
        </div>
      )}
    </div>
  );
}

export function MenuItem({
  children,
  onClick,
  danger,
}: {
  children: ReactNode;
  onClick: () => void;
  danger?: boolean;
}) {
  return (
    <button type="button" role="menuitem" className={`menu-item ${danger ? "danger" : ""}`} onClick={onClick}>
      {children}
    </button>
  );
}

export type IconName =
  | "telegram"
  | "rss"
  | "columns"
  | "review"
  | "rules"
  | "journal"
  | "plus"
  | "trash"
  | "edit"
  | "refresh"
  | "sync"
  | "login"
  | "close"
  | "check"
  | "external"
  | "search"
  | "chevron"
  | "minus";

const iconPaths: Record<IconName, string> = {
  telegram: "M4 12h16M12 4l8 8-8 8",
  rss: "M4 4v16h16M8 16a2 2 0 1 0 .01 0M8 8c4.4 0 8 3.6 8 8M8 12c2.2 0 4 1.8 4 4",
  columns: "M4 6h16M4 12h10M4 18h16",
  review: "M4 5h16v14H4zM8 9h8M8 13h5",
  rules: "M12 4v16M8 8h8M8 16h8",
  journal: "M5 5h14v14H5zM8 9h8M8 13h6",
  plus: "M12 5v14M5 12h14",
  trash: "M4 7h16M9 7V5h6v2M8 7v12h8V7",
  edit: "M4 20h4l10-10-4-4L4 16v4zM12 8l4 4",
  refresh: "M4 12a8 8 0 0 1 14-5M20 12a8 8 0 0 1-14 5M20 4v5h-5M4 20v-5h5",
  sync: "M4 12a8 8 0 0 1 13-6M20 12a8 8 0 0 1-13 6M17 4h3v3M7 20H4v-3",
  login: "M10 4H6v16h4M14 8l4 4-4 4M10 12h8",
  close: "M6 6l12 12M18 6L6 18",
  check: "M5 12l5 5L19 7",
  external: "M10 4h10v10M20 4L10 14M4 10v10h10",
  search: "M11 5a6 6 0 1 0 0 12 6 6 0 0 0 0-12zM20 20l-4-4",
  chevron: "M8 10l4 4 4-4",
  minus: "M6 12h12",
};

export function Icon({ name, size = 16 }: { name: IconName; size?: number }) {
  return (
    <svg className="ico" viewBox="0 0 24 24" width={size} height={size} aria-hidden>
      <path d={iconPaths[name]} fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

/** @deprecated use Icon */
export function Glyph({ name }: { name: IconName }) {
  return <Icon name={name} size={18} />;
}

export function cleanSourceName(name: string) {
  return name.replace(/\s+Messages$/i, "").trim() || name;
}
