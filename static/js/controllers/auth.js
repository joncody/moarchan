import dom from "../dom.js";
import frame from "../frame.js";

frame.controllers.auth = function auth(global) {
    let isRegisterMode = false;

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

    function handleLogin(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const aliasInput = dom("#auth-alias").get(0);
        const passInput = dom("#auth-password").get(0);

        const alias = (aliasInput && aliasInput.value ? aliasInput.value.trim() : "");
        const password = (passInput && passInput.value ? passInput.value : "");

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
            method: "POST",
            body: fd,
            headers: {
                "X-Requested-With": "XMLHttpRequest"
            }
        }).then(function (res) {
            if (res.ok) {
                showMessage("Login successful! Redirecting...", false);
                setTimeout(function () {
                    frame.loadRoute("/");
                }, 500);
            } else {
                showMessage("Invalid credentials or login error (" + res.status + ").", true);
            }
        }).catch(function (err) {
            showMessage("Network error during login: " + err.message, true);
        });
    }

    function handleRegister(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const aliasInput = dom("#auth-alias").get(0);
        const passInput = dom("#auth-password").get(0);
        const repeatInput = dom("#auth-password-repeat").get(0);

        const alias = (aliasInput && aliasInput.value ? aliasInput.value.trim() : "");
        const password = (passInput && passInput.value ? passInput.value : "");
        const repeat = (repeatInput && repeatInput.value ? repeatInput.value : "");

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
            showMessage("Password must be between 8 and 72 characters.", true);
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
            method: "POST",
            body: fd,
            headers: {
                "X-Requested-With": "XMLHttpRequest"
            }
        }).then(function (res) {
            if (res.ok) {
                showMessage("Registration successful! Redirecting...", false);
                setTimeout(function () {
                    frame.loadRoute("/");
                }, 500);
            } else if (res.status === 409) {
                showMessage("Alias is already taken.", true);
            } else {
                showMessage("Registration failed (" + res.status + ").", true);
            }
        }).catch(function (err) {
            showMessage("Network error during registration: " + err.message, true);
        });
    }

    function showLoginForm() {
        isRegisterMode = false;
        dom(".auth-title").text("Login");
        dom(".form-toggler").text("Need an account? Register here");
        dom("#auth-repeat-row").addClass("hide");
        dom("#auth-submit-btn").text("Login");
        dom("#auth-submit-btn").off("click").on("click", handleLogin, false);
    }

    function showRegisterForm() {
        isRegisterMode = true;
        dom(".auth-title").text("Register");
        dom(".form-toggler").text("Already have an account? Login here");
        dom("#auth-repeat-row").removeClass("hide");
        dom("#auth-submit-btn").text("Register");
        dom("#auth-submit-btn").off("click").on("click", handleRegister, false);
    }

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
