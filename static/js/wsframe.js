"use strict";

import gg from "./gg.js";
import wsrooms from "./wsrooms.js";

const global = globalThis || window || this;

const app = {
    base: gg("[data-base]"),
    controllers: {},
    hashmatch: /^#*(.*)$/,
    hash: "/",
    hrefs: gg("[data-href]"),
    retries: 0,
    socket: null
};

function request() {
    app.socket.send("request", app.hash);
}

function onhashchange() {
    const hash = app.hashmatch.exec(global.location.hash)[1];

    if (hash !== app.hash) {
        app.hash = hash;
        request();
    }
}
global.addEventListener("hashchange", onhashchange, false);

function changehash(e, node) {
    global.location.hash = node.data("href");
}

function assignHrefs() {
    app.hrefs = gg("[data-href]").on("click", changehash, false);
}

app.hrefListener = function hrefListener(listener) {
    app.hrefs.off("click");
    if (typeof listener !== "function") {
        listener = changehash;
    }
    app.hrefs = gg("[data-href]").on("click", listener, false);
};

function onopen() {
    app.retries = 0;
    app.hash = app.hashmatch.exec(global.location.hash)[1];
    if (!app.hash) {
        global.location.hash = "/";
        app.hash = "/";
    }
    request();
}

function onresponse(payload) {
    const msg = JSON.parse(gg.utils.toStringFromCodes(payload));

    gg.removeKeyboardListeners();
    gg.removeMouseListeners();
    app.base.html(msg.template);
    app.hrefs.off("click");
    assignHrefs();
    if (msg.controllers) {
        msg.controllers.forEach(function (c) {
            if (app.controllers.hasOwnProperty(c)) {
                app.controllers[c](global);
            }
        });
    }
}

(function init() {
    app.socket = wsrooms((global.location.protocol === "https:" ? "wss:" : "ws:") + "//" + global.location.host + "/ws");
    app.socket.on("open", onopen);
    app.socket.on("close", function () {
        if (app.retries < 10) {
            global.setTimeout(init, 3000);
        }
        app.retries += 1;
    });
    app.socket.on("error", function (err) {
        console.log(err);
        if (app.retries < 10) {
            global.setTimeout(init, 3000);
        }
    });
    app.socket.on("response", onresponse);
}());

export default app;
