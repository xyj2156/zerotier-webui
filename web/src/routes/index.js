import { createRouter, createWebHistory } from 'vue-router';
import { hasConnection } from '@/utils/auth';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/connect',
      name: 'connect',
      component: () => import('@/views/connect.vue'),
      meta: { title: '连接控制器 - ZeroTier 管理器', public: true },
    },
    {
      path: '/',
      name: 'index',
      component: () => import('@/views/dashboard.vue'),
      meta: { title: '仪表盘 - ZeroTier 管理器' },
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

// 没有控制器令牌就没有任何可展示的数据，因此未连接时统一落到「连接」页。
router.beforeEach((to) => {
  if (!to.meta.public && !hasConnection()) {
    return { name: 'connect' };
  }
  return true;
});

router.afterEach((to) => {
  document.title = to.meta?.title || 'ZeroTier 管理器';
});

export default router;
