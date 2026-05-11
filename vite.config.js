import { defineConfig } from "vite";
import laravel from "laravel-vite-plugin";
import { bunny } from "laravel-vite-plugin/fonts";
import unoCss from "unocss/vite";
import pug from "vite-plugin-pug";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
    plugins: [
        unoCss(),
        pug(),
        vue(),
        laravel({
            input: ["resources/styles/main.css", "resources/js/main.js"],
            refresh: true,
            fonts: [
                bunny("Instrument Sans", {
                    weights: [400, 500, 600],
                }),
            ],
        }),
    ],
    server: {
        watch: {
            ignored: ["**/storage/framework/views/**"],
        },
    },
    resolve: {
        alias: {
            "@": "/resources/js",
        },
    },
});
