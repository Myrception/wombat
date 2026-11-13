// Only the editor core and the Monarch tokenizers we need are bundled; the
// full "monaco-editor" entry would pull in every language and service.
import * as monaco from "monaco-editor/editor/editor.api";
import "monaco-editor/languages/definitions/javascript/register";
import "monaco-editor/languages/definitions/shell/register";
import "./monaco.css";

export { monaco };

// Options shared by all editors. Newer Monaco features are turned off to keep
// the editors as plain as they were.
export const baseOptions = {
    theme: "nord-dark",
    fontSize: 14,
    minimap: { enabled: false },
    wordWrap: "on",
    links: false,
    matchBrackets: "never",
    guides: { indentation: false },
    bracketPairColorization: { enabled: false },
    stickyScroll: { enabled: false },
    renderValidationDecorations: "off",
    scrollBeyondLastLine: false,
    selectionHighlight: false,
    automaticLayout: true,
    hideCursorInOverviewRuler: true,
    overviewRulerBorder: false,
    padding: { top: 12, bottom: 12 },
    scrollbar: { useShadows: false },
};

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
