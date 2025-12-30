// HTML rendering logic for Fenestro
// Handles full HTML documents with <head> and <body> tags

/**
 * Parse HTML and extract scripts, styles, and body content.
 * Uses DOMParser to properly handle full HTML documents.
 *
 * @param {string} html - The HTML string to parse
 * @returns {Object} Parsed content with scripts, styles, links, and bodyContent
 */
export function parseHTML(html) {
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, 'text/html');

    // Collect all scripts in document order (from both head and body)
    const scripts = [];
    doc.querySelectorAll('script').forEach(script => {
        scripts.push({
            src: script.getAttribute('src'),
            textContent: script.textContent,
            attributes: Array.from(script.attributes).map(attr => ({
                name: attr.name,
                value: attr.value
            }))
        });
        // Remove from parsed doc so they don't get inserted with innerHTML
        script.remove();
    });

    // Collect styles from head
    const styles = [];
    doc.querySelectorAll('head style').forEach(style => {
        styles.push(style.textContent);
    });

    // Collect link elements (stylesheets) from head
    const links = [];
    doc.querySelectorAll('head link[rel="stylesheet"]').forEach(link => {
        links.push(
            Array.from(link.attributes).map(attr => ({
                name: attr.name,
                value: attr.value
            }))
        );
    });

    // Get the body content (or full content if no body tag)
    const bodyContent = doc.body ? doc.body.innerHTML : doc.documentElement.innerHTML;

    return {
        scripts,
        styles,
        links,
        bodyContent
    };
}

/**
 * Check if a URL is relative (not absolute or protocol-relative)
 * @param {string} url - The URL to check
 * @returns {boolean} True if the URL is relative
 */
function isRelativeUrl(url) {
    if (!url) return false;
    // Absolute URLs start with a protocol (http:, https:, data:, etc.) or are protocol-relative (//)
    // Also exclude URLs starting with / (root-relative)
    return !url.match(/^([a-zA-Z][a-zA-Z0-9+.-]*:|\/\/|\/)/);
}

/**
 * Rewrite a relative URL to use the /localfile/ prefix
 * @param {string} url - The URL to rewrite
 * @returns {string} The rewritten URL, or original if not relative
 */
function rewriteRelativeUrl(url) {
    if (!isRelativeUrl(url)) {
        return url;
    }
    return '/localfile/' + url;
}

/**
 * Rewrite relative URLs in HTML content to use the /localfile/ prefix
 * Handles src, href, and other URL-containing attributes
 * @param {string} html - The HTML content to process
 * @returns {string} The HTML with rewritten URLs
 */
function rewriteRelativeUrlsInHtml(html) {
    // Create a temporary container to parse and modify the HTML
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, 'text/html');

    // Attributes that can contain URLs
    const urlAttributes = ['src', 'href', 'poster', 'data'];

    // Find all elements with URL attributes
    urlAttributes.forEach(attr => {
        doc.querySelectorAll(`[${attr}]`).forEach(el => {
            const value = el.getAttribute(attr);
            if (isRelativeUrl(value)) {
                el.setAttribute(attr, rewriteRelativeUrl(value));
            }
        });
    });

    // Handle srcset attribute (comma-separated list of URLs with optional size descriptors)
    doc.querySelectorAll('[srcset]').forEach(el => {
        const srcset = el.getAttribute('srcset');
        const rewritten = srcset.split(',').map(part => {
            const trimmed = part.trim();
            const [url, ...rest] = trimmed.split(/\s+/);
            if (isRelativeUrl(url)) {
                return [rewriteRelativeUrl(url), ...rest].join(' ');
            }
            return trimmed;
        }).join(', ');
        el.setAttribute('srcset', rewritten);
    });

    // Handle inline styles with url() references
    doc.querySelectorAll('[style]').forEach(el => {
        let style = el.getAttribute('style');
        // Match url(...) patterns, being careful with quotes
        style = style.replace(/url\(\s*(['"]?)([^)'"]+)\1\s*\)/gi, (match, quote, url) => {
            if (isRelativeUrl(url)) {
                return `url(${quote}${rewriteRelativeUrl(url)}${quote})`;
            }
            return match;
        });
        el.setAttribute('style', style);
    });

    return doc.body.innerHTML;
}

/**
 * Render parsed HTML content into a container element.
 * Injects styles into document head and executes scripts.
 * External scripts are loaded asynchronously but in order.
 *
 * @param {Object} parsed - Output from parseHTML()
 * @param {HTMLElement} contentContainer - Element to render body content into
 * @param {Document} targetDocument - Document to inject styles/scripts into (default: document)
 * @param {string} basePath - Optional base path for resolving relative URLs (file system path)
 * @returns {Promise<void>} Resolves when all scripts have been loaded and executed
 */
export async function renderParsedHTML(parsed, contentContainer, targetDocument = document, basePath = '') {
    // Determine if we need to rewrite URLs (when basePath is provided)
    const shouldRewriteUrls = !!basePath;

    // Remove existing user styles
    const existingUserStyles = targetDocument.head.querySelectorAll('style[data-user-content]');
    existingUserStyles.forEach(style => style.remove());

    // Add new styles (rewrite url() references if needed)
    parsed.styles.forEach(styleContent => {
        const newStyle = targetDocument.createElement('style');
        newStyle.setAttribute('data-user-content', 'true');
        if (shouldRewriteUrls) {
            // Rewrite url() references in CSS
            styleContent = styleContent.replace(/url\(\s*(['"]?)([^)'"]+)\1\s*\)/gi, (match, quote, url) => {
                if (isRelativeUrl(url)) {
                    return `url(${quote}${rewriteRelativeUrl(url)}${quote})`;
                }
                return match;
            });
        }
        newStyle.textContent = styleContent;
        targetDocument.head.appendChild(newStyle);
    });

    // Remove existing user links
    const existingUserLinks = targetDocument.head.querySelectorAll('link[data-user-content]');
    existingUserLinks.forEach(link => link.remove());

    // Add new stylesheet links (rewrite href if needed)
    parsed.links.forEach(linkAttrs => {
        const newLink = targetDocument.createElement('link');
        newLink.setAttribute('data-user-content', 'true');
        linkAttrs.forEach(attr => {
            let value = attr.value;
            if (shouldRewriteUrls && attr.name === 'href' && isRelativeUrl(value)) {
                value = rewriteRelativeUrl(value);
            }
            newLink.setAttribute(attr.name, value);
        });
        targetDocument.head.appendChild(newLink);
    });

    // Remove existing user scripts
    const existingUserScripts = targetDocument.body.querySelectorAll('script[data-user-content]');
    existingUserScripts.forEach(script => script.remove());

    // Set the body content (rewrite URLs if needed)
    let bodyContent = parsed.bodyContent;
    if (shouldRewriteUrls) {
        bodyContent = rewriteRelativeUrlsInHtml(bodyContent);
    }
    contentContainer.innerHTML = bodyContent;

    // Execute scripts in order, waiting for external scripts to load
    for (const scriptInfo of parsed.scripts) {
        const newScript = targetDocument.createElement('script');
        newScript.setAttribute('data-user-content', 'true');
        scriptInfo.attributes.forEach(attr => {
            if (attr.name !== 'src') {
                newScript.setAttribute(attr.name, attr.value);
            }
        });

        if (scriptInfo.src) {
            // External script - wait for it to load before continuing
            // Rewrite src if needed
            let src = scriptInfo.src;
            if (shouldRewriteUrls && isRelativeUrl(src)) {
                src = rewriteRelativeUrl(src);
            }
            await new Promise((resolve) => {
                newScript.onload = resolve;
                newScript.onerror = () => {
                    console.error(`Failed to load external script: ${src}`);
                    resolve(); // Continue even on error to not block other scripts
                };
                newScript.src = src;
                targetDocument.body.appendChild(newScript);
            });
        } else {
            // Inline script - executes synchronously when appended
            newScript.textContent = scriptInfo.textContent;
            targetDocument.body.appendChild(newScript);
        }
    }
}

/**
 * Convenience function that parses and renders HTML in one step.
 * Returns a Promise that resolves when all scripts have been loaded and executed.
 *
 * @param {string} html - The HTML string to render
 * @param {HTMLElement} contentContainer - Element to render body content into
 * @param {Document} targetDocument - Document to inject styles/scripts into (default: document)
 * @param {string} basePath - Optional base path for resolving relative URLs (file system path)
 * @returns {Promise<void>} Resolves when rendering is complete
 */
export async function renderHTML(html, contentContainer, targetDocument = document, basePath = '') {
    const parsed = parseHTML(html);
    await renderParsedHTML(parsed, contentContainer, targetDocument, basePath);
}
