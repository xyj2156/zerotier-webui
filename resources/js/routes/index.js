import { createRouter, createWebHistory } from 'vue-router';
import { getToken } from '@/utils/auth';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/login.vue'),
      meta: { title: '登录 - ZeroTier 管理器', public: true },
    },
    {
      path: '/',
      name: 'index',
      component: () => import('@/views/dashboard.vue'),
      meta: { title: '仪表盘 - ZeroTier 管理器' },
    },
    {
      path: '/user',
      name: 'user',
      component: () => import('@/views/user.vue'),
      meta: { title: '账户 - ZeroTier 管理器' },
    },
    {
      path: '/networks',
      name: 'networks',
      component: () => import('@/views/network.vue'),
      meta: { title: '网络管理 - ZeroTier 管理器' },
    },
    {
      path: '/network/:nwid',
      name: 'network-detail',
      component: () => import('@/views/network-detail.vue'),
      meta: { title: '网络详情 - ZeroTier 管理器' },
    },
  ],
});

router.beforeEach((to) => {
  if (!to.meta.public && !getToken()) {
    return { name: 'login' };
  }
  if (to.name === 'login' && getToken()) {
    return { name: 'index' };
  }
  return true;
});

router.afterEach((to) => {
  document.title = to.meta?.title || 'ZeroTier 管理器';
});

export default router;
