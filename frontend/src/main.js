import "./monaco";
import App from './views/App.svelte';
import { WindowGetInfo } from '@wailsjs/runtime/runtime';

let app;

// Initialize the app directly - no more Wails.Init
document.addEventListener("DOMContentLoaded", () => {
    // Check platform - Wails v2 has built-in platform detection
    WindowGetInfo().then(info => {
        window.isWin = info.platform === "windows";
        
        // Initialize your Svelte app
        app = new App({
            target: document.body,
        });
    });
    
    // In Wails v2, you can still prevent context menu in production
    // This can be handled through the build tags or runtime checks
    if (process.env.NODE_ENV === "production") {
        window.addEventListener('contextmenu', e => e.preventDefault());
    }
});

export default app;
