/**
 * @fileoverview Main homepage controller managing disclaimer consent
 * and user sessions.
 */

import dom from "../dom.js";
import frame from "../frame.js";

/**
 * Controller for the homepage index view.
 *
 * @returns {function(): void} Teardown function to unbind DOM events.
 */
frame.controllers.main = function main() {
    /**
     * Handles disclaimer agreement and hides the disclaimer banner.
     *
     * @param {Event} [e] - Click event object.
     * @returns {void}
     */
    function agreeDisclaimer(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        dom(".disclaimer-container").addClass("hide");
    }

    /**
     * Closes the disclaimer banner.
     *
     * @param {Event} [e] - Click event object.
     * @returns {void}
     */
    function closeDisclaimer(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        dom(".disclaimer-container").addClass("hide");
    }

    /**
     * Submits a logout request and refreshes the current window.
     *
     * @param {Event} [e] - Click event object.
     * @returns {void}
     */
    function handleLogout(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        fetch("/logout", {
            headers: {
                "X-CSRF-Token": frame.getCSRFToken(),
                "X-Requested-With": "XMLHttpRequest"
            },
            method: "POST"
        }).then(function () {
            globalThis.location.reload();
        }).catch(function (err) {
            console.error("Logout request failed:", err);
        });
    }

    dom("#agree-to-disclaimer").on("click", agreeDisclaimer, false);
    dom(".close-disclaimer").on("click", closeDisclaimer, false);
    dom("[data-action='logout']").on("click", handleLogout, false);

    return function cleanup() {
        dom("#agree-to-disclaimer").off("click");
        dom(".close-disclaimer").off("click");
        dom("[data-action='logout']").off("click");
    };
};
