// 构建后补回 dist/.gitkeep。
//
// Vite 的 emptyOutDir 每次都会清空 dist/，而 go:embed all:dist 要求该目录至少存在一个可嵌入文件
// （否则新 clone 未跑 pnpm build 就 go build 会直接失败）。这里把它补回来，让占位文件不会被构建过程带走。
import { mkdirSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const keep = resolve(here, '../dist/.gitkeep');

mkdirSync(dirname(keep), { recursive: true });
writeFileSync(
  keep,
  '# 占位：保证未构建前端时 go:embed all:dist 仍可编译；真实产物由 pnpm build 生成。\n'
);
