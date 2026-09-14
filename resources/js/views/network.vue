<template lang="pug">
  el-card(class="h-[calc(100%-2px)]" ref="cardRef")
    template(#header)
      .flex.justify-between
        .flex.items-center
          span 我的网络
          el-popover(content="刷新" placement="top")
            template(#reference)
              el-icon.cursor-pointer.text-primary.ml-10px(@click="load")
                Refresh
        .float-right
          el-button(type="primary" size="small" @click="dialog.show()")
            el-icon
              Plus
            | 新建网络

    el-table(:data="networks" border stripe v-loading="pending" :max-height="tableMaxHeight")
      el-table-column(label="网络ID" prop="nwid" width="200")
        template(#default="{ row }")
          el-button(link type="primary" @click="redirect(row.nwid)") {{ row.nwid }}
      el-table-column(label="名称" prop="name")
      el-table-column(label="类型" width="100")
        template(#default="{ row }")
          el-tag(v-if="row.private" type="primary") 私有
          el-tag(v-else type="warning") 公开
      el-table-column(label="状态" width="100")
        template(#default="{ row }")
          el-tag(v-if="row.active" type="success") 启用
          el-tag(v-else type="info") 停用
      el-table-column(label="创建时间" width="200")
        template(#default="{ row }") {{ formatTime(row.creationTime) }}
      el-table-column(label="操作" width="140" fixed="right")
        template(#default="{ row }")
          el-button(type="primary" link size="small" @click="redirect(row.nwid)") 管理
          el-popconfirm(title="确定删除该网络？" @confirm="del(row.nwid)")
            template(#reference)
              el-button(type="danger" link size="small" class="ml-10px") 删除

  el-dialog(v-model="dialog.visible" :close-on-click-modal="false" append-to-body width="520px")
    template(#header) 新建网络
    el-form(label-width="120px" size="default")
      el-form-item(label="网络名称")
        el-input(v-model="dialog.data.name" placeholder="请输入网络名称")
      el-form-item(label="网络类型")
        el-radio-group(v-model="dialog.data.private")
          el-radio(:value="true") 私有
          el-radio(:value="false") 公开
      el-form-item(label="CIDR")
        el-input(v-model="dialog.data.cidr" placeholder="如 10.147.20.0/24" @input="onCidr")
          template(#append)
            el-button(@click="randomIPv4") 随机
      el-form-item(label="IP 分配池")
        .flex.items-center.gap-10px
          el-input(v-model="dialog.poolStart" placeholder="起始" readonly)
          span -
          el-input(v-model="dialog.poolEnd" placeholder="结束" readonly)
      el-divider()
      el-form-item(label="MTU")
        el-input-number(v-model="dialog.data.mtu" :min="1280" :max="2800" :step="1")
      el-form-item(label="组播上限")
        el-input-number(v-model="dialog.data.multicastLimit" :min="1" :max="255")
      el-form-item(label="组播 TTL")
        el-input-number(v-model="dialog.data.multicastTTL" :min="1" :max="255")
      el-form-item(label="允许广播")
        el-switch(v-model="dialog.data.enableBroadcast")
      el-form-item(label="IPv4 自动分配")
        el-switch(v-model="dialog.v4zt")
      el-form-item(label="IPv6 自动分配")
        .flex.gap-15px
          el-checkbox(v-model="dialog.v6plane") 6PLANE
          el-checkbox(v-model="dialog.rfc4193") RFC4193
    template(#footer)
      el-button(@click="dialog.visible = false") 取消
      el-button(type="primary" :loading="pending" @click="dialog.confirm") 确定
</template>

<script setup>
  import { useForgeApi } from '@route-forge/vue';
  import { ElMessage } from 'element-plus';
  import { Plus, Refresh } from '@element-plus/icons-vue';
  import router from '@/routes/index.js';

  const cardRef = ref(null);
  const tableMaxHeight = ref('');
  const networks = ref([]);
  const { pending, call } = useForgeApi('admin');

  function blank() {
    return {
      data: {
        visible: false,
        name: '',
        private: true,
        cidr: '',
        mtu: 2800,
        multicastLimit: 32,
        multicastTTL: 128,
        enableBroadcast: true,
        v4zt: true,
        v6plane: false,
        rfc4193: false,
      },
    };
  }
  const dialog = reactive(blank());

  dialog.show = function () {
    Object.assign(dialog, blank(), { visible: true });
  };
  dialog.confirm = async function () {
    const body = {
      name: dialog.name,
      private: dialog.private,
      cidr: dialog.cidr,
      mtu: dialog.mtu,
      multicastLimit: dialog.multicastLimit,
      multicastTTL: dialog.multicastTTL,
      enableBroadcast: dialog.enableBroadcast,
      v4AssignMode: { zt: dialog.v4zt },
      v6AssignMode: {
        '6plane': dialog.v6plane,
        rfc4193: dialog.rfc4193,
        zt: false,
      },
    };
    const { error } = await call('networks.store', { body });
    if (error) {
      ElMessage.error(error.message || '创建失败');
      return;
    }
    ElMessage.success('创建成功');
    dialog.visible = false;
    load();
  };

  onMounted(function () {
    load();
    if (cardRef?.value?.$el) {
      const ob = new ResizeObserver(function (entries) {
        entries.forEach(function (dom) {
          tableMaxHeight.value = dom.contentRect.height - 101 + 'px';
        });
      });
      ob.observe(cardRef.value.$el);
      onUnmounted(function () {
        ob.disconnect();
      });
    }
  });

  async function load() {
    const { data, error } = await call('networks');
    if (!error) networks.value = data || [];
  }

  async function del(nwid) {
    const { error } = await call('networks.destroy', { nwid });
    if (error) {
      ElMessage.error(error.message || '删除失败');
      return;
    }
    ElMessage.success('删除成功');
    load();
  }

  function randomOctet() {
    return Math.floor(Math.random() * 255);
  }
  function randomIPv4() {
    dialog.cidr = '10.' + randomOctet() + '.' + randomOctet() + '.0/24';
    onCidr(dialog.cidr);
  }

  const onCidr = (function () {
    let timer;
    return function (cidr) {
      clearTimeout(timer);
      timer = setTimeout(function () {
        const r = cidrToPool(cidr);
        dialog.poolStart = r ? r.start : '';
        dialog.poolEnd = r ? r.end : '';
      }, 400);
    };
  })();

  function cidrToPool(cidr) {
    const parts = (cidr || '').trim().split('/');
    if (parts.length !== 2) return null;
    const prefix = parseInt(parts[1], 10);
    const ip32 = ipToInt(parts[0]);
    if (ip32 === null || !(prefix >= 1 && prefix <= 31)) return null;
    const host = ((1 << (32 - prefix)) - 1) >>> 0;
    const net = (ip32 & ~host) >>> 0;
    const bcast = (net + host) >>> 0;
    return { start: intToIp(net + 1), end: intToIp(bcast - 1) };
  }
  function ipToInt(ip) {
    const m = ip.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/);
    if (!m) return null;
    const oct = m.slice(1).map(Number);
    if (oct.some((o) => o > 255)) return null;
    return (
      (((oct[0] << 24) >>> 0) + (oct[1] << 16) + (oct[2] << 8) + oct[3]) >>> 0
    );
  }
  function intToIp(n) {
    return [(n >>> 24) & 255, (n >>> 16) & 255, (n >>> 8) & 255, n & 255].join(
      '.'
    );
  }

  function formatTime(ts) {
    return ts ? new Date(Number(ts)).toLocaleString() : '-';
  }
  function redirect(nwid) {
    router.push(`/network/${nwid}`);
  }
</script>
