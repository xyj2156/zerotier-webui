<template lang="pug">
  el-card(class="h-[calc(100%-2px)]")
    template(#header)
      span 网络ID：{{ nwid }}
      el-popover(content="复制网络ID" placement="top")
        template(#reference)
          el-icon.ml-5px.cursor-pointer.text-primary(@click="copyNwid")
            CopyDocument
      el-popover(content="刷新" placement="top")
        template(#reference)
          el-icon.ml-5px.cursor-pointer.text-primary(@click="load")
            Refresh
      span.ml-10px 名称：{{ network.name || '-' }}

    el-tabs.h-full(v-model="activeTab" type="card")
      // ===================== 成员管理 =====================
      el-tab-pane.h-full(label="成员管理" name="members")
        el-table(:data="members" border stripe max-height="100%" v-loading="pendingMembers")
          el-table-column(type="expand")
            template(#default="{ row }")
              .p-15px
                el-form(label-width="120px" size="small")
                  el-form-item(label="不自动分配IP")
                    el-switch(v-model="row.noAutoAssignIps" @change="dirty(row)")
                  el-form-item(label="隐藏成员")
                    el-switch(v-model="row.hidden" @change="dirty(row)")
                  el-form-item(label="节点IP")
                    span.font-mono {{ row.physicalAddress || row.publicAddress || '-' }}
                  el-form-item(label="稳定端点")
                    span.font-mono.text-12px {{ (row.stableEndpoints || []).join(' , ') || '-' }}
                  el-form-item(label="成员流规则")
                    el-input(v-model="row.flowRules" type="textarea" :rows="3" @change="dirty(row)")
          el-table-column(label="节点ID" width="120")
            template(#default="{ row }")
              span.font-mono {{ row.nodeId || row.address }}
          el-table-column(label="名称" min-width="140")
            template(#default="{ row }")
              el-input(v-model="row.name" size="small" @change="dirty(row)")
          el-table-column(label="分配 IP" min-width="180")
            template(#default="{ row }")
              .flex.flex-wrap.gap-5px
                el-tag(
                  v-for="(ip, i) in row.ipList"
                  :key="i"
                  closable
                  size="small"
                  @close="removeIp(row, i)"
                ) {{ ip }}
                el-button(link type="primary" size="small" @click="addIp(row)") + IP
          el-table-column(label="授权" width="80")
            template(#default="{ row }")
              el-switch(v-model="row.authorized" @change="dirty(row)")
          el-table-column(label="桥接" width="80")
            template(#default="{ row }")
              el-switch(v-model="row.activeBridge" @change="dirty(row)")
          el-table-column(label="状态" width="90")
            template(#default="{ row }")
              el-tag(:type="row.online ? 'success' : 'info'") {{ row.online ? '在线' : '离线' }}
          el-table-column(label="末次活跃" width="180")
            template(#default="{ row }") {{ formatTime(row.lastOnline) }}
          el-table-column(label="版本" width="100")
            template(#default="{ row }") {{ row.clientVersion || '-' }}
          el-table-column(label="操作" width="120" fixed="right")
            template(#default="{ row }")
              el-button(v-if="row._dirty" type="success" link size="small" @click="save(row)") 保存
              el-popconfirm(title="确定移除该成员？" @confirm="delMember(row.nodeId || row.address)")
                template(#reference)
                  el-button(type="danger" link size="small" class="ml-8px") 移除

      // ===================== 网络配置 =====================
      el-tab-pane.h-full(label="网络配置" name="config")
        .config-pane(v-loading="pendingConfig")
          .config-body
            // 左栏：结构化表单，独立 el-scrollbar
            el-scrollbar.config-col
              el-form.label-col(label-width="130px" size="default")
                el-divider(content-position="left") 基本信息
                el-form-item(label="名称")
                  el-input(v-model="form.name")
                el-form-item(label="启用网络")
                  el-switch(v-model="form.active")
                el-form-item(label="私有网络")
                  el-switch(v-model="form.private")
                el-form-item(label="允许广播")
                  el-switch(v-model="form.enableBroadcast")
                el-form-item(label="MTU")
                  el-input-number(v-model="form.mtu" :min="1280" :max="2800")
                el-form-item(label="组播上限")
                  el-input-number(v-model="form.multicastLimit" :min="1" :max="255")
                el-form-item(label="组播 TTL")
                  el-input-number(v-model="form.multicastTTL" :min="1" :max="255")

                el-divider(content-position="left") 地址分配
                el-form-item(label="IPv4 (zt)")
                  el-switch(v-model="form.v4zt")
                el-form-item(label="IPv6")
                  .flex.gap-15px
                    el-checkbox(v-model="form.v6plane") 6PLANE
                    el-checkbox(v-model="form.rfc4193") RFC4193
                    el-checkbox(v-model="form.v6zt") zt
                el-form-item(label="IP 分配池")
                  .w-full
                    .flex.items-center.gap-8px.mb-8px(v-for="(pool, i) in form.pools" :key="'p' + i")
                      el-input(v-model="pool.ipRangeStart" placeholder="起始 IP")
                      span -
                      el-input(v-model="pool.ipRangeEnd" placeholder="结束 IP")
                      el-button(type="danger" link @click="form.pools.splice(i, 1)") 删除
                    el-button(link type="primary" @click="form.pools.push({ ipRangeStart: '', ipRangeEnd: '' })") + 添加池

                el-divider(content-position="left") 路由
                el-form-item(label="转发路由")
                  .w-full
                    .flex.items-center.gap-8px.mb-8px(v-for="(r, i) in form.routes" :key="'r' + i")
                      el-input(v-model="r.target" placeholder="目标 如 10.0.0.0/24")
                      el-input(v-model="r.via" placeholder="via（可空）")
                      el-button(type="danger" link @click="form.routes.splice(i, 1)") 删除
                    el-button(link type="primary" @click="form.routes.push({ target: '', via: null })") + 添加路由

                el-divider(content-position="left") DNS
                el-form-item(label="域名")
                  el-input(v-model="form.dnsDomain" placeholder="如 example.zt")
                el-form-item(label="DNS 服务器")
                  el-input(v-model="form.dnsServers" placeholder="逗号分隔")
                el-form-item(label="搜索域")
                  el-input(v-model="form.dnsSearch" placeholder="逗号分隔")

                el-divider(content-position="left") 远程追踪 / 流规则
                el-form-item(label="远程追踪目标")
                  el-input(v-model="form.remoteTraceTarget" placeholder="节点地址，留空=关闭")
                el-form-item(label="远程追踪级别")
                  el-input-number(v-model="form.remoteTraceLevel" :min="0" :max="7")
                el-form-item(label="flowRules")
                  el-input(v-model="form.flowRules" type="textarea" :rows="3" placeholder="C++ 风格流规则，留空则不修改")

            // 右栏：完整配置 JSON（与左侧表单双向实时同步）+ 底部提示
            .config-side
              .side-head 完整配置（高级 · 与左侧表单双向实时同步）
              .editor-area
                code-editor(v-model="rawJson" fill @focus="jsonFocused = true" @blur="jsonFocused = false")
              .side-hint
                span.text-danger(v-if="jsonError") JSON 暂不可解析，已保留上次有效值：{{ jsonError }}
                span.text-gray-400(v-else) 编辑此处或左侧表单，两边自动同步；焦点在 JSON 上时以 JSON 为准

          // 底部固定操作栏
          .config-footer
            el-button(type="primary" :loading="savingConfig" @click="saveConfig") 保存配置
            el-button(class="ml-10px" @click="resetForm") 重置

      // ===================== 原始数据 =====================
      el-tab-pane.h-full(label="原始数据" name="raw")
        .p-15px.h-full(overflow="hidden")
          code-editor(:model-value="jsonDump" readonly fill)
</template>

<script setup>
  import { useForgeApi } from '@route-forge/vue';
  import { ElMessage } from 'element-plus';
  import { CopyDocument, Refresh } from '@element-plus/icons-vue';
  import CodeEditor from '@/components/CodeEditor.vue';

  const route = useRoute();
  const nwid = route.params.nwid;
  const activeTab = ref('members');

  const network = ref({});
  const members = ref([]);
  const { pending: pendingMembers, call: callMember } = useForgeApi('admin');
  const { pending: pendingConfig, call: callNet } = useForgeApi('admin');
  const savingConfig = ref(false);
  const rawJson = ref('{}');
  // JSON 编辑器交互态：焦点在其中时暂停「表单→JSON」回写，避免打断手改；解析失败时红字提示并保留上次有效值
  const jsonFocused = ref(false);
  const jsonError = ref('');
  // 内部同步守卫（非响应式即可）：区分「用户编辑」与「程序回写」，打断表单↔JSON 双向 watch 的回环
  let internalSync = false;

  // 完整 config 对象：表单字段投影到它，保存时整体透传，未编辑字段原样保留
  const fullConfig = ref({});

  const jsonDump = computed(() => JSON.stringify(network.value, null, 2));

  const form = reactive({
    name: '',
    active: true,
    private: true,
    enableBroadcast: true,
    mtu: 2800,
    multicastLimit: 32,
    multicastTTL: 128,
    v4zt: true,
    v6plane: false,
    rfc4193: false,
    v6zt: false,
    pools: [],
    routes: [],
    dnsDomain: '',
    dnsServers: '',
    dnsSearch: '',
    remoteTraceTarget: '',
    remoteTraceLevel: 0,
    flowRules: '',
  });

  onMounted(load);

  async function load() {
    await Promise.all([loadNetwork(), loadMembers()]);
  }

  async function loadNetwork() {
    const { data, error } = await callNet('networks.show', { nwid });
    if (error || !data) return;
    network.value = data;
    reloadUiFromConfig(data);
  }

  // 深拷贝：structuredClone 无法处理 Vue reactive Proxy，config 全为纯 JSON，用 JSON round-trip 最稳
  function deepClone(v) {
    return v === undefined ? {} : JSON.parse(JSON.stringify(v));
  }

  // 把一份完整 config 灌回界面：同步 fullConfig + 表单投影 + JSON 文本（作为初始/重置，屏蔽同步回环）
  function reloadUiFromConfig(source) {
    fullConfig.value = deepClone(source);
    jsonError.value = '';
    internalSync = true;
    projectFormFromConfig(fullConfig.value);
    rawJson.value = JSON.stringify(fullConfig.value, null, 2);
    deferResetSync();
  }

  // config 对象 → 表单字段投影（初始/加载/JSON 应用/重置共用一份）
  function projectFormFromConfig(n) {
    form.name = n.name ?? '';
    form.active = n.active ?? true;
    form.private = !!n.private;
    form.enableBroadcast = n.enableBroadcast ?? true;
    form.mtu = n.mtu ?? 2800;
    form.multicastLimit = n.multicastLimit ?? 32;
    form.multicastTTL = n.multicastTTL ?? 128;
    form.v4zt = !!n.v4AssignMode?.zt;
    form.v6plane = !!n.v6AssignMode?.['6plane'];
    form.rfc4193 = !!n.v6AssignMode?.rfc4193;
    form.v6zt = !!n.v6AssignMode?.zt;
    form.pools = (n.ipAssignmentPools || []).map((p) => ({ ...p }));
    form.routes = (n.routes || []).map((r) => ({ target: r.target, via: r.via ?? null }));
    form.dnsDomain = n.dns?.domain || '';
    form.dnsServers = (n.dns?.servers || []).join(',');
    form.dnsSearch = (n.dns?.searchDomains || []).join(',');
    form.remoteTraceTarget = n.remoteTraceTarget || '';
    form.remoteTraceLevel = n.remoteTraceLevel ?? 0;
    form.flowRules = n.flowRules || '';
  }

  // 表单 → config（在 fullConfig 基础上只覆盖结构化字段，JSON 里编辑过的任意额外键原样保留）
  function configFromForm() {
    const cfg = deepClone(fullConfig.value || {});
    cfg.name = form.name;
    cfg.active = form.active;
    cfg.private = form.private;
    cfg.enableBroadcast = form.enableBroadcast;
    cfg.mtu = form.mtu;
    cfg.multicastLimit = form.multicastLimit;
    cfg.multicastTTL = form.multicastTTL;
    cfg.v4AssignMode = { zt: form.v4zt };
    cfg.v6AssignMode = { '6plane': form.v6plane, rfc4193: form.rfc4193, zt: form.v6zt };
    cfg.ipAssignmentPools = form.pools.filter((p) => p.ipRangeStart && p.ipRangeEnd);
    cfg.routes = form.routes.filter((r) => r.target).map((r) => ({ target: r.target, via: r.via || null }));
    cfg.remoteTraceTarget = form.remoteTraceTarget || null;
    cfg.remoteTraceLevel = form.remoteTraceLevel;
    const servers = form.dnsServers ? form.dnsServers.split(',').map((s) => s.trim()).filter(Boolean) : [];
    const search = form.dnsSearch ? form.dnsSearch.split(',').map((s) => s.trim()).filter(Boolean) : [];
    if (form.dnsDomain || servers.length || search.length) {
      cfg.dns = { domain: form.dnsDomain || null, servers, searchDomains: search };
    } else {
      delete cfg.dns;
    }
    if (form.flowRules) cfg.flowRules = form.flowRules;
    return cfg;
  }

  // 程序化写值期间置 internalSync，屏蔽随后同批次被触发的 watch，避免双向回环
  function deferResetSync() {
    nextTick(() => {
      internalSync = false;
    });
  }

  // 表单变化 → 实时回写 JSON（用户焦点在 JSON 编辑器时暂停，避免打断手改）
  watch(
    form,
    () => {
      if (internalSync || jsonFocused.value) return;
      const cfg = configFromForm();
      fullConfig.value = cfg;
      internalSync = true;
      rawJson.value = JSON.stringify(cfg, null, 2);
      deferResetSync();
    },
    { deep: true },
  );

  // JSON 文本变化（用户编辑）→ 防抖解析回写表单；解析失败保留上次有效值并红字提示
  let jsonTimer = null;
  watch(rawJson, () => {
    if (internalSync) return;
    if (jsonTimer) clearTimeout(jsonTimer);
    jsonTimer = setTimeout(applyJson, 400);
  });

  function applyJson() {
    jsonTimer = null;
    let parsed;
    try {
      parsed = JSON.parse(rawJson.value);
    } catch (e) {
      jsonError.value = e.message;
      return;
    }
    jsonError.value = '';
    internalSync = true;
    fullConfig.value = parsed;
    projectFormFromConfig(parsed);
    deferResetSync();
  }

  // 重置：丢弃未保存改动，回到最近一次从控制器读到的配置
  function resetForm() {
    if (jsonTimer) {
      clearTimeout(jsonTimer);
      jsonTimer = null;
    }
    reloadUiFromConfig(network.value);
    ElMessage.info('已重置为服务器配置');
  }

  async function saveConfig() {
    // 若 JSON 防抖尚未落定，先冲刷，确保表单/JSON 的最新编辑都进入 fullConfig
    if (jsonTimer) {
      clearTimeout(jsonTimer);
      jsonTimer = null;
      applyJson();
    }
    if (jsonError.value) {
      ElMessage.error('JSON 无效，无法保存：' + jsonError.value);
      return;
    }
    const body = fullConfig.value;
    savingConfig.value = true;
    const { error } = await callNet('networks.update', { nwid, body });
    savingConfig.value = false;
    if (error) {
      ElMessage.error(error.message || '保存失败');
      return;
    }
    ElMessage.success('配置已保存');
    loadNetwork();
  }

  async function loadMembers() {
    const { data, error } = await callMember('networks.members', { nwid });
    if (error) return;
    const arr = Array.isArray(data) ? data : Object.values(data || {});
    members.value = arr.map((m) => normalizeMember(m));
  }

  function normalizeMember(m) {
    const ipList = (m.ipAssignments || [])
      .map((ip) => (typeof ip === 'string' ? ip : ip.address))
      .filter(Boolean);
    return {
      ...m,
      ipList,
      name: m.name || '',
      authorized: !!m.authorized,
      activeBridge: !!m.activeBridge,
      noAutoAssignIps: !!m.noAutoAssignIps,
      hidden: !!m.hidden,
      flowRules: m.flowRules || '',
      _dirty: false,
    };
  }

  function dirty(row) {
    row._dirty = true;
  }
  function addIp(row) {
    const ip = prompt('输入要分配的 IP');
    if (ip) {
      row.ipList.push(ip.trim());
      dirty(row);
    }
  }
  function removeIp(row, i) {
    row.ipList.splice(i, 1);
    dirty(row);
  }

  async function save(row) {
    const { error } = await callMember('networks.members.update', {
      nwid,
      id: row.nodeId || row.address,
      body: {
        name: row.name,
        authorized: row.authorized,
        activeBridge: row.activeBridge,
        noAutoAssignIps: row.noAutoAssignIps,
        hidden: row.hidden,
        flowRules: row.flowRules || undefined,
        ipAssignments: row.ipList,
      },
    });
    if (error) {
      ElMessage.error(error.message || '保存失败');
      return;
    }
    row._dirty = false;
    ElMessage.success('已保存');
    loadMembers();
  }

  async function delMember(nodeId) {
    const { error } = await callMember('networks.members.destroy', { nwid, id: nodeId });
    if (error) {
      ElMessage.error(error.message || '移除失败');
      return;
    }
    ElMessage.success('已移除');
    loadMembers();
  }

  function formatTime(ts) {
    return ts ? new Date(Number(ts)).toLocaleString() : '-';
  }
  function copyNwid() {
    navigator.clipboard.writeText(nwid);
    ElMessage.success('已复制');
  }
</script>

<style scoped>
  /* 列布局：内容区吃满剩余高度，底部操作栏固定在末尾 */
  .config-pane {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .config-body {
    flex: 1;
    min-height: 0;
    display: flex;
    align-items: stretch;
  }
  /* 左右两栏：各占一半，宽度不足时各自保留最小宽度；高度由 flex 拉伸/分配，无需显式 height */
  .config-col,
  .config-side {
    flex: 1 1 0;
    min-width: 0;
    min-height: 0;
  }
  .label-col {
    padding: 16px 32px 16px 16px;
    box-sizing: border-box;
  }
  /* 左栏 el-scrollbar 撑满高度以启用内部滚动 */
  .config-col :deep(.el-scrollbar__wrap) {
    height: 100%;
  }
  /* 右栏纵向三段：标题固定 / 编辑器填充并内部滚动 / 提示固定 */
  .config-side {
    display: flex;
    flex-direction: column;
    border-left: 1px solid var(--el-border-color-lighter);
  }
  .side-head {
    flex: 0 0 auto;
    padding: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
  .editor-area {
    flex: 1;
    min-height: 0;
    padding: 0 16px;
  }
  .editor-area > * {
    height: 100%;
  }
  .side-hint {
    flex: 0 0 auto;
    padding: 6px 16px 12px;
    font-size: 12px;
    line-height: 1.5;
  }
  /* 底部固定操作栏 */
  .config-footer {
    flex: 0 0 auto;
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    padding: 12px 16px;
    border-top: 1px solid var(--el-border-color-lighter);
    background: var(--el-bg-color);
  }
  /* 窄屏：上下堆叠，两栏各占一半高度并仍各自独立滚动 */
  @media (max-width: 900px) {
    .config-body {
      flex-direction: column;
    }
    .config-side {
      border-left: none;
      border-top: 1px solid var(--el-border-color-lighter);
    }
  }
</style>
