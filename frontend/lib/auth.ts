const ACCESS_KEY = "at_token";
const REFRESH_KEY = "at_refresh";

export function getToken(): string | undefined {
  if (typeof window === "undefined") return undefined;
  return localStorage.getItem(ACCESS_KEY) || undefined;
}

export function setAuth(accessToken: string, refreshToken: string) {
  if (typeof window === "undefined") return;
  localStorage.setItem(ACCESS_KEY, accessToken);
  localStorage.setItem(REFRESH_KEY, refreshToken);
}

export function clearAuth() {
  if (typeof window === "undefined") return;
  localStorage.removeItem(ACCESS_KEY);
  localStorage.removeItem(REFRESH_KEY);
}
