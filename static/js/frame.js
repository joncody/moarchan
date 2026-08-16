function createApplication() {
    let activeCleanups = [];
    const controllers = Object.create(null);
    let path = "/";
    const router = {
        loadRoute: undefined
    };

    function getCSRFToken() {
        const meta = document.querySelector("meta[name=\"csrf-token\"]");
        if (meta !== null && meta.content) {
            return meta.content;
        }
        return "";
    }

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
                        globalThis,
                        msg.template
                    );
                    if (typeof cleanupFn === "function") {
                        activeCleanups.push(cleanupFn);
                    }
                }
            });
        }
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

    function loadRoute(targetPath) {
        if (typeof targetPath === "string" && targetPath.length > 0) {
            path = targetPath;
            if (
                globalThis !== undefined &&
                globalThis.history &&
                globalThis.location &&
                globalThis.location.pathname !== targetPath
            ) {
                globalThis.history.pushState(null, "", targetPath);
            }
        } else if (globalThis !== undefined && globalThis.location) {
            path = globalThis.location.pathname;
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
                    throw new Error(msg);
                }).catch(function () {
                    throw new Error("Error " + response.status);
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
    router.loadRoute = loadRoute;

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
        loadRoute(href);
    }

    function handleLogout(event) {
        if (event && typeof event.preventDefault === "function") {
            event.preventDefault();
        }
        fetch("/logout", {
            method: "POST",
            headers: {
                "X-Requested-With": "XMLHttpRequest",
                "X-CSRF-Token": getCSRFToken()
            }
        }).then(function () {
            const current = (globalThis !== undefined && globalThis.location ? globalThis.location.pathname : "/");
            loadRoute(current);
        }).catch(function (err) {
            console.error("Logout request error:", err);
        });
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
        const logoutBtns = document.querySelectorAll("[data-action='logout']");
        logoutBtns.forEach(function (el) {
            el.removeEventListener("click", handleLogout);
            el.addEventListener("click", handleLogout);
        });
    }

    function subscribeToStream(topic, handlers) {
        const url = "/api/stream?topic=" + encodeURIComponent(topic);
        const sse = new EventSource(url);
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
        if (globalThis !== undefined && globalThis.addEventListener) {
            globalThis.addEventListener("popstate", function () {
                loadRoute(globalThis.location.pathname);
            });
        }
        const initialPath = (
            globalThis !== undefined
                ? globalThis.location.pathname
                : "/"
        );
        loadRoute(initialPath);
    }

    init();

    return Object.freeze({
        assignHrefs,
        controllers,
        getCSRFToken,
        getPath: function () {
            return path;
        },
        loadRoute,
        subscribeToStream
    });
}

const app = createApplication();
export default Object.freeze(app);
