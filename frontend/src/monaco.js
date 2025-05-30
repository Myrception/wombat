import "monaco-editor/esm/vs/basic-languages/javascript/javascript.contribution";
import "monaco-editor/esm/vs/basic-languages/shell/shell.contribution";
import "monaco-editor/esm/vs/editor/editor.api";
import "./monaco.css";

monaco.editor.defineTheme("nord-dark", {
    base: "vs-dark",
    inherit: true,
    rules: [
        { token: "identifier", foreground: "#81a1c1" },
        { token: "string", foreground: "#a3be8c" },
        { token: "number", foreground: "#b48ead" },
        { token: "keyword", foreground: "#8fbcbb" },
        { token: "delimiter", foreground: "#88c0d0" },
        { token: "type.identifier", foreground: "#b48ead" }, // enum
    ],
    colors: {
        "foreground": "#eceff4",
        "editor.background": "#2e3440",
        "editor.selectionBackground": "#4c566a",
        "editor.inactiveSelectionBackground": "#434c5e",
        "editor.lineHighlightBorder": "#2e3440",
        "scrollbarSlider.background": "#3b4252",
        "scrollbarSlider.hoverBackground" : "#434c5e",
        "scrollbarSlider.activeBackground" : "#4c566a",
    },
    settings: {
        fontSize: 14,
        lineHeight: 1.5,
    }
});

document.head.insertAdjacentHTML('beforeend', `
<style>
    /* Improve scrollbar visibility on high-DPI displays */
    ::-webkit-scrollbar {
        width: 14px;
        height: 14px;
    }
    
    ::-webkit-scrollbar-track {
        background: var(--bg-color);
    }
    
    ::-webkit-scrollbar-thumb {
        background-color: var(--bg-color3);
        border-radius: 7px;
        border: 3px solid var(--bg-color);
    }
    
    ::-webkit-scrollbar-thumb:hover {
        background-color: var(--accent-color3);
    }
    
    /* Improve high contrast for text */
    @media screen and (min-resolution: 2dppx) {
        :root {
            --text-color3: #d8dee9;
        }
    }
</style>
`);
