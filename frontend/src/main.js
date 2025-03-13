import "./monaco";
import App from './views/App.svelte';
import { GetWindowInfo } from '../wailsjs/go/app/api';

let app;

document.addEventListener("DOMContentLoaded", () => {
    GetWindowInfo().then(info => {
        window.isWin = info.isWindows;
        app = new App({
            target: document.body,
        });
    });
   
    // Apply production-only behaviors
    if (false) {
        window.addEventListener('contextmenu', e => e.preventDefault());
    }
});

    let zoomLevel = 1.0;

    function zoomIn() {
      zoomLevel += 0.1;
      applyZoom();
    }

    function zoomOut() {
      zoomLevel -= 0.1;
      applyZoom();
    }

    function resetZoom() {
      zoomLevel = 1.0;
      applyZoom();
    }

    function applyZoom() {
      document.body.style.transform = `scale(${zoomLevel})`;
      document.body.style.transformOrigin = '0 0';
    }

    document.addEventListener('keydown', (event) => {
      if (event.ctrlKey && (event.key === '+' || event.key === '=')) {
        event.preventDefault(); // Prevent default browser zoom
        zoomIn();
      } else if (event.ctrlKey && event.key === '-') {
        event.preventDefault(); // Prevent default browser zoom
        zoomOut();
      } else if (event.ctrlKey && event.key === '0') {
        event.preventDefault();
        resetZoom();
      }
    });

export default app;
