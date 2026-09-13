// 令牌与当前用户的轻量存储（真实部署下的 SPA，使用 localStorage 持久化）
const TOKEN_KEY = 'zt_token';
const USER_KEY = 'zt_user';

export function getToken() {
  try {
    return localStorage.getItem(TOKEN_KEY) || '';
  } catch {
    return '';
  }
}

export function setToken(token) {
  try {
    token ? localStorage.setItem(TOKEN_KEY, token) : localStorage.removeItem(TOKEN_KEY);
  } catch {}
}

export function getUser() {
  try {
    const raw = localStorage.getItem(USER_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

export function setUser(user) {
  try {
    user ? localStorage.setItem(USER_KEY, JSON.stringify(user)) : localStorage.removeItem(USER_KEY);
  } catch {}
}

export function clearAuth() {
  setToken('');
  setUser(null);
}
