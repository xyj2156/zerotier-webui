import { createRouter, createWebHistory } from 'vue-router';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: () => import('@/layouts/index.vue'),
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'index',
          component: () => import('@/views/index.vue'),
          meta: {
            title: '首页',
          },
        },
      ],
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
