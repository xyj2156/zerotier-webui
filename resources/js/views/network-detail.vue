<template lang="pug">
  el-card
    template(#header)
      span 网络详情：{{ nwid }}
      el-tag(type="success" style="margin-left:10px") {{ network.name }}

    el-tabs(v-model="activeTab" type="card")
      el-tab-pane(label="成员管理" name="members")
        .toolbar(style="margin-bottom:15px")
          el-button(type="primary" size="small" @click="load")
            el-icon
              Refresh
            | 刷新
          el-button(type="warning" size="small" @click="copyNwid")
            el-icon
              CopyDocument
            | 复制网络ID

        // 成员表格
        el-table(:data="members" border stripe)
          el-table-column(label="节点ID" prop="id" width="180")
          el-table-column(label="名称")
            template(#default="scope")
              el-input(
                v-model="scope.row.name"
                size="small"
                @blur="save(scope.row)"
              )
          el-table-column(label="IP地址" width="150")
            template(#default="scope") {{ scope.row.ipAssignments?.[0] || '-' }}
  el-table-column(label="授权" width="100")
    template(#default="scope")
      el-switch(
        v-model="scope.row.authorized"
        @change="save(scope.row)"
      )
  el-table-column(label="活跃" width="100")
    template(#default="scope")
      el-tag(
        :type="scope.row.isOnline ? 'success' : 'info'"
      ) {{ scope.row.isOnline ? '在线' : '离线' }}
  el-table-column(label="操作" width="120")
    template(#default="scope")
      el-button(
        type="danger"
        size="small"
        icon="Delete"
        @click="del(scope.row.id)"
      ) 删除

  el-tab-pane(label="网络配置" name="config")
    el-descriptions(:column="1" border)
      el-descriptions-item(label="网络ID") {{ nwid }}
      el-descriptions-item(label="名称") {{ network.name }}
      el-descriptions-item(label="子网") { network.routes?.[0]?.target || '10.144.0.0/16' }}
</template>

<script setup>
  import { ref, onMounted } from 'vue';
  import axios from '@/utils/fetch.js';
  import { ElMessage } from 'element-plus';
  import { useRoute } from 'vue-router';

  const route = useRoute();
  const nwid = route.params.nwid;
  const activeTab = ref('members');

  const network = ref({});
  const members = ref([]);

  // 加载数据
  const load = async () => {
    const [net, mem] = await Promise.all([
      axios.get(`/api/networks/${nwid}`),
      axios.get(`/api/networks/${nwid}/members`),
    ]);
    network.value = net.data;
    members.value = Object.values(mem.data).map((m) => ({
      ...m,
      isOnline:
        m.lastSeen && Date.now() - new Date(m.lastSeen).getTime() < 300000,
    }));
  };

  // 保存成员
  const save = async (row) => {
    await axios.post(`/api/networks/${nwid}/members/${row.id}`, {
      name: row.name,
      authorized: row.authorized,
    });
    ElMessage.success('保存成功');
  };

  // 删除成员
  const del = async (id) => {
    if (!confirm('确定删除？')) return;
    await axios.delete(`/api/networks/${nwid}/members/${id}`);
    ElMessage.success('删除成功');
    load();
  };

  // 复制网络ID
  const copyNwid = () => {
    navigator.clipboard.writeText(nwid);
    ElMessage.success('已复制');
  };

  onMounted(load);
</script>
