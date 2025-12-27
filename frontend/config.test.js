import { describe, it, expect, beforeEach } from 'vitest';
import { applyFontSize, injectChromeCSS } from './config.js';

describe('applyFontSize', () => {
    let contentElement;

    beforeEach(() => {
        contentElement = document.createElement('div');
        contentElement.id = 'content';
        document.body.appendChild(contentElement);
    });

    it('applies font size when config has valid font_size', () => {
        const config = { font_size: 18 };

        const result = applyFontSize(config, contentElement);

        expect(result).toBe(true);
        expect(contentElement.style.fontSize).toBe('18px');
    });

    it('applies font size with snake_case field name (as sent by backend)', () => {
        // This is the format the backend sends after JSON serialization
        const config = { font_size: 24, chrome_css: '/path/to/file.css' };

        const result = applyFontSize(config, contentElement);

        expect(result).toBe(true);
        expect(contentElement.style.fontSize).toBe('24px');
    });

    it('does not apply font size when font_size is 0', () => {
        const config = { font_size: 0 };

        const result = applyFontSize(config, contentElement);

        expect(result).toBe(false);
        expect(contentElement.style.fontSize).toBe('');
    });

    it('does not apply font size when font_size is negative', () => {
        const config = { font_size: -10 };

        const result = applyFontSize(config, contentElement);

        expect(result).toBe(false);
        expect(contentElement.style.fontSize).toBe('');
    });

    it('does not apply font size when font_size is missing', () => {
        const config = {};

        const result = applyFontSize(config, contentElement);

        expect(result).toBe(false);
        expect(contentElement.style.fontSize).toBe('');
    });

    it('does not apply font size when config is null', () => {
        const result = applyFontSize(null, contentElement);

        expect(result).toBe(false);
        expect(contentElement.style.fontSize).toBe('');
    });

    it('does not apply font size when config is undefined', () => {
        const result = applyFontSize(undefined, contentElement);

        expect(result).toBe(false);
        expect(contentElement.style.fontSize).toBe('');
    });

    it('does NOT work with PascalCase FontSize (wrong format)', () => {
        // If backend sends PascalCase (missing json tags), it should NOT work
        const config = { FontSize: 18 };

        const result = applyFontSize(config, contentElement);

        expect(result).toBe(false);
        expect(contentElement.style.fontSize).toBe('');
    });
});

describe('injectChromeCSS', () => {
    beforeEach(() => {
        // Clean up any existing chrome CSS
        const existing = document.getElementById('fenestro-chrome-css');
        if (existing) {
            existing.remove();
        }
    });

    it('injects CSS into document head', () => {
        const css = 'body { background: red; }';

        const style = injectChromeCSS(css);

        expect(style).not.toBeNull();
        expect(style.id).toBe('fenestro-chrome-css');
        expect(style.textContent).toBe(css);
        expect(document.head.contains(style)).toBe(true);
    });

    it('returns null when CSS is empty string', () => {
        const style = injectChromeCSS('');

        expect(style).toBeNull();
        expect(document.getElementById('fenestro-chrome-css')).toBeNull();
    });

    it('returns null when CSS is null', () => {
        const style = injectChromeCSS(null);

        expect(style).toBeNull();
    });

    it('returns null when CSS is undefined', () => {
        const style = injectChromeCSS(undefined);

        expect(style).toBeNull();
    });

    it('replaces existing chrome CSS', () => {
        const oldCSS = 'body { color: blue; }';
        const newCSS = 'body { color: green; }';

        injectChromeCSS(oldCSS);
        const newStyle = injectChromeCSS(newCSS);

        const styles = document.querySelectorAll('#fenestro-chrome-css');
        expect(styles.length).toBe(1);
        expect(newStyle.textContent).toBe(newCSS);
    });

    it('injects into custom target document', () => {
        // Create a mock document-like object
        const mockHead = document.createElement('head');
        const mockDoc = {
            getElementById: (id) => mockHead.querySelector(`#${id}`),
            createElement: (tag) => document.createElement(tag),
            head: mockHead
        };

        const css = '.test { color: red; }';
        const style = injectChromeCSS(css, mockDoc);

        expect(style).not.toBeNull();
        expect(mockHead.contains(style)).toBe(true);
    });
});

describe('config integration', () => {
    it('handles complete config object with both fields', () => {
        // Simulate the exact format sent by the Go backend
        const config = {
            font_size: 20,
            chrome_css: '/Users/test/.config/fenestro/chrome.css'
        };

        const contentElement = document.createElement('div');
        const fontResult = applyFontSize(config, contentElement);

        expect(fontResult).toBe(true);
        expect(contentElement.style.fontSize).toBe('20px');
        // chrome_css value is a path, actual CSS content comes from GetChromeCSS()
    });

    it('handles default config (zero values)', () => {
        // Default config from Go: FontSize=0, ChromeCSS=""
        const config = {
            font_size: 0,
            chrome_css: ''
        };

        const contentElement = document.createElement('div');
        const fontResult = applyFontSize(config, contentElement);
        const cssResult = injectChromeCSS(config.chrome_css);

        expect(fontResult).toBe(false);
        expect(cssResult).toBeNull();
    });
});
