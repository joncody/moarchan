function getGlobalEnvironment() {
    if (globalThis !== undefined) {
        return globalThis;
    }
    if (window !== undefined) {
        return window;
    }
    return undefined;
}

const globalEnv = getGlobalEnvironment();

function createApplication() {
    let activeCleanups = [];
    const controllers = Object.create(null);
    let path = "/";

    function runCleanups() {
        if (Array.isArray(activeCleanups)) {
            activeCleanups.forEach(function (fn) {
                if (typeof fn === "function") {
                    fn();
                }
            });
        }
        activeCleanups = [];
    }

    function createEventSource(url) {
        return Reflect.construct(EventSource, [url]);
    }

    function createError(message) {
        return Reflect.construct(Error, [message]);
    }

    function navigate(event) {
        if (
            event === undefined ||
            event.currentTarget === null ||
            event.currentTarget.dataset === undefined ||
            event.currentTarget.dataset.href === undefined
        ) {
            return;
        }

        event.preventDefault();
        const href = event.currentTarget.dataset.href;
        if (globalEnv.location.pathname !== href) {
            globalEnv.history.pushState(null, "", href);
            loadRoute(href);
        }
    }

    function assignHrefs() {
        const oldHrefs = document.querySelectorAll("[data-href]");
        oldHrefs.forEach(function (el) {
            el.removeEventListener("click", navigate);
        });

        const newHrefs = document.querySelectorAll("[data-href]");
        newHrefs.forEach(function (el) {
            el.addEventListener("click", navigate);
        });
    }

    function renderError(message) {
        const base = document.querySelector("[data-base]");
        if (base === null) {
            return;
        }
        const safeMsg = message || "An unexpected error occurred.";
        base.innerHTML = (
            "<div class=\"page\"><div class=\"pink-section\">" +
            "<h3 class=\"box-header pink-header\">Error</h3>" +
            "<p class=\"box-message\">" +
            safeMsg +
            "</p></div></div>"
        );
        assignHrefs();
    }

    function handleRenderSuccess(msg) {
        runCleanups();

        const base = document.querySelector("[data-base]");
        if (base !== null) {
            base.innerHTML = msg.template;
        }

        assignHrefs();

        if (Array.isArray(msg.controllers)) {
            msg.controllers.forEach(function (name) {
                if (
                    Object.prototype.hasOwnProperty.call(
                        controllers,
                        name
                    )
                ) {
                    const cleanupFn = controllers[name](
                        globalEnv,
                        msg.template
                    );
                    if (typeof cleanupFn === "function") {
                        activeCleanups.push(cleanupFn);
                    }
                }
            });
        }
    }

    function loadRoute(targetPath) {
        if (typeof targetPath === "string" && targetPath.length > 0) {
            path = targetPath;
        } else if (globalEnv !== undefined && globalEnv.location) {
            path = globalEnv.location.pathname;
        }

        const endpoint = "/api/render?path=" + encodeURIComponent(path);

        fetch(endpoint, {
            headers: {
                "X-Requested-With": "XMLHttpRequest"
            }
        }).then(function (response) {
            if (!response.ok) {
                return response.json().then(function (errData) {
                    const msg = errData.error || (
                        "Error " + response.status
                    );
                    throw createError(msg);
                }).catch(function () {
                    throw createError("Error " + response.status);
                });
            }
            return response.json();
        }).then(function (msg) {
            handleRenderSuccess(msg);
        }).catch(function (err) {
            runCleanups();
            renderError(err.message);
        });
    }

    function subscribeToStream(topic, handlers) {
        const url = "/api/stream?topic=" + encodeURIComponent(topic);
        const sse = createEventSource(url);

        if (handlers !== null && typeof handlers === "object") {
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
    }

    function init() {
        if (globalEnv !== undefined && globalEnv.addEventListener) {
            globalEnv.addEventListener("popstate", function () {
                loadRoute(globalEnv.location.pathname);
            });
        }
        const initialPath = (
            globalEnv !== undefined
            ? globalEnv.location.pathname
            : "/"
        );
        loadRoute(initialPath);
    }

    init();

    return Object.freeze({
        assignHrefs,
        controllers,
        getPath: function () {
            return path;
        },
        loadRoute,
        subscribeToStream
    });
}

const app = createApplication();

export default Object.freeze(app);
