"use strict";

import dom from "./dom.js";
import wsframe from "./wsframe.js";

wsframe.controllers.auth = function (global) {
    "use strict";

    function enter() {
        const type = dom(".form-toggler").html() === "Login" ? "register" : "login";
        const alias = dom(".form-input[name='alias']").attr("value");
        const passhash = sjcl.codec.hex.fromBits(sjcl.hash.sha256.hash(dom(".form-input[name='password']").attr("value")));
        const passhash_repeat = sjcl.codec.hex.fromBits(sjcl.hash.sha256.hash(dom(".form-input[name='password-repeat']").attr("value")));
        const xhr = new XMLHttpRequest();
        const fd = new FormData();

        dom(".form-input[name='password']").attr("value", "");
        dom(".form-input[name='password-repeat']").attr("value", "");
        if (type === "register" && passhash !== passhash_repeat) {
            return;
        }
        fd.append("alias", alias);
        fd.append("passhash", passhash);
        xhr.onload = function () {
            if (xhr.readyState === 4 && xhr.status === 200) {
                global.location.reload();
            }
        };
        xhr.open("POST", type === "login" ? "/login" : "/register", true);
        xhr.setRequestHeader("X-Requested-With", "XMLHttpRequest");
        xhr.responseType = "text";
        xhr.send(fd);
    }
    dom("button[name='enter']").on("click", enter, false);

    function leave() {
        const xhr = new XMLHttpRequest();

        xhr.onload = function () {
            if (xhr.readyState === 4 && xhr.status === 200) {
                global.location.reload();
            }
        };
        xhr.open("POST", "/logout", true);
        xhr.setRequestHeader("X-Requested-With", "XMLHttpRequest");
        xhr.responseType = "text";
        xhr.send(null);
    }
    dom("button[name='leave']").on("click", leave, false);

    function toggleForm(e) {
        const node = dom(e.currentTarget);
        if (node.text()[0] === "Register") {
            node.html("Login");
            dom(".form-input[name='password-repeat']").removeClass("collapsed");
        } else {
            node.html("Register");
            dom(".form-input[name='password-repeat']").addClass("collapsed");
        }
    }
    dom(".form-toggler").on("click", toggleForm, false);

};
