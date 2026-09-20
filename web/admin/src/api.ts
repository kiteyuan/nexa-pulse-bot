const tokenKey = "nexa_admin_token";

export function getToken(): string {
  return localStorage.getItem(tokenKey) ?? "";
}

export function setToken(token: string): void {
  if (!token) {
    localStorage.removeItem(tokenKey);
    return;
  }
  localStorage.setItem(tokenKey, token);
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (!headers.has("Content-Type") && init.body) {
    headers.set("Content-Type", "application/json");
  }
  headers.set("Authorization", `Bearer ${getToken()}`);
  const res = await fetch(path, { ...init, headers });
  const data = (await res.json()) as T & { error?: string };
  if (!res.ok) {
    throw new Error(data.error || res.statusText);
  }
  return data;
}
