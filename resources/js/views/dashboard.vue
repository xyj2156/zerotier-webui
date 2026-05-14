<template lang="pug">
  el-card(class="h-[calc(100%-2px)]")
    template(#header)
      span 系统状态
    el-descriptions(:column="2" border)
      el-descriptions-item(label="零层地址") {{ status?.address || '-' }}
      el-descriptions-item(label="版本") {{ status?.version || '-' }}
      el-descriptions-item(label="在线状态")
        el-tag(v-if="status.online" type="success") 在线
        el-tag(v-else type="danger") 离线
      el-descriptions-item(label="网络数量") {{ networks?.length }}
</template>

<script setup>
  import { onMounted, ref } from 'vue';
  import axios from '@/utils/fetch.js';

  const status = ref({});
  const networks = ref([]);

  onMounted(() => {
    axios.get('/api/status').then(function (res) {
      status.value = res;
    });
    axios.get('/api/networks').then(function (res) {
      networks.value = res;
    });
  });
</script>
