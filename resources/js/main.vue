<template lang="pug">
  el-config-provider(:locale="zhCN")
    el-container
      el-header.border.border-solid.border-b(class="border-b-[var(--el-menu-border-color)]")
        .flex.align-center.items-center.gap-5
          .logo
            | 阿杰很厉害
          el-menu(mode="horizontal" index="1" :ellipsis="false" :default-active="menu_active")
            el-menu-item(v-for="(item, i) in menus" :index="item.route" @click="handleMenuClick(item)") {{ item.name }}
          el-tag(type="success" v-if="ztVersion") {{ ztVersion }}
          template 离线
      el-main
        router-view
</template>

<script setup>
  import fetch from '@/utils/fetch.js';
  import zhCN from 'element-plus/es/locale/lang/zh-cn';

  const route = useRoute();
  const router = useRouter();

  const ztVersion = ref('');
  const menu_active = ref('index');
  const menus = [
    { name: '主页', route: 'index', path: '/' },
    { name: '用户', route: 'user', path: '/user' },
    { name: '网络', route: 'networks', path: '/network' },
  ];

  onMounted(async () => {
    try {
      const res = await fetch.get('/api/status');
      if (res.version) ztVersion.value = res.version;
    } catch (e) {}
  });

  const act = { eq: '', like: '' };

  watch(
    () => route.path,
    (val) => {
      console.log(val);
      menus.forEach(function (item) {
        if (item.path === val) {
          act.eq = item.route;
        }
        if (val.startsWith(item.path)) {
          act.like = item.route;
        }
      });
      if (act.eq) {
        menu_active.value = act.eq;
      } else if (act.like) {
        menu_active.value = act.like;
      }
    },
    {
      immediate: true,
    }
  );

  function handleMenuClick(menu) {
    console.log(menu);
    if (menu.route) {
      router.push({ name: menu.route });
      return;
    }
    // 处理添加网络弹窗
  }
</script>

<style scoped></style>
