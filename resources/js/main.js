import 'virtual:uno.css';
import { createApp } from 'vue';
import { createPinia } from 'pinia';
import { createRouteForgePlugin } from '@route-forge/vue';
import router from './routes';
import main from './main.vue';
import { clearAuth, getToken } from './utils/auth';

// route-forge：以命名路由方式调用后端 API，替换原 axios / fetch.js 封装
const forge = createRouteForgePlugin({
  adapter: 'builtin', // 内置 fetch 适配器，零 axios 依赖
  endpoint: '/_forge/routes',
});

// 请求：注入 Bearer 令牌
forge.interceptors.request.use(function (config) {
  const token = getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
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
      throw new Error(body.message || '请求失败');
    }
    return body;
  },
  function (err) {
    const status = err?.response?.status ?? err?.status;
    if (status === 401) {
      clearAuth();
      if (router.currentRoute.value.name !== 'login') {
        router.push({ name: 'login' });
      }
    }
    throw err;
  }
);

const app = createApp(main);

app.use(router);
app.use(createPinia());
app.use(forge);
forge.ready().then(function () {
  app.mount('#jason');
}).catch(function() {
  console.error(...arguments)
});
