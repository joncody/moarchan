"use strict";

import roomer from "./roomer.js";

const global = (
    globalThis !== undefined
    ? globalThis
    : (
        window !== undefined
        ? window
        : globalThis
    )
);

const decoder = new TextDecoder("utf-8");
const app = {
    base: document.querySelector("[data-base]"),
    controllers: {},
    hashmatch: /^#*(.*)$/,
    hash: "/",
    hrefs: document.querySelectorAll("[data-href]"),
    retries: 0,
    socket: null
};

function changehash(event) {
    global.location.hash = event.currentTarget.dataset.href;
}

function assignHrefs() {
    app.hrefs.forEach((el) => el.removeEventListener("click", changehash));
    app.hrefs = document.querySelectorAll("[data-href]");
    app.hrefs.forEach((el) => el.addEventListener("click", changehash));
}

app.assignHrefs = assignHrefs;

global.addEventListener("hashchange", function () {
    const hash = app.hashmatch.exec(global.location.hash)[1];

    if (hash !== app.hash) {
        app.hash = hash;
        app.socket.send("request", app.hash);
    }
});

(function init() {
    app.socket = roomer((
        global.location.protocol === "https:"
        ? "wss:"
        : "ws:"
    ) + "//" + global.location.host + "/ws");

    app.socket.on("open", function () {
        app.retries = 0;
        app.hash = app.hashmatch.exec(global.location.hash)[1];
        if (!app.hash) {
            global.location.hash = "/";
            app.hash = "/";
        }
        app.socket.send("request", app.hash);
    });

    app.socket.on("close", function () {
        if (app.retries < 10) {
            global.setTimeout(init, 3000);
        }
        app.retries += 1;
    });

    app.socket.on("response", function (payload) {
        const msg = JSON.parse(decoder.decode(payload));

        app.base.innerHTML = msg.template;
        assignHrefs();
        if (msg.controllers) {
            msg.controllers.forEach(function (c) {
                if (app.controllers.hasOwnProperty(c)) {
                    app.controllers[c](global, msg.template);
                }
            });
        }
    });
}());

export default app;
