import { createApp } from "vue";
import { createPinia } from "pinia";
import piniaPluginPersistedState from "pinia-plugin-persistedstate";
import router from "./routes";

async function loadBoostrapIcons() {
    if (import.meta.env.DEV) {
        const cssUrl = new URL('bootstrap-icons/font/bootstrap-icons.css', import.meta.url).href;
        const link = document.createElement("link");
        link.rel = 'stylesheet';
        link.href = cssUrl;
        document.head.appendChild(link);

        return;
    }

    await import("bootstrap-icons/font/bootstrap-icons.css");
}

await loadBoostrapIcons();
import "bootstrap/dist/css/bootstrap.min.css";
import "bootstrap";

import "./style.css";
import App from "./App.vue";

const pinia = createPinia();
pinia.use(piniaPluginPersistedState);

createApp(App).use(pinia).use(router).mount("#app");
