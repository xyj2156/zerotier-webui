<template lang="pug">
  .min-h-full.flex.items-center.justify-center(class="bg-[var(--el-fill-color-light)]")
    el-card.w-360px
      template(#header) ZeroTier 登录
      el-form(size="default" @submit.prevent="submit")
        el-form-item
          el-input(v-model="form.email" placeholder="邮箱" :prefix-icon="Message")
        el-form-item
          el-input(
            v-model="form.password"
            type="password"
            show-password
            placeholder="密码"
            :prefix-icon="Lock"
            @keyup.enter="submit"
          )
        el-button(type="primary" class="w-full" :loading="pending" @click="submit") 登录
</template>

<script setup>
  import { useForgeApi } from '@route-forge/vue';
  import { ElMessage } from 'element-plus';
  import { Message, Lock } from '@element-plus/icons-vue';
  import { setToken, setUser } from '@/utils/auth';

  const router = useRouter();
  const form = reactive({ email: '', password: '' });
  const { pending, call } = useForgeApi('public');

  async function submit() {
    if (!form.email || !form.password) {
      ElMessage.warning('请输入邮箱和密码');
      return;
    }
    const { data, error } = await call('login', {
      body: { email: form.email, password: form.password },
    });
    if (error || !data?.token) {
      ElMessage.error(error?.message || '登录失败');
      return;
    }
    setToken(data.token);
    setUser(data.user);
    ElMessage.success('登录成功');
    router.replace({ name: 'index' });
  }
</script>
