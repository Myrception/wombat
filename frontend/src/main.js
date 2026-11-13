import "./monaco";
import "./zoom";
import App from './views/App.svelte';
import { FrontendReady } from '../wailsjs/go/app/api';

let app;

function start() {
    app = new App({
        target: document.body,
    });

    // All components are mounted and subscribed to backend events now,
    // so the backend can make the initial connection.
    FrontendReady();
}

if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", start);
} else {
    start();
}

export default app;
