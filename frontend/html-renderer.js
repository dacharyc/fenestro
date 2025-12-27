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
 * Render parsed HTML content into a container element.
 * Injects styles into document head and executes scripts.
 * External scripts are loaded asynchronously but in order.
 *
 * @param {Object} parsed - Output from parseHTML()
 * @param {HTMLElement} contentContainer - Element to render body content into
 * @param {Document} targetDocument - Document to inject styles/scripts into (default: document)
 * @returns {Promise<void>} Resolves when all scripts have been loaded and executed
 */
export async function renderParsedHTML(parsed, contentContainer, targetDocument = document) {
    // Remove existing user styles
    const existingUserStyles = targetDocument.head.querySelectorAll('style[data-user-content]');
    existingUserStyles.forEach(style => style.remove());

    // Add new styles
    parsed.styles.forEach(styleContent => {
        const newStyle = targetDocument.createElement('style');
        newStyle.setAttribute('data-user-content', 'true');
        newStyle.textContent = styleContent;
        targetDocument.head.appendChild(newStyle);
    });

    // Remove existing user links
    const existingUserLinks = targetDocument.head.querySelectorAll('link[data-user-content]');
    existingUserLinks.forEach(link => link.remove());

    // Add new stylesheet links
    parsed.links.forEach(linkAttrs => {
        const newLink = targetDocument.createElement('link');
        newLink.setAttribute('data-user-content', 'true');
        linkAttrs.forEach(attr => {
            newLink.setAttribute(attr.name, attr.value);
        });
        targetDocument.head.appendChild(newLink);
    });

    // Remove existing user scripts
    const existingUserScripts = targetDocument.body.querySelectorAll('script[data-user-content]');
    existingUserScripts.forEach(script => script.remove());

    // Set the body content
    contentContainer.innerHTML = parsed.bodyContent;

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
            await new Promise((resolve) => {
                newScript.onload = resolve;
                newScript.onerror = () => {
                    console.error(`Failed to load external script: ${scriptInfo.src}`);
                    resolve(); // Continue even on error to not block other scripts
                };
                newScript.src = scriptInfo.src;
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
 * @returns {Promise<void>} Resolves when rendering is complete
 */
export async function renderHTML(html, contentContainer, targetDocument = document) {
    const parsed = parseHTML(html);
    await renderParsedHTML(parsed, contentContainer, targetDocument);
}
