// frontend/src/main.js - Enhanced zoom functionality

import "./monaco";
import App from './views/App.svelte';
import { GetWindowInfo } from '../wailsjs/go/app/api';
import { EventsEmit } from '../wailsjs/runtime/runtime';

let app;

document.addEventListener("DOMContentLoaded", () => {
    GetWindowInfo().then(info => {
        window.isWin = info.isWindows;
        
        initializeZoom();
        
        app = new App({
            target: document.body,
        });
    });
});

let zoomLevel = 1.0;
const ZOOM_STEP = 0.1;
const MAX_ZOOM = 3.0;
const MIN_ZOOM = 0.5;

function initializeZoom() {
    const screenWidth = window.screen.width;
    const screenHeight = window.screen.height;
    
    if (screenWidth >= 3840 || screenHeight >= 2160) {
        // 4K displays (3840×2160 or higher)
        zoomLevel = 1.5;
    } else if (screenWidth >= 2560 || screenHeight >= 1440) {
        // 2K/QHD displays (2560×1440 or higher)
        zoomLevel = 1.25;
    }
    
    applyZoom();
    
    saveZoomLevel();
}

function zoomIn() {
    if (zoomLevel < MAX_ZOOM) {
        zoomLevel = Math.min(MAX_ZOOM, zoomLevel + ZOOM_STEP);
        applyZoom();
        saveZoomLevel();
        notifyZoomChange();
    }
}

function zoomOut() {
    if (zoomLevel > MIN_ZOOM) {
        zoomLevel = Math.max(MIN_ZOOM, zoomLevel - ZOOM_STEP);
        applyZoom();
        saveZoomLevel();
        notifyZoomChange();
    }
}

function resetZoom() {
    // Use the initially calculated zoom level based on resolution instead of always 1.0
    initializeZoom();
    notifyZoomChange();
}

function applyZoom() {
     // Apply zoom using CSS variables instead of transform to prevent layout issues
    document.documentElement.style.setProperty('--app-scale', zoomLevel); 
    // Update font sizes and other scalable properties
    document.documentElement.style.fontSize = `${zoomLevel * 10}pt`;
    
    // Notify components about zoom change
    EventsEmit("wombat:zoom_changed", zoomLevel);
}

function adjustContainerHeight() {
    // Adjust the height of the container to account for zooming
    // This prevents unnecessary scrollbars and clipping
    const viewportHeight = window.innerHeight;
    const scaledHeight = viewportHeight / zoomLevel;
    document.body.style.height = `${scaledHeight}px`;
}

// Notify the app about zoom changes to update UI components that need to know the zoom level
function notifyZoomChange() {
    EventsEmit("wombat:zoom_changed", zoomLevel);
}

// Save the zoom level to localStorage for persistence
function saveZoomLevel() {
    localStorage.setItem('wombat_zoom_level', zoomLevel);
}

// Load saved zoom level from localStorage
function loadZoomLevel() {
    const savedZoom = localStorage.getItem('wombat_zoom_level');
    if (savedZoom !== null) {
        zoomLevel = parseFloat(savedZoom);
        applyZoom();
    }
}

// Try to load saved zoom level, if available
loadZoomLevel();

function updateSplitPaneDividers() {
  // Find all split pane dividers
  const dividers = document.querySelectorAll('.divider');
  
  // Force a reflow on each divider
  dividers.forEach(divider => {
    // Get the parent container dimensions
    const container = divider.closest('.container');
    if (container) {
      // For horizontal dividers, ensure height matches container
      if (divider.classList.contains('horizontal')) {
        divider.style.height = container.offsetHeight + 'px';
      }
      // For vertical dividers, ensure width matches container
      else if (divider.classList.contains('vertical')) {
        divider.style.width = container.offsetWidth + 'px';
      }
    }
  });
}

// Handle window resize to adjust container height
window.addEventListener('resize', () => {
  adjustContainerHeight();
  setTimeout(updateSplitPaneDividers, 100);
  location.reload(); 
  window.dispatchEvent(new CustomEvent('wombat:window-resized'));
});

window.updateSplitPaneDividers = updateSplitPaneDividers;

// Keep the existing keyboard shortcuts
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

// Expose zoom functions globally for access from UI
window.appZoom = {
    zoomIn,
    zoomOut,
    resetZoom,
    getZoomLevel: () => zoomLevel
};

export default app;
