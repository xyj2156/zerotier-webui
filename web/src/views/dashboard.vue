<template lang="pug">
  el-card(class="h-[calc(100%-2px)]")
    template(#header)
      .flex.items-center
        span 系统状态
        el-popover(content="刷新" placement="top")
          template(#reference)
            el-icon.ml-5px.cursor-pointer.text-primary(@click="load")
              Refresh
        div(class="ml-auto")
          el-popconfirm(
            title="清除本机保存的控制器令牌？"
            confirm-button-text="清除"
            cancel-button-text="取消"
            @confirm="clear"
          )
            template(#reference)
              el-button(size="small" type="danger" plain) 清除连接
    el-descriptions(:column="2" border v-loading="pending")
      el-descriptions-item(label="节点地址") {{ status.address || '-' }}
      el-descriptions-item(label="版本") {{ status.version || '-' }}
      el-descriptions-item(label="平台") {{ status.platform || '-' }}
      el-descriptions-item(label="在线状态")
        el-tag(v-if="online" type="success") 在线
        el-tag(v-else type="danger") 离线
      el-descriptions-item(label="托管网络数")
        el-link(underline="never" @click="router.push({ name: 'networks' })") {{ networkCount }}
      el-descriptions-item(label="Peers") {{ status.peers ?? status.numPeers ?? '-' }}
</template>

<script setup>
  import { useForgeApi } from '@route-forge/vue';
  import { ElMessage } from 'element-plus';
  import { Refresh } from '@element-plus/icons-vue';
  import { clearConnection } from '@/utils/auth';

  const router = useRouter();
  const { pending, call } = useForgeApi('admin');

  const status = ref({});
  const networkCount = ref(0);
  const online = ref(false);

  onMounted(load);

  async function load() {
    const [s, c] = await Promise.all([call('status'), call('network-count')]);
    online.value = !s.error && !!s.data;
    status.value = s.data || {};
    if (!c.error) networkCount.value = c.data ?? 0;
  }

  function clear() {
    clearConnection();
    ElMessage.info('已清除连接信息');
    router.replace({ name: 'connect' });
  }
</script>
