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
      el-descriptions-item(label="在线状态")
        el-tag(v-if="online" type="success") 在线
        el-tag(v-else type="danger") 离线
      el-descriptions-item(label="Peers") {{ peersText }}
      el-descriptions-item(label="公网端点")
        span.font-mono {{ surfaceText }}
      el-descriptions-item(label="监听端口") {{ primaryPortText }}
      el-descriptions-item(label="TCP 中继") {{ relayText }}
      el-descriptions-item(label="托管网络数")
        el-link(underline="never" @click="router.push({ name: 'networks' })") {{ networkCount }}
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
  const peerTotal = ref(null);
  const peerOnline = ref(null);

  const peersText = computed(() =>
    peerTotal.value === null ? '-' : `${peerOnline.value} / ${peerTotal.value}`,
  );

  const nodeSettings = computed(() => status.value.config?.settings || {});
  const surfaceText = computed(() => {
    const list = nodeSettings.value.surfaceAddresses;
    return Array.isArray(list) && list.length ? list.join(' , ') : '-';
  });
  const primaryPortText = computed(() => nodeSettings.value.primaryPort ?? '-');
  const relayText = computed(() =>
    status.value.tcpFallbackActive ? '已启用' : '未启用',
  );

  onMounted(load);

  async function load() {
    const [s, c, p] = await Promise.all([
      call('status'),
      call('network-count'),
      call('peers'),
    ]);
    status.value = s.data || {};
    // 在线优先信任 /status.online 布尔字段；老版本没有该字段时，请求成功即视为在线。
    online.value = !s.error && (typeof status.value.online === 'boolean' ? status.value.online : true);
    if (!c.error) networkCount.value = c.data ?? 0;

    if (p.error || !Array.isArray(p.data)) {
      peerTotal.value = null;
      peerOnline.value = null;
    } else {
      peerTotal.value = p.data.length;
      peerOnline.value = p.data.filter(
        (peer) => (peer.paths || []).some((path) => path && path.active && !path.expired),
      ).length;
    }
  }

  function clear() {
    clearConnection();
    ElMessage.info('已清除连接信息');
    router.replace({ name: 'connect' });
  }
</script>
