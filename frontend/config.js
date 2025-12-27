// Config application logic for Fenestro
// Handles applying configuration values to the DOM

/**
 * Apply font size configuration to a content element.
 *
 * @param {Object} config - The config object from the backend
 * @param {HTMLElement} contentElement - The element to apply font size to
 * @returns {boolean} True if font size was applied, false otherwise
 */
export function applyFontSize(config, contentElement) {
    if (config && config.font_size && config.font_size > 0) {
        contentElement.style.fontSize = config.font_size + 'px';
        return true;
    }
    return false;
}

/**
 * Inject custom chrome CSS into the document.
 *
 * @param {string} chromeCSS - The CSS content to inject
 * @param {Document} targetDocument - The document to inject into (default: document)
 * @returns {HTMLStyleElement|null} The created style element, or null if no CSS provided
 */
export function injectChromeCSS(chromeCSS, targetDocument = document) {
    if (!chromeCSS) {
        return null;
    }

    // Remove existing chrome CSS if present
    const existing = targetDocument.getElementById('fenestro-chrome-css');
    if (existing) {
        existing.remove();
    }

    const style = targetDocument.createElement('style');
    style.id = 'fenestro-chrome-css';
    style.textContent = chromeCSS;
    targetDocument.head.appendChild(style);
    return style;
}
