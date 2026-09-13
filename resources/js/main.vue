<template lang="pug">
  el-config-provider(:locale="zhCN")
    el-container.h-full(direction="vertical")
      template(v-if="chrome")
        el-header.border.border-solid.border-b(class="border-b-[var(--el-menu-border-color)]")
          .flex.align-center.items-center.gap-5
            .logo
              | ZeroTier 管理器
            el-menu(
              mode="horizontal"
              :ellipsis="false"
              :default-active="menu_active"
              @select="handleMenuClick"
            )
              el-menu-item(v-for="item in menus" :key="item.route" :index="item.route") {{ item.name }}
            .ml-auto.flex.items-center.gap-3
              el-tag(v-if="online" type="success") 在线 · {{ ztVersion }}
              el-tag(v-else type="danger") 离线
        el-main(class="h-[calc(100%-60px)]")
          router-view
      template(v-else)
        router-view
</template>

<script setup>
  import zhCN from 'element-plus/es/locale/lang/zh-cn';
  import { useForgeApi } from '@route-forge/vue';
  import { getToken } from '@/utils/auth';

  const route = useRoute();
  const router = useRouter();

  const ztVersion = ref('');
  const online = ref(false);
  const { call } = useForgeApi('admin');

  const chrome = computed(() => route.name !== 'login');

  const menus = [
    { name: '主页', route: 'index', path: '/' },
    { name: '网络', route: 'networks', path: '/networks' },
    { name: '账户', route: 'user', path: '/user' },
  ];

  const menu_active = computed(() => {
    const hit = menus.find((m) => m.route === route.name);
    return hit ? hit.route : route.name;
  });

  async function loadStatus() {
    if (!getToken()) return;
    const { data, error } = await call('status');
    if (error || !data) {
      online.value = false;
      return;
    }
    online.value = true;
    ztVersion.value = data.version || data.controller?.version || '';
  }

  function handleMenuClick(name) {
    router.push({ name });
  }

  watch(chrome, (v) => v && loadStatus(), { immediate: true });
  watch(() => route.name, () => getToken() && loadStatus());
</script>

<style scoped></style>
