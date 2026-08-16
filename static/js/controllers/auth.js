/**
 * @fileoverview Authentication controller handling user login and
 * registration demo flows.
 */

import dom from "../dom.js";
import frame from "../frame.js";

/**
 * Controller for the login and registration view.
 *
 * @returns {function(): void} Teardown function to unbind DOM events.
 */
frame.controllers.auth = function auth() {
    let isRegisterMode = false;

    /**
     * Displays a status or error message in the auth message container.
     *
     * @param {string} text - Message text to display.
     * @param {boolean} [isError=false] - True for error formatting.
     * @returns {void}
     */
    function showMessage(text, isError) {
        const msgEl = dom("#auth-message");
        msgEl.removeClass("hide");
        msgEl.removeClass("red-text");
        msgEl.removeClass("green-text");
        if (isError) {
            msgEl.addClass("red-text");
        } else {
            msgEl.addClass("green-text");
        }
        msgEl.text(text);
    }

    /**
     * Submits login credentials to the server session endpoint.
     *
     * @param {Event} [e] - Form submission event.
     * @returns {void}
     */
    function handleLogin(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const aliasInput = dom("#auth-alias").get(0);
        const passInput = dom("#auth-password").get(0);
        const alias = (
            (aliasInput && aliasInput.value)
            ? aliasInput.value.trim()
            : ""
        );
        const password = (
            (passInput && passInput.value)
            ? passInput.value
            : ""
        );
        if (passInput) {
            passInput.value = "";
        }
        if (!alias || !password) {
            showMessage("Alias and password are required.", true);
            return;
        }
        const fd = new FormData();
        fd.append("alias", alias);
        fd.append("password", password);

        fetch("/login", {
            body: fd,
            headers: {
                "X-CSRF-Token": frame.getCSRFToken(),
                "X-Requested-With": "XMLHttpRequest"
            },
            method: "POST"
        }).then(function (res) {
            if (res.ok) {
                showMessage("Login successful! Redirecting...", false);
                setTimeout(function () {
                    frame.loadRoute("/");
                }, 500);
            } else {
                showMessage(
                    "Invalid credentials or login error (" +
                    res.status +
                    ").",
                    true
                );
            }
        }).catch(function (err) {
            showMessage(
                "Network error during login: " + err.message,
                true
            );
        });
    }

    /**
     * Submits registration credentials to create a new user account.
     *
     * @param {Event} [e] - Form submission event.
     * @returns {void}
     */
    function handleRegister(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const aliasInput = dom("#auth-alias").get(0);
        const passInput = dom("#auth-password").get(0);
        const repeatInput = dom("#auth-password-repeat").get(0);
        const alias = (
            (aliasInput && aliasInput.value)
            ? aliasInput.value.trim()
            : ""
        );
        const password = (
            (passInput && passInput.value)
            ? passInput.value
            : ""
        );
        const repeat = (
            (repeatInput && repeatInput.value)
            ? repeatInput.value
            : ""
        );
        if (passInput) {
            passInput.value = "";
        }
        if (repeatInput) {
            repeatInput.value = "";
        }
        if (!alias || !password) {
            showMessage("Alias and password are required.", true);
            return;
        }
        if (password.length < 8 || password.length > 72) {
            showMessage(
                "Password must be between 8 and 72 characters.",
                true
            );
            return;
        }
        if (password !== repeat) {
            showMessage("Passwords do not match.", true);
            return;
        }
        const fd = new FormData();
        fd.append("alias", alias);
        fd.append("password", password);

        fetch("/register", {
            body: fd,
            headers: {
                "X-CSRF-Token": frame.getCSRFToken(),
                "X-Requested-With": "XMLHttpRequest"
            },
            method: "POST"
        }).then(function (res) {
            if (res.ok) {
                showMessage(
                    "Registration successful! Redirecting...",
                    false
                );
                setTimeout(function () {
                    frame.loadRoute("/");
                }, 500);
            } else if (res.status === 409) {
                showMessage("Alias is already taken.", true);
            } else {
                showMessage(
                    "Registration failed (" + res.status + ").",
                    true
                );
            }
        }).catch(function (err) {
            showMessage(
                "Network error during registration: " + err.message,
                true
            );
        });
    }

    /**
     * Renders the authentication modal in login mode.
     *
     * @returns {void}
     */
    function showLoginForm() {
        isRegisterMode = false;
        dom(".auth-title").text("Login");
        dom(".form-toggler").text("Need an account? Register here");
        dom("#auth-repeat-row").addClass("hide");
        dom("#auth-submit-btn").text("Login");
        dom("#auth-submit-btn").off("click").on("click", handleLogin, false);
    }

    /**
     * Renders the authentication modal in registration mode.
     *
     * @returns {void}
     */
    function showRegisterForm() {
        isRegisterMode = true;
        dom(".auth-title").text("Register");
        dom(".form-toggler").text("Already have an account? Login here");
        dom("#auth-repeat-row").removeClass("hide");
        dom("#auth-submit-btn").text("Register");
        dom("#auth-submit-btn").off("click").on("click", handleRegister, false);
    }

    /**
     * Toggles between login and registration form modes.
     *
     * @param {Event} [e] - Triggering click event.
     * @returns {void}
     */
    function toggleMode(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        if (isRegisterMode) {
            showLoginForm();
        } else {
            showRegisterForm();
        }
    }

    showLoginForm();
    dom(".form-toggler").on("click", toggleMode, false);

    return function cleanup() {
        dom("#auth-submit-btn").off("click");
        dom(".form-toggler").off("click");
    };
};
