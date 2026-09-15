// 控制器连接信息存储：令牌与基址只存在浏览器 sessionStorage。
//
// 用 sessionStorage 而不是 localStorage 的两点考虑：
//  1. 关闭标签页即自动失效，不在浏览器里长期驻留凭据；
//  2. 语义上「一个标签页 = 一次会话」，配合页面上的清除按钮即可立刻回到未连接态。
// 代价：新开标签页或重启浏览器需要重新粘贴令牌（跨标签不共享）。
//
// 单执行文件本身不落盘任何凭据——它只是本机桥，每次请求把请求头里的令牌透传给
// zerotier-one 本地控制器；能连通即等于令牌正确，因此不需要服务端会话或账号体系。
const TOKEN_KEY = 'zt_token';
const BASE_KEY = 'zt_base';

function store() {
  return window.sessionStorage;
}

export function getToken() {
  try {
    return store().getItem(TOKEN_KEY) || '';
  } catch {
    return '';
  }
}

export function setToken(token) {
  try {
    token ? store().setItem(TOKEN_KEY, token) : store().removeItem(TOKEN_KEY);
  } catch {}
}

// getBase 返回控制器基址覆盖值；空串表示沿用桥启动时的 --url。
export function getBase() {
  try {
    return store().getItem(BASE_KEY) || '';
  } catch {
    return '';
  }
}

export function setBase(base) {
  try {
    base ? store().setItem(BASE_KEY, base) : store().removeItem(BASE_KEY);
  } catch {}
}

export function clearConnection() {
  setToken('');
  setBase('');
}

// hasConnection 是路由守卫用的最小判据：没有令牌就没有可连的控制器。
export function hasConnection() {
  return getToken() !== '';
}
