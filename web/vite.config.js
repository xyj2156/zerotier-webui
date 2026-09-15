import { defineConfig } from 'vite';
import unoCss from 'unocss/vite';
import pug from 'vite-plugin-pug';
import vue from '@vitejs/plugin-vue';
import AutoImport from 'unplugin-auto-import/vite';
import Components from 'unplugin-vue-components/vite';
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers';

// 单执行文件形态：前端构建产物输出到 dist/，由 Go 侧 go:embed 打进二进制。
// 开发期 vite dev server 把 /api 与 /_forge 代理给 go run 起的本地桥（默认 9090）。
export default defineConfig({
  plugins: [
    unoCss(),
    pug(),
    vue(),
    AutoImport({
      imports: ['vue', 'vue-router', 'pinia'],
      resolvers: [ElementPlusResolver()],
      dts: true,
    }),
    Components({
      resolvers: [ElementPlusResolver()],
      dts: true,
    }),
  ],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:9090',
      '/_forge': 'http://127.0.0.1:9090',
    },
  },
  resolve: {
    alias: {
      '@': '/src',
    },
  },
});
