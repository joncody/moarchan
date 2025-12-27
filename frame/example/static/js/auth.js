"use strict";

import dom from "./dom.js";
import frame from "./frame.js";

frame.controllers.auth = function (global) {
    "use strict";

    // ================ Event Handlers ================

    function handleLogin(e) {
        e.preventDefault();
        const alias = dom(".form-input[name='alias']").get(0).value.trim();
        const password = dom(".form-input[name='password']").get(0).value;
        // Clear sensitive fields immediately
        dom(".form-input[name='password']").get(0).value = "";
        if (!alias || !password) {
            // Optional: show validation message
            return;
        }
        const fd = new FormData();
        fd.append("alias", alias);
        fd.append("password", password);
        fetch("/login", {
            method: "POST",
            body: fd,
            headers: { "X-Requested-With": "XMLHttpRequest" }
        })
        .then(res => {
            if (res.ok) {
                global.location.reload();
            } else {
                // Optional: display error (e.g., "Invalid credentials")
                console.warn("Login failed:", res.status);
            }
        })
        .catch(err => {
            console.error("Login request failed:", err);
        });
    }

    function handleRegister(e) {
        e.preventDefault();
        const alias = dom(".form-input[name='alias']").get(0).value.trim();
        const password = dom(".form-input[name='password']").get(0).value;
        const passwordRepeat = dom(".form-input[name='password-repeat']").get(0).value;
        // Clear password fields immediately
        dom(".form-input[name='password']").get(0).value = "";
        dom(".form-input[name='password-repeat']").get(0).value = "";
        if (!alias || !password || password !== passwordRepeat) {
            // Optional: show "Passwords don't match" etc.
            return;
        }
        const fd = new FormData();
        fd.append("alias", alias);
        fd.append("password", password);

        fetch("/register", {
            method: "POST",
            body: fd,
            headers: { "X-Requested-With": "XMLHttpRequest" }
        })
        .then(res => {
            if (res.ok) {
                global.location.reload();
            } else if (res.status === 409) {
                // Alias already taken
                console.warn("Alias already in use");
            } else {
                console.warn("Registration failed:", res.status);
            }
        })
        .catch(err => {
            console.error("Registration request failed:", err);
        });
    }

    function handleLogout(e) {
        e.preventDefault();
        fetch("/logout", {
            method: "POST",
            headers: { "X-Requested-With": "XMLHttpRequest" }
        })
        .then(res => {
            if (res.ok) {
                global.location.reload();
            }
        })
        .catch(err => {
            console.error("Logout failed:", err);
        });
    }

    // ================ Form State Management ================

    function showLoginForm() {
        dom(".form-toggler").html("Register");
        dom(".form-input[name='password-repeat']").addClass("collapsed");
        dom("button[name='enter']").off("click").on("click", handleLogin);
    }

    function showRegisterForm() {
        dom(".form-toggler").html("Login");
        dom(".form-input[name='password-repeat']").removeClass("collapsed");
        dom("button[name='enter']").off("click").on("click", handleRegister);
    }

    // ================ Setup ================

    // Initial state: login form
    showLoginForm();

    // Toggle between login/register
    dom(".form-toggler").on("click", function () {
        const isRegisterMode = dom(".form-input[name='password-repeat']").hasClass("collapsed");
        if (isRegisterMode) {
            showRegisterForm();
        } else {
            showLoginForm();
        }
    });

    // Attach logout handler
    dom("button[name='leave']").on("click", handleLogout, false);
};
