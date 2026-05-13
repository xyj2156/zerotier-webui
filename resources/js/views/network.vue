<template lang="pug">
  el-card
    template(#header)
      span 我的网络
      el-button(
        type="primary"
        size="small"
        style="float:right"
        @click="dialog.show()"
      )
        el-icon
          Plus
        | 新建网络

    el-table(:data="networks" border stripe)
      el-table-column(label="网络ID" prop="nwid")
        template(#default="{row}")
          el-button(link type="primary" @click="redirect(row.id)") {{ row.nwid }}
      el-table-column(label="名称" prop="name")
      el-table-column(label="类型")
        template(#default="scope")
          el-tag(type="primary" v-if="scope.row.private") 私有
          el-tag(type="warning" v-else) 公开
  el-dialog(v-model="dialog.visible" :close-on-click-modal="false")
    template(#header) 新建网络
    el-form(label-width="auto" size="small")
      el-form-item(label="网络名称")
        el-input(v-model="dialog.data.name" placeholder="请输入网络名称")
      el-form-item(label="网络类型")
        el-select(v-model="dialog.data.type" placeholder="请选择网络类型")
          el-option(label="私有" value="private")
          el-option(label="公开" value="public")
      el-form-item(label="CIDR")
        el-input(v-model="dialog.data.cidr" placeholder="请输入CIDR" @input="CIDRtoPool(dialog.data.cidr)")
          template(#append)
            el-button(type="primary" @click="randomIPv4") 随机生成
      el-form-item(label="ip分配池" )
        .flex.items-center(class="gap-10px")
          el-input(v-model="dialog.data.poolStart" placeholder="请输入CIDR" readonly)
          el-input(v-model="dialog.data.poolEnd" placeholder="请输入CIDR" readonly)
    template(#footer)
      el-button(type="primary" @click="dialog.confirm") 确定
      el-button(type="danger" @click="dialog.visible = false;") 取消
</template>

<script setup>
  import { onMounted, ref } from 'vue';
  import axios from '@/utils/fetch.js';
  import { ElMessage } from 'element-plus';
  import { Plus } from '@element-plus/icons-vue';
  import router from '@/routes/index.js';

  const networks = ref([]);

  const dialog = reactive({
    visible: false,
    data: { name: '', type: 'private', cidr: '', poolStart: '', poolEnd: '' },
    show() {
      dialog.data = {
        name: '',
        type: 'private',
        cidr: '',
        poolStart: '',
        poolEnd: '',
      };
      dialog.visible = true;
    },
    confirm() {
      axios
        .post('/api/networks', dialog.data)
        .then(() => {
          ElMessage.success('创建成功');
          dialog.visible = false;
          load();
        })
        .catch((err) => {
          ElMessage.error(err.message);
        });
    },
  });

  onMounted(load);

  async function load() {
    const res = await axios.get('/api/networks');
    networks.value = res;
  }

  function randomOctet() {
    return Math.floor(Math.random() * 255);
  }

  function randomIPv4() {
    dialog.data.cidr = '10.' + randomOctet() + '.' + randomOctet() + '.0/24';
    CIDRtoPool(dialog.data.cidr);
  }

  const CIDRtoPool = (function () {
    let timer;
    return function (CIDR) {
      clearTimeout(timer);
      timer = setTimeout(function () {
        const [start, prefix] = CIDR.split('/');
        if (
          undefined !== start &&
          undefined !== prefix &&
          prefix > 0 &&
          prefix < 33 &&
          /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/.test(
            start
          )
        ) {
          const host32 = ((1 << (32 - parseInt(prefix))) - 1) >>> 0;
          const net = start.split('.').map((oct) => {
            return parseInt(oct);
          });
          let net32 = 0 >>> 0;
          net32 = (net[0] << 24) + (net[1] << 16) + (net[2] << 8) + net[3];
          net32 &= ~host32;
          const bcast32 = net32 + host32;
          dialog.data.cidr = int32toIPv4String(net32) + '/' + prefix;
          dialog.data.poolStart = int32toIPv4String(net32 + 1);
          dialog.data.poolEnd = int32toIPv4String(bcast32 - 1);
        }
      }, 600);
    };
  })();

  function int32toIPv4String(int32) {
    let ipv4 = '';
    ipv4 = ((int32 & 0xff000000) >>> 24).toString();
    ipv4 += '.' + ((int32 & 0x00ff0000) >>> 16).toString();
    ipv4 += '.' + ((int32 & 0x0000ff00) >>> 8).toString();
    ipv4 += '.' + (int32 & 0x000000ff).toString();
    return ipv4;
  }

  function redirect(id) {
    router.push(`/network/${id}`);
  }
</script>
