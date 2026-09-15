import 'virtual:uno.css';
import './styles/main.css';
import { createApp } from 'vue';
import { createPinia } from 'pinia';
import { createRouteForgePlugin } from '@route-forge/vue';
import router from './routes';
import main from './main.vue';
import { getBase, getToken } from './utils/auth';

// route-forge：以命名路由方式调用后端 API，路由表由桥的 /_forge/routes 下发
const forge = createRouteForgePlugin({
  adapter: 'builtin', // 内置 fetch 适配器，零 axios 依赖
  endpoint: '/_forge/routes',
});

// 请求：注入浏览器保存的控制器令牌与可选基址覆盖。桥自身不存凭据，只做透传。
forge.interceptors.request.use(function (config) {
  config.headers = config.headers || {};
  const token = getToken();
  if (token) {
    config.headers['X-ZT-Token'] = token;
  }
  const base = getBase();
  if (base) {
    config.headers['X-ZT-Base'] = base;
  }
  return config;
});

// 响应：解包后端统一信封 { status, message, result } → result；非 0 抛业务错误
forge.interceptors.response.use(
  function (resp) {
    const body = resp.data;
    if (body && typeof body === 'object' && 'status' in body) {
      if (body.status === 0) {
        return body.result;
      }
      const message = body.message || '请求失败';
      // 缺令牌属于连接问题，统一回「连接」页处理，不在各业务页重复判断
      if (
        message.includes('未配置控制器令牌') &&
        router.currentRoute.value.name !== 'connect'
      ) {
        router.replace({ name: 'connect' });
      }
      throw new Error(message);
    }
    return body;
  },
  function (err) {
    throw err;
  }
);

const app = createApp(main);

app.use(router);
app.use(createPinia());
app.use(forge);
forge
  .ready()
  .then(function () {
    app.mount('#jason');
  })
  .catch(function () {
    document.getElementById('jason').textContent =
      '无法读取路由元信息（/_forge/routes），请确认 zerotier-webui 已启动。';
  });
