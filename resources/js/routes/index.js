import { createRouter, createWebHistory } from 'vue-router';

const router = createRouter({
  history: createWebHistory(),
  routes: [
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
      meta: { title: '用户管理 - ZeroTier 管理器' },
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

router.beforeEach((to, from) => {
  document.title = 'Loading ...';
  console.log('before', from);
});

router.afterEach((to, from) => {
  document.title = to.meta?.title;
  console.log('after', to);
});

export default router;
