/**
 * @fileoverview Utility functions for rendering and escaping post content.
 */

/**
 * Escapes unsafe characters for HTML display.
 *
 * @param {string} str - Raw input text.
 * @returns {string} HTML-escaped text.
 */
function escapeHtml(str) {
    if (typeof str !== "string") {
        return "";
    }
    return str
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

export default Object.freeze({
    escapeHtml
});
