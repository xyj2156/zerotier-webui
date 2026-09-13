<template lang="pug">
  el-card(class="h-[calc(100%-2px)]")
    template(#header) 账户
    el-descriptions(:column="1" border v-loading="pending")
      el-descriptions-item(label="名称") {{ user.name || '-' }}
      el-descriptions-item(label="邮箱") {{ user.email || '-' }}
    .mt-20px
      el-button(type="danger" @click="logout") 退出登录
</template>

<script setup>
  import { useForgeApi } from '@route-forge/vue';
  import { ElMessage } from 'element-plus';
  import { getUser, setUser, clearAuth } from '@/utils/auth';

  const router = useRouter();
  const user = ref(getUser() || {});
  const { pending, call } = useForgeApi('admin');

  onMounted(async () => {
    const { data, error } = await call('me');
    if (!error && data) {
      user.value = data;
      setUser(data);
    }
  });

  async function logout() {
    await call('logout');
    clearAuth();
    ElMessage.success('已退出');
    router.replace({ name: 'login' });
  }
</script>
