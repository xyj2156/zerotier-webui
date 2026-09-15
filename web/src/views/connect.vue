<template lang="pug">
  .mx-auto(class="max-w-[720px] pt-24px")
    el-card
      template(#header) 连接控制器
      el-form(size="default" label-width="96px" @submit.prevent="save")
        el-form-item(label="控制器地址")
          el-input(v-model="form.base" :placeholder="info?.controller || 'http://127.0.0.1:9993'" clearable)
          .text-12px.mt-4px(class="text-[var(--el-text-color-secondary)]")
            | 留空则沿用桥启动时的地址；控制器在别的机器时填 http://内网IP:9993
        el-form-item(label="访问令牌")
          el-input(
            v-model="form.token"
            type="password"
            show-password
            placeholder="authtoken.secret 文件内容"
            @keyup.enter="save"
          )
          .text-12px.mt-4px(class="text-[var(--el-text-color-secondary)]")
            | Windows：%LOCALAPPDATA%\\ZeroTier\\authtoken.secret（ProgramData 下那份需管理员权限）；
            | Linux：/var/lib/zerotier-one/authtoken.secret；macOS：/var/db/zerotier-one/authtoken.secret
        el-form-item
          el-button(type="primary" :loading="pending" @click="save") 保存并自检
          el-button(@click="clear") 清除
    el-card.mt-16px(v-if="status || info")
      template(#header) 自检结果
      el-descriptions(:column="1" border)
        el-descriptions-item(label="控制器节点") {{ status?.address || '-' }}
        el-descriptions-item(label="控制器版本") {{ status?.version || '-' }}
        el-descriptions-item(label="配置目录") {{ status?.configPath || '-' }}
        el-descriptions-item(label="桥版本") {{ info?.version || '-' }}
        el-descriptions-item(label="桥监听") {{ info?.listen || '-' }}
        el-descriptions-item(label="运行环境") {{ info?.runtime || '-' }}
    p.text-12px.mt-16px(class="text-[var(--el-text-color-secondary)]")
      | 令牌只保存在本标签页的 sessionStorage，由前端随请求下发给桥；桥不存储凭据，也不会把它发往配置之外的地址。
</template>

<script setup>
  import { useForgeApi } from '@route-forge/vue';
  import { ElMessage } from 'element-plus';
  import {
    clearConnection,
    getBase,
    getToken,
    hasConnection,
    setBase,
    setToken,
  } from '@/utils/auth';

  const router = useRouter();
  const form = reactive({ base: getBase(), token: getToken() });
  const status = ref(null);
  const info = ref(null);

  // 两个层级分开取：info 免令牌，status 需要刚保存的令牌
  const { pending, call } = useForgeApi('admin');
  const { call: callPublic } = useForgeApi('public');

  async function refresh() {
    const { data: infoData } = await callPublic('info');
    info.value = infoData ?? null;
    if (!hasConnection()) {
      status.value = null;
      return;
    }
    const { data, error } = await call('status');
    if (error) {
      status.value = null;
      ElMessage.error(error.message || '控制器不可达');
      return;
    }
    status.value = data;
    ElMessage.success('已连接控制器');
  }

  async function save() {
    const token = form.token.trim();
    if (!token) {
      ElMessage.warning('请填入 authtoken.secret 内容');
      return;
    }
    setToken(token);
    setBase(form.base.trim());
    await refresh();
  }

  function clear() {
    clearConnection();
    form.token = '';
    form.base = '';
    status.value = null;
    info.value = null;
    ElMessage.info('已清除本机保存的连接信息');
  }

  onMounted(async () => {
    const { data } = await callPublic('info');
    info.value = data ?? null;
    if (hasConnection()) {
      await refresh();
      if (status.value) {
        router.replace({ name: 'index' });
      }
    }
  });
</script>
