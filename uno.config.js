import {
    defineConfig,
    presetAttributify,
    presetIcons,
    presetWind3,
} from "unocss";
import extractorPug from "@unocss/extractor-pug";

export default defineConfig({
    presets: [
        presetWind3(), // 基础原子类（如 p-4, text-center）
        presetAttributify(), // 属性模式（如 <div text="red-500" />）
        presetIcons(), // 图标支持（如 <i class="i-mdi-home" />）
    ],
    extractors: [extractorPug()],
});
