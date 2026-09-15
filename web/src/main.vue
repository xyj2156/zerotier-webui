<template lang="pug">
  el-config-provider(:locale="zhCN")
    el-container.h-full(direction="vertical")
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
            el-tag(v-else-if="checked" type="danger") 离线
      el-main(class="h-[calc(100%-60px)]")
        router-view
</template>

<script setup>
  import zhCN from 'element-plus/es/locale/lang/zh-cn';
  import { useForgeApi } from '@route-forge/vue';
  import { hasConnection } from '@/utils/auth';

  const route = useRoute();
  const router = useRouter();

  const ztVersion = ref('');
  const online = ref(false);
  const checked = ref(false);
  const { call } = useForgeApi('admin');

  const menus = [
    { name: '主页', route: 'index', path: '/' },
    { name: '网络', route: 'networks', path: '/networks' },
    { name: '连接', route: 'connect', path: '/connect' },
  ];

  const menu_active = computed(() => {
    const hit = menus.find((m) => m.route === route.name);
    return hit ? hit.route : route.name;
  });

  // 控制器状态是全局信息（顶栏角标），未填令牌时不做无谓请求。
  async function loadStatus() {
    if (!hasConnection()) {
      online.value = false;
      checked.value = false;
      return;
    }
    const { data, error } = await call('status');
    online.value = !error && !!data;
    ztVersion.value = data?.version || data?.controller?.version || '';
    checked.value = true;
  }

  function handleMenuClick(name) {
    router.push({ name });
  }

  watch(
    () => route.name,
    () => loadStatus(),
    { immediate: true }
  );
</script>

<style scoped></style>
