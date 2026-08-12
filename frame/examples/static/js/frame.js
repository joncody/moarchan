"use strict";

function get_global_environment() {
    if (typeof globalThis !== "undefined") {
        return globalThis;
    }
    if (typeof window !== "undefined") {
        return window;
    }
    return undefined;
}

const global_env = get_global_environment();

const app = {
    activeCleanups: [],
    base: document.querySelector("[data-base]"),
    controllers: {},
    path: "/",
    hrefs: document.querySelectorAll("[data-href]")
};

function navigate(e) {
    if (e && e.currentTarget && e.currentTarget.dataset && e.currentTarget.dataset.href) {
        e.preventDefault();
        const href = e.currentTarget.dataset.href;
        if (global_env.location.pathname !== href) {
            global_env.history.pushState(null, "", href);
            loadRoute(href);
        }
    }
}

function assignHrefs() {
    app.hrefs.forEach(function (el) {
        el.removeEventListener("click", navigate);
    });
    app.hrefs = document.querySelectorAll("[data-href]");
    app.hrefs.forEach(function (el) {
        el.addEventListener("click", navigate);
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

function loadRoute(path) {
    app.path = path || global_env.location.pathname;

    fetch("/api/render?path=" + encodeURIComponent(app.path), {
        headers: {
            "X-Requested-With": "XMLHttpRequest"
        }
    }).then(function (response) {
        if (!response.ok) {
            return response.json().then(function (errData) {
                throw new Error(errData.error || ("Error " + response.status));
            }).catch(function () {
                throw new Error("Error " + response.status);
            });
        }
        return response.json();
    }).then(function (msg) {
        runCleanups();

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
    }).catch(function (err) {
        runCleanups();
        app.base.innerHTML = (
            "<div class=\"page\"><div class=\"pink-section\">" +
            "<h3 class=\"box-header pink-header\">Error</h3><p class=\"box-message\">" +
            (err.message || "An unexpected error occurred.") +
            "</p></div></div>"
        );
        assignHrefs();
    });
}

global_env.addEventListener("popstate", function () {
    loadRoute(global_env.location.pathname);
});

function createEventSource(url) {
    return new EventSource(url);
}

app.subscribeToStream = function (topic, handlers) {
    const sse = createEventSource("/api/stream?topic=" + encodeURIComponent(topic));

    if (handlers && typeof handlers === "object") {
        Object.keys(handlers).forEach(function (eventType) {
            sse.addEventListener(eventType, function (e) {
                try {
                    const data = JSON.parse(e.data);
                    handlers[eventType](data);
                } catch (err) {
                    console.error("SSE JSON parse error: ", err);
                }
            });
        });
    }

    return function cleanup() {
        sse.close();
    };
};

(function init() {
    loadRoute(global_env.location.pathname);
}());

export default app;
