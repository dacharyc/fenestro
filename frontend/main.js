// Fenestro - Find in page and sidebar functionality

import { renderHTML as renderHTMLContent } from './html-renderer.js';
import { applyFontSize, injectChromeCSS } from './config.js';

(function() {
    'use strict';

    // State
    let matches = [];
    let currentMatchIndex = -1;
    let lastSearchTerm = '';
    let files = [];
    let selectedIndex = 0;
    let zoomLevel = 1.0;
    const ZOOM_STEP = 0.1;
    const ZOOM_MIN = 0.25;
    const ZOOM_MAX = 5.0;

    // DOM elements
    const findBar = document.getElementById('find-bar');
    const findInput = document.getElementById('find-input');
    const findCount = document.getElementById('find-count');
    const findPrev = document.getElementById('find-prev');
    const findNext = document.getElementById('find-next');
    const findClose = document.getElementById('find-close');
    const content = document.getElementById('content');
    const sidebar = document.getElementById('sidebar');
    const fileList = document.getElementById('file-list');

    // Render HTML content using the html-renderer module
    async function renderHTML(html, basePath = '') {
        await renderHTMLContent(html, content, document, basePath);
    }

    // Load HTML content from backend
    async function loadContent() {
        try {
            const html = await window.go.main.App.GetHTMLContent();
            const basePath = await window.go.main.App.GetCurrentBasePath();
            await renderHTML(html, basePath);
        } catch (err) {
            content.innerHTML = '<p style="color: red;">Error loading content: ' + err + '</p>';
        }
    }

    // Load files and update sidebar
    async function loadFiles() {
        try {
            files = await window.go.main.App.GetFiles();
            selectedIndex = await window.go.main.App.GetCurrentIndex();
            updateSidebar();
        } catch (err) {
            console.error('Error loading files:', err);
        }
    }

    // Update the sidebar display
    function updateSidebar() {
        // Show/hide sidebar based on file count
        if (files.length > 1) {
            sidebar.classList.remove('hidden');
        } else {
            sidebar.classList.add('hidden');
        }

        // Render file list
        fileList.innerHTML = '';
        files.forEach((file, index) => {
            const item = document.createElement('div');
            item.className = 'file-item' + (index === selectedIndex ? ' selected' : '');
            item.textContent = file.name;
            item.title = file.path || file.name;
            item.addEventListener('click', () => selectFile(index));
            fileList.appendChild(item);
        });
    }

    // Select a file by index
    async function selectFile(index) {
        try {
            const html = await window.go.main.App.SelectFile(index);
            const basePath = await window.go.main.App.GetCurrentBasePath();
            await renderHTML(html, basePath);
            selectedIndex = index;
            updateSidebar();
            // Clear find highlights when switching files
            clearHighlights();
            findInput.value = '';
            findCount.textContent = '';
            matches = [];
            currentMatchIndex = -1;
            lastSearchTerm = '';
        } catch (err) {
            console.error('Error selecting file:', err);
        }
    }

    // Handle file-added event from backend
    function onFileAdded(data) {
        files = data.files;
        // Don't change selection, just update sidebar
        updateSidebar();
    }

    // Handle content-replaced event from backend
    function onContentReplaced(data) {
        files = data.files;
        selectedIndex = data.currentIndex;
        updateSidebar();
        loadContent();
    }

    // Show find bar
    function showFindBar() {
        findBar.classList.remove('hidden');
        findInput.focus();
        findInput.select();
    }

    // Hide find bar
    function hideFindBar() {
        findBar.classList.add('hidden');
        clearHighlights();
        findInput.value = '';
        findCount.textContent = '';
        matches = [];
        currentMatchIndex = -1;
        lastSearchTerm = '';
    }

    // Clear all highlights
    function clearHighlights() {
        const highlights = content.querySelectorAll('.find-highlight');
        highlights.forEach(el => {
            const parent = el.parentNode;
            parent.replaceChild(document.createTextNode(el.textContent), el);
            parent.normalize();
        });
    }

    // Highlight all matches
    function highlightMatches(searchTerm) {
        if (!searchTerm) {
            clearHighlights();
            findCount.textContent = '';
            matches = [];
            currentMatchIndex = -1;
            return;
        }

        // Clear previous highlights
        clearHighlights();
        matches = [];
        currentMatchIndex = -1;

        const searchLower = searchTerm.toLowerCase();
        const walker = document.createTreeWalker(
            content,
            NodeFilter.SHOW_TEXT,
            null,
            false
        );

        const nodesToProcess = [];
        let node;
        while (node = walker.nextNode()) {
            if (node.textContent.toLowerCase().includes(searchLower)) {
                nodesToProcess.push(node);
            }
        }

        nodesToProcess.forEach(textNode => {
            const text = textNode.textContent;
            const textLower = text.toLowerCase();
            const parent = textNode.parentNode;

            // Skip if already in a highlight span
            if (parent.classList && parent.classList.contains('find-highlight')) {
                return;
            }

            const fragment = document.createDocumentFragment();
            let lastIndex = 0;
            let matchIndex;

            while ((matchIndex = textLower.indexOf(searchLower, lastIndex)) !== -1) {
                // Add text before match
                if (matchIndex > lastIndex) {
                    fragment.appendChild(document.createTextNode(text.slice(lastIndex, matchIndex)));
                }

                // Add highlighted match
                const span = document.createElement('span');
                span.className = 'find-highlight';
                span.textContent = text.slice(matchIndex, matchIndex + searchTerm.length);
                fragment.appendChild(span);
                matches.push(span);

                lastIndex = matchIndex + searchTerm.length;
            }

            // Add remaining text
            if (lastIndex < text.length) {
                fragment.appendChild(document.createTextNode(text.slice(lastIndex)));
            }

            parent.replaceChild(fragment, textNode);
        });

        updateMatchCount();

        // Jump to first match
        if (matches.length > 0) {
            currentMatchIndex = 0;
            highlightCurrentMatch();
        }
    }

    // Update the match count display
    function updateMatchCount() {
        if (matches.length === 0) {
            findCount.textContent = findInput.value ? 'No matches' : '';
        } else {
            findCount.textContent = `${currentMatchIndex + 1} of ${matches.length}`;
        }
    }

    // Highlight current match and scroll to it
    function highlightCurrentMatch() {
        // Remove current class from all
        matches.forEach(m => m.classList.remove('current'));

        if (currentMatchIndex >= 0 && currentMatchIndex < matches.length) {
            const current = matches[currentMatchIndex];
            current.classList.add('current');
            current.scrollIntoView({ behavior: 'smooth', block: 'center' });
            updateMatchCount();
        }
    }

    // Go to next match
    function nextMatch() {
        if (matches.length === 0) return;
        currentMatchIndex = (currentMatchIndex + 1) % matches.length;
        highlightCurrentMatch();
    }

    // Go to previous match
    function prevMatch() {
        if (matches.length === 0) return;
        currentMatchIndex = (currentMatchIndex - 1 + matches.length) % matches.length;
        highlightCurrentMatch();
    }

    // Debounce function
    function debounce(func, wait) {
        let timeout;
        return function(...args) {
            clearTimeout(timeout);
            timeout = setTimeout(() => func.apply(this, args), wait);
        };
    }

    // Zoom functions
    function applyZoom() {
        content.style.zoom = zoomLevel;
    }

    function zoomIn() {
        if (zoomLevel < ZOOM_MAX) {
            zoomLevel = Math.min(ZOOM_MAX, Math.round((zoomLevel + ZOOM_STEP) * 10) / 10);
            applyZoom();
        }
    }

    function zoomOut() {
        if (zoomLevel > ZOOM_MIN) {
            zoomLevel = Math.max(ZOOM_MIN, Math.round((zoomLevel - ZOOM_STEP) * 10) / 10);
            applyZoom();
        }
    }

    function resetZoom() {
        zoomLevel = 1.0;
        applyZoom();
    }

    // Debounced search
    const debouncedSearch = debounce((term) => {
        if (term !== lastSearchTerm) {
            lastSearchTerm = term;
            highlightMatches(term);
        }
    }, 150);

    // Event listeners
    findInput.addEventListener('input', (e) => {
        debouncedSearch(e.target.value);
    });

    findInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            if (e.shiftKey) {
                prevMatch();
            } else {
                nextMatch();
            }
        } else if (e.key === 'Escape') {
            hideFindBar();
        }
    });

    findNext.addEventListener('click', nextMatch);
    findPrev.addEventListener('click', prevMatch);
    findClose.addEventListener('click', hideFindBar);

    // Global keyboard shortcuts
    document.addEventListener('keydown', (e) => {
        if ((e.metaKey || e.ctrlKey) && e.key === 'f') {
            e.preventDefault();
            showFindBar();
        } else if (e.key === 'Escape' && !findBar.classList.contains('hidden')) {
            hideFindBar();
        } else if ((e.metaKey || e.ctrlKey) && (e.key === '=' || e.key === '+')) {
            // Cmd+= or Cmd++ for zoom in
            e.preventDefault();
            zoomIn();
        } else if ((e.metaKey || e.ctrlKey) && e.key === '-') {
            // Cmd+- for zoom out
            e.preventDefault();
            zoomOut();
        } else if ((e.metaKey || e.ctrlKey) && e.key === '0') {
            // Cmd+0 for reset zoom
            e.preventDefault();
            resetZoom();
        }
    });

    // Load and apply configuration
    async function loadConfig() {
        try {
            const config = await window.go.main.App.GetConfig();
            applyFontSize(config, content);

            // Load and inject custom chrome CSS
            const chromeCSS = await window.go.main.App.GetChromeCSS();
            injectChromeCSS(chromeCSS);
        } catch (err) {
            console.error('Error loading config:', err);
        }
    }

    // Window geometry saving
    // Debounced save to avoid excessive disk writes
    const saveWindowGeometry = debounce(async () => {
        try {
            await window.go.main.App.SaveWindowGeometry();
        } catch (err) {
            // Silently ignore - not critical
        }
    }, 500);

    // Listen for resize events (handles both resize and maximize/restore)
    window.addEventListener('resize', saveWindowGeometry);

    // Poll for position changes since there's no native window move event
    // We check every 2 seconds and save if changed
    let geometryCheckInterval;
    function startGeometryTracking() {
        // Save initial geometry after a short delay to let window settle
        setTimeout(saveWindowGeometry, 1000);

        // Check periodically for position changes
        geometryCheckInterval = setInterval(saveWindowGeometry, 2000);
    }

    // Save geometry when window loses focus (user likely done moving/resizing)
    window.addEventListener('blur', saveWindowGeometry);

    // Initialize
    document.addEventListener('DOMContentLoaded', async () => {
        await loadConfig();
        await loadContent();
        await loadFiles();
        startGeometryTracking();
    });

    // Listen for backend events
    if (window.runtime) {
        window.runtime.EventsOn('file-added', onFileAdded);
        window.runtime.EventsOn('content-replaced', onContentReplaced);
    }
})();
