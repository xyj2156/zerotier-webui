import "virtual:uno.css";
import { createApp } from "vue";
import router from "./routes";
import { createPinia } from "pinia";
import main from "@/main.vue";

const app = createApp(main);

app.use(router);
app.use(createPinia());
app.mount("#jason");
