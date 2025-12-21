"use strict";

import gg from "./gg.js";
import wsframe from "./wsframe.js";

wsframe.controllers.auth = function (global) {
    "use strict";

    function enter() {
        const type = gg(".form-toggler").html() === "Login" ? "register" : "login";
        const alias = gg(".form-input[name='alias']").attr("value");
        const passhash = sjcl.codec.hex.fromBits(sjcl.hash.sha256.hash(gg(".form-input[name='password']").attr("value")));
        const passhash_repeat = sjcl.codec.hex.fromBits(sjcl.hash.sha256.hash(gg(".form-input[name='password-repeat']").attr("value")));
        const xhr = new XMLHttpRequest();
        const fd = new FormData();

        gg(".form-input[name='password']").attr("value", "");
        gg(".form-input[name='password-repeat']").attr("value", "");
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
    gg("button[name='enter']").on("click", enter, false);

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
    gg("button[name='leave']").on("click", leave, false);

    function toggleForm(e) {
        const node = gg(e.currentTarget);
        if (node.text()[0] === "Register") {
            node.html("Login");
            gg(".form-input[name='password-repeat']").removeClass("collapsed");
        } else {
            node.html("Register");
            gg(".form-input[name='password-repeat']").addClass("collapsed");
        }
    }
    gg(".form-toggler").on("click", toggleForm, false);

};
