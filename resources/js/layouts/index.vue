<script setup>
import { ElConfigProvider } from "element-plus";
import zhCN from "element-plus/es/locale/lang/zh-cn";
</script>

<template lang="pug">
  el-config-provider(:locale="zhCN" )
    router-view
</template>

<style scoped></style>
