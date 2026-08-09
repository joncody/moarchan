"use strict";

import roomer from "./roomer.js";

function get_global_environment() {
    if (globalThis !== undefined) {
        return globalThis;
    }
    if (window !== undefined) {
        return window;
    }
    return undefined;
}

const global_env = get_global_environment();

const decoder = new TextDecoder("utf-8");
const app = {
    activeCleanups: [],
    base: document.querySelector("[data-base]"),
    controllers: {},
    hash: "/",
    hashmatch: /^#*(.*)$/,
    hrefs: document.querySelectorAll("[data-href]"),
    retries: 0,
    socket: null
};

function changehash(event) {
    if (event && event.currentTarget && event.currentTarget.dataset) {
        global_env.location.hash = event.currentTarget.dataset.href;
    }
}

function assignHrefs() {
    app.hrefs.forEach(function (el) {
        el.removeEventListener("click", changehash);
    });
    app.hrefs = document.querySelectorAll("[data-href]");
    app.hrefs.forEach(function (el) {
        el.addEventListener("click", changehash);
    });
}

app.assignHrefs = assignHrefs;

function runCleanups() {
    if (Array.isArray(app.activeCleanups)) {
        app.activeCleanups.forEach(function (fn) {
            if (typeof fn === "function") {
                fn();
            }
        });
    }
    app.activeCleanups = [];
}

global_env.addEventListener("hashchange", function () {
    const match = app.hashmatch.exec(global_env.location.hash);
    const hash = (match && match[1]) ? match[1] : "/";

    if (hash !== app.hash) {
        app.hash = hash;
        if (app.socket && typeof app.socket.send === "function") {
            app.socket.send("request", app.hash);
        }
    }
});

(function init() {
    const protocol = (
        global_env.location.protocol === "https:"
        ? "wss:"
        : "ws:"
    );
    app.socket = roomer(
        protocol + "//" + global_env.location.host + "/ws"
    );

    app.socket.on("open", function () {
        app.retries = 0;
        const match = app.hashmatch.exec(global_env.location.hash);
        app.hash = (match && match[1]) ? match[1] : "";
        if (!app.hash) {
            global_env.location.hash = "/";
            app.hash = "/";
        }
        app.socket.send("request", app.hash);
    });

    app.socket.on("close", function () {
        if (app.retries < 10) {
            global_env.setTimeout(init, 3000);
        }
        app.retries += 1;
    });

    app.socket.on("response", function (payload) {
        let msg;
        try {
            msg = JSON.parse(decoder.decode(payload));
        } catch (err) {
            return;
        }

        runCleanups();

        if (typeof app.socket.purge === "function") {
            app.socket.purge();
        }

        app.base.innerHTML = msg.template;
        assignHrefs();

        if (Array.isArray(msg.controllers)) {
            msg.controllers.forEach(function (c) {
                if (
                    Object.prototype.hasOwnProperty.call(
                        app.controllers,
                        c
                    )
                ) {
                    const cleanupFn = app.controllers[c](
                        global_env,
                        msg.template
                    );
                    if (typeof cleanupFn === "function") {
                        app.activeCleanups.push(cleanupFn);
                    }
                }
            });
        }
    });

    app.socket.on("error", function (payload) {
        let errData;
        try {
            errData = JSON.parse(decoder.decode(payload));
        } catch (err) {
            errData = { error: "An unexpected error occurred." };
        }

        runCleanups();

        if (typeof app.socket.purge === "function") {
            app.socket.purge();
        }

        app.base.innerHTML = (
            "<div class=\"page\"><div class=\"pink-section\">" +
            "<h3 class=\"box-header pink-header\">Error " +
            (errData.status || 500) +
            "</h3><p class=\"box-message\">" +
            (errData.error || "An unexpected error occurred.") +
            "</p></div></div>"
        );
        assignHrefs();
    });
}());

export default app;
