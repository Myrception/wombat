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
    zoomLevel = 1.0; // Always reset to 1.0 for consistency
    applyZoom();
    saveZoomLevel();
    notifyZoomChange();
}

function adjustFixedPositionElements(zoomLevel) {
  // Find all elements that might need position adjustment on zoom
  const fixedElements = document.querySelectorAll('.fixed-position, [data-fixed-position]');
  
  fixedElements.forEach(element => {
    const zoomFactor = 1 / zoomLevel;
    
    // Apply inverse positioning to counteract the zoom effect
    if (element.dataset.originalBottom === undefined) {
      // Store original positions if not already stored
      element.dataset.originalBottom = window.getComputedStyle(element).bottom;
      element.dataset.originalRight = window.getComputedStyle(element).right;
      element.dataset.originalTop = window.getComputedStyle(element).top;
      element.dataset.originalLeft = window.getComputedStyle(element).left;
    }
    
    // Apply adjusted positions
    if (element.dataset.originalBottom && element.dataset.originalBottom !== 'auto') {
      const originalBottom = parseFloat(element.dataset.originalBottom);
      element.style.bottom = `${originalBottom * zoomFactor}px`;
    }
    
    if (element.dataset.originalRight && element.dataset.originalRight !== 'auto') {
      const originalRight = parseFloat(element.dataset.originalRight);
      element.style.right = `${originalRight * zoomFactor}px`;
    }
    
    if (element.dataset.originalTop && element.dataset.originalTop !== 'auto') {
      const originalTop = parseFloat(element.dataset.originalTop);
      element.style.top = `${originalTop * zoomFactor}px`;
    }
    
    if (element.dataset.originalLeft && element.dataset.originalLeft !== 'auto') {
      const originalLeft = parseFloat(element.dataset.originalLeft);
      element.style.left = `${originalLeft * zoomFactor}px`;
    }
  });
  
  // Specific handling for Edit button which might not be caught by the selector above
  const editBtn = document.querySelector('.edit');
  if (editBtn) {
    const zoomFactor = 1 / zoomLevel;
    editBtn.style.bottom = `calc(var(--padding) * ${zoomFactor})`;
    editBtn.style.right = `calc(var(--padding) * ${zoomFactor})`;
  }
}

function applyZoom() {
    // Direct transform approach - scales the entire UI consistently
    document.body.style.transform = `scale(${zoomLevel})`;
    document.body.style.transformOrigin = 'top left';
    
    // Adjust the body dimensions to account for the scaling
    document.body.style.width = `${100 / zoomLevel}%`;
    document.body.style.height = `${100 / zoomLevel}vh`;
    
    // Store the zoom level for components that need to know it
    document.body.dataset.zoomLevel = zoomLevel;
    
    // Also set CSS variables for components that use them
    document.documentElement.style.setProperty('--app-scale', zoomLevel);
   
    adjustFixedPositionElements(zoomLevel);

    // Notify components about zoom change
    EventsEmit("wombat:zoom_changed", zoomLevel);
}

function initializeZoomAttributes() {
    document.querySelectorAll('.needs-zoom').forEach((element, index) => {
        if (!element.hasAttribute('data-zoom-scale')) {
            element.setAttribute('data-zoom-scale', '1');
        }
    });
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
  setTimeout(() => {
        updateSplitPaneDividers();
        window.dispatchEvent(new CustomEvent('wombat:window-resized'));
    }, 100);
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
