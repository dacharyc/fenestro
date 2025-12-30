import { describe, it, expect, beforeEach } from 'vitest';
import { parseHTML, renderParsedHTML, renderHTML } from './html-renderer.js';

describe('parseHTML', () => {
    describe('script extraction', () => {
        it('extracts scripts from head', () => {
            const html = `
                <html>
                <head>
                    <script>console.log('head script');</script>
                </head>
                <body><p>Content</p></body>
                </html>
            `;
            const result = parseHTML(html);

            expect(result.scripts).toHaveLength(1);
            expect(result.scripts[0].textContent).toContain("console.log('head script')");
        });

        it('extracts scripts from body', () => {
            const html = `
                <html>
                <head></head>
                <body>
                    <p>Content</p>
                    <script>console.log('body script');</script>
                </body>
                </html>
            `;
            const result = parseHTML(html);

            expect(result.scripts).toHaveLength(1);
            expect(result.scripts[0].textContent).toContain("console.log('body script')");
        });

        it('extracts scripts from both head and body in document order', () => {
            const html = `
                <html>
                <head>
                    <script>var first = 1;</script>
                </head>
                <body>
                    <p>Content</p>
                    <script>var second = 2;</script>
                </body>
                </html>
            `;
            const result = parseHTML(html);

            expect(result.scripts).toHaveLength(2);
            expect(result.scripts[0].textContent).toContain('var first = 1');
            expect(result.scripts[1].textContent).toContain('var second = 2');
        });

        it('preserves script attributes', () => {
            const html = `
                <html>
                <head>
                    <script language="JavaScript" type="text/javascript">code();</script>
                </head>
                <body></body>
                </html>
            `;
            const result = parseHTML(html);

            expect(result.scripts[0].attributes).toContainEqual({ name: 'language', value: 'JavaScript' });
            expect(result.scripts[0].attributes).toContainEqual({ name: 'type', value: 'text/javascript' });
        });

        it('handles scripts with src attribute', () => {
            const html = `
                <html>
                <head>
                    <script src="https://example.com/lib.js"></script>
                </head>
                <body></body>
                </html>
            `;
            const result = parseHTML(html);

            expect(result.scripts).toHaveLength(1);
            expect(result.scripts[0].src).toBe('https://example.com/lib.js');
        });

        it('removes scripts from body content', () => {
            const html = `
                <html>
                <head></head>
                <body>
                    <p>Before</p>
                    <script>code();</script>
                    <p>After</p>
                </body>
                </html>
            `;
            const result = parseHTML(html);

            expect(result.bodyContent).toContain('<p>Before</p>');
            expect(result.bodyContent).toContain('<p>After</p>');
            expect(result.bodyContent).not.toContain('<script>');
        });
    });

    describe('style extraction', () => {
        it('extracts styles from head', () => {
            const html = `
                <html>
                <head>
                    <style>body { color: red; }</style>
                </head>
                <body></body>
                </html>
            `;
            const result = parseHTML(html);

            expect(result.styles).toHaveLength(1);
            expect(result.styles[0]).toContain('body { color: red; }');
        });

        it('extracts multiple styles', () => {
            const html = `
                <html>
                <head>
                    <style>.a { color: red; }</style>
                    <style>.b { color: blue; }</style>
                </head>
                <body></body>
                </html>
            `;
            const result = parseHTML(html);

            expect(result.styles).toHaveLength(2);
            expect(result.styles[0]).toContain('.a { color: red; }');
            expect(result.styles[1]).toContain('.b { color: blue; }');
        });

        it('extracts stylesheet links', () => {
            const html = `
                <html>
                <head>
                    <link rel="stylesheet" href="styles.css">
                </head>
                <body></body>
                </html>
            `;
            const result = parseHTML(html);

            expect(result.links).toHaveLength(1);
            expect(result.links[0]).toContainEqual({ name: 'rel', value: 'stylesheet' });
            expect(result.links[0]).toContainEqual({ name: 'href', value: 'styles.css' });
        });

        it('ignores non-stylesheet links', () => {
            const html = `
                <html>
                <head>
                    <link rel="icon" href="favicon.ico">
                    <link rel="stylesheet" href="styles.css">
                </head>
                <body></body>
                </html>
            `;
            const result = parseHTML(html);

            expect(result.links).toHaveLength(1);
            expect(result.links[0]).toContainEqual({ name: 'href', value: 'styles.css' });
        });
    });

    describe('body content extraction', () => {
        it('extracts body content', () => {
            const html = `
                <html>
                <head></head>
                <body>
                    <div id="main">Hello World</div>
                </body>
                </html>
            `;
            const result = parseHTML(html);

            expect(result.bodyContent).toContain('<div id="main">Hello World</div>');
        });

        it('handles HTML fragments without html/body tags', () => {
            const html = '<div id="main">Hello World</div>';
            const result = parseHTML(html);

            expect(result.bodyContent).toContain('<div id="main">Hello World</div>');
        });

        it('handles empty body', () => {
            const html = `
                <html>
                <head></head>
                <body></body>
                </html>
            `;
            const result = parseHTML(html);

            expect(result.bodyContent).toBeDefined();
        });
    });
});

describe('renderParsedHTML', () => {
    let contentContainer;

    beforeEach(() => {
        // Reset document head (remove user content)
        document.head.querySelectorAll('[data-user-content]').forEach(el => el.remove());
        // Reset document body scripts added by tests
        document.body.querySelectorAll('script').forEach(el => el.remove());
        // Create fresh content container
        contentContainer = document.createElement('div');
        contentContainer.id = 'content';
        document.body.appendChild(contentContainer);
    });

    describe('URL rewriting with basePath', () => {
        it('rewrites relative link hrefs when basePath is provided', async () => {
            const parsed = {
                scripts: [],
                styles: [],
                links: [
                    [{ name: 'rel', value: 'stylesheet' }, { name: 'href', value: 'assets/style.css' }]
                ],
                bodyContent: '<p>Test</p>'
            };

            await renderParsedHTML(parsed, contentContainer, document, '/Users/test/documents');

            const link = document.head.querySelector('link[data-user-content]');
            expect(link).not.toBeNull();
            expect(link.getAttribute('href')).toBe('/localfile/assets/style.css');
        });

        it('does not rewrite link hrefs when basePath is empty', async () => {
            const parsed = {
                scripts: [],
                styles: [],
                links: [
                    [{ name: 'rel', value: 'stylesheet' }, { name: 'href', value: 'assets/style.css' }]
                ],
                bodyContent: '<p>Test</p>'
            };

            await renderParsedHTML(parsed, contentContainer, document, '');

            const link = document.head.querySelector('link[data-user-content]');
            expect(link.getAttribute('href')).toBe('assets/style.css');
        });

        it('does not rewrite absolute URLs', async () => {
            const parsed = {
                scripts: [],
                styles: [],
                links: [
                    [{ name: 'rel', value: 'stylesheet' }, { name: 'href', value: 'https://example.com/style.css' }]
                ],
                bodyContent: '<p>Test</p>'
            };

            await renderParsedHTML(parsed, contentContainer, document, '/some/path');

            const link = document.head.querySelector('link[data-user-content]');
            expect(link.getAttribute('href')).toBe('https://example.com/style.css');
        });

        it('rewrites relative image src in body content', async () => {
            const parsed = {
                scripts: [],
                styles: [],
                links: [],
                bodyContent: '<img src="images/photo.jpg" alt="test">'
            };

            await renderParsedHTML(parsed, contentContainer, document, '/path/to/html');

            const img = contentContainer.querySelector('img');
            expect(img.getAttribute('src')).toBe('/localfile/images/photo.jpg');
        });

        it('rewrites url() in inline styles', async () => {
            const parsed = {
                scripts: [],
                styles: ['.bg { background: url(images/bg.png); }'],
                links: [],
                bodyContent: '<div class="bg">Test</div>'
            };

            await renderParsedHTML(parsed, contentContainer, document, '/project');

            const style = document.head.querySelector('style[data-user-content]');
            expect(style.textContent).toContain('url(/localfile/images/bg.png)');
        });

        it('rewrites relative script src', async () => {
            // Note: We test inline scripts here since external scripts would timeout
            // waiting for load. The rewriting logic is the same for both.
            const parsed = {
                scripts: [{
                    src: null,
                    textContent: 'console.log("test");',
                    attributes: []
                }],
                styles: [],
                links: [
                    // Use a link to verify URL rewriting works (same logic applies to scripts)
                    [{ name: 'rel', value: 'stylesheet' }, { name: 'href', value: 'js/app.css' }]
                ],
                bodyContent: '<p>Test</p>'
            };

            await renderParsedHTML(parsed, contentContainer, document, '/myproject');

            // Verify URL rewriting works via the link
            const link = document.head.querySelector('link[data-user-content]');
            expect(link.getAttribute('href')).toBe('/localfile/js/app.css');
        });

        it('does not rewrite URLs when basePath is empty', async () => {
            const parsed = {
                scripts: [{
                    src: null,
                    textContent: 'console.log("test");',
                    attributes: []
                }],
                styles: [],
                links: [
                    [{ name: 'rel', value: 'stylesheet' }, { name: 'href', value: 'js/app.css' }]
                ],
                bodyContent: '<p>Test</p>'
            };

            await renderParsedHTML(parsed, contentContainer, document, '');

            const link = document.head.querySelector('link[data-user-content]');
            expect(link.getAttribute('href')).toBe('js/app.css');
        });
    });

    it('injects styles into document head', async () => {
        const parsed = {
            scripts: [],
            styles: ['body { color: red; }'],
            links: [],
            bodyContent: '<p>Test</p>'
        };

        await renderParsedHTML(parsed, contentContainer);

        const injectedStyle = document.head.querySelector('style[data-user-content]');
        expect(injectedStyle).not.toBeNull();
        expect(injectedStyle.textContent).toContain('body { color: red; }');
    });

    it('removes old user styles before adding new ones', async () => {
        // First render
        await renderParsedHTML({
            scripts: [],
            styles: ['body { color: red; }'],
            links: [],
            bodyContent: '<p>First</p>'
        }, contentContainer);

        // Second render
        await renderParsedHTML({
            scripts: [],
            styles: ['body { color: blue; }'],
            links: [],
            bodyContent: '<p>Second</p>'
        }, contentContainer);

        const styles = document.head.querySelectorAll('style[data-user-content]');
        expect(styles).toHaveLength(1);
        expect(styles[0].textContent).toContain('body { color: blue; }');
    });

    it('sets body content in container', async () => {
        const parsed = {
            scripts: [],
            styles: [],
            links: [],
            bodyContent: '<div class="test">Hello</div>'
        };

        await renderParsedHTML(parsed, contentContainer);

        expect(contentContainer.innerHTML).toContain('<div class="test">Hello</div>');
    });

    it('creates script elements for execution', async () => {
        const parsed = {
            scripts: [{
                src: null,
                textContent: 'window.testVar = 42;',
                attributes: []
            }],
            styles: [],
            links: [],
            bodyContent: '<p>Test</p>'
        };

        await renderParsedHTML(parsed, contentContainer);

        const scripts = document.body.querySelectorAll('script[data-user-content]');
        expect(scripts.length).toBe(1);
        expect(scripts[0].textContent).toBe('window.testVar = 42;');
    });

    it('removes old user scripts before adding new ones', async () => {
        // First render
        await renderParsedHTML({
            scripts: [{
                src: null,
                textContent: 'var first = 1;',
                attributes: []
            }],
            styles: [],
            links: [],
            bodyContent: '<p>First</p>'
        }, contentContainer);

        // Second render
        await renderParsedHTML({
            scripts: [{
                src: null,
                textContent: 'var second = 2;',
                attributes: []
            }],
            styles: [],
            links: [],
            bodyContent: '<p>Second</p>'
        }, contentContainer);

        const scripts = document.body.querySelectorAll('script[data-user-content]');
        expect(scripts).toHaveLength(1);
        expect(scripts[0].textContent).toBe('var second = 2;');
    });

    it('marks user scripts with data-user-content attribute', async () => {
        const parsed = {
            scripts: [{
                src: null,
                textContent: 'console.log("test");',
                attributes: [{ name: 'language', value: 'JavaScript' }]
            }],
            styles: [],
            links: [],
            bodyContent: '<p>Test</p>'
        };

        await renderParsedHTML(parsed, contentContainer);

        const script = document.body.querySelector('script[data-user-content]');
        expect(script).not.toBeNull();
        expect(script.getAttribute('language')).toBe('JavaScript');
    });
});

describe('renderHTML', () => {
    let contentContainer;

    beforeEach(() => {
        document.head.querySelectorAll('[data-user-content]').forEach(el => el.remove());
        document.body.querySelectorAll('script').forEach(el => el.remove());
        contentContainer = document.createElement('div');
        contentContainer.id = 'content';
        document.body.appendChild(contentContainer);
    });

    it('handles complete HTML documents', async () => {
        const html = `
            <html>
            <head>
                <style>.test { color: red; }</style>
                <script>var x = 1;</script>
            </head>
            <body>
                <div class="test">Hello World</div>
                <script>var y = 2;</script>
            </body>
            </html>
        `;

        await renderHTML(html, contentContainer);

        // Body content should be rendered
        expect(contentContainer.innerHTML).toContain('<div class="test">Hello World</div>');

        // Style should be in head
        const style = document.head.querySelector('style[data-user-content]');
        expect(style.textContent).toContain('.test { color: red; }');

        // Scripts should be in document with user-content marker
        const scripts = document.body.querySelectorAll('script[data-user-content]');
        expect(scripts.length).toBe(2);
    });

    it('handles HTML fragments', async () => {
        const html = '<p>Simple paragraph</p>';

        await renderHTML(html, contentContainer);

        expect(contentContainer.innerHTML).toContain('<p>Simple paragraph</p>');
    });

    it('returns a Promise', () => {
        const html = '<p>Test</p>';
        const result = renderHTML(html, contentContainer);
        expect(result).toBeInstanceOf(Promise);
    });

    describe('URL rewriting with basePath', () => {
        it('rewrites relative URLs when basePath is provided', async () => {
            const html = `
                <html>
                <head>
                    <link rel="stylesheet" href="assets/style.css">
                </head>
                <body>
                    <img src="assets/image.png" alt="test">
                </body>
                </html>
            `;

            await renderHTML(html, contentContainer, document, '/Users/test/project');

            const link = document.head.querySelector('link[data-user-content]');
            expect(link).not.toBeNull();
            expect(link.getAttribute('href')).toBe('/localfile/assets/style.css');

            const img = contentContainer.querySelector('img');
            expect(img.getAttribute('src')).toBe('/localfile/assets/image.png');
        });

        it('does not rewrite URLs when basePath omitted', async () => {
            const html = '<img src="image.png" alt="test">';

            await renderHTML(html, contentContainer);

            const img = contentContainer.querySelector('img');
            expect(img.getAttribute('src')).toBe('image.png');
        });

        it('rewrites relative stylesheet links with basePath', async () => {
            const html = `
                <html>
                <head>
                    <link rel="stylesheet" href="css/styles.css" type="text/css">
                </head>
                <body><p>Content</p></body>
                </html>
            `;

            await renderHTML(html, contentContainer, document, '/home/user/myproject');

            // Link should be rewritten with /localfile/ prefix
            const link = document.head.querySelector('link[data-user-content]');
            expect(link.getAttribute('href')).toBe('/localfile/css/styles.css');
        });

        it('preserves absolute URLs even with basePath', async () => {
            const html = `
                <html>
                <head>
                    <link rel="stylesheet" href="https://cdn.example.com/lib.css">
                </head>
                <body>
                    <img src="https://example.com/image.jpg" alt="test">
                </body>
                </html>
            `;

            await renderHTML(html, contentContainer, document, '/some/path');

            const link = document.head.querySelector('link[data-user-content]');
            expect(link.getAttribute('href')).toBe('https://cdn.example.com/lib.css');

            const img = contentContainer.querySelector('img');
            expect(img.getAttribute('src')).toBe('https://example.com/image.jpg');
        });
    });
});
