/**
 * @fileoverview Crockfordian SPA micro-framework runtime.
 * Manages HTML5 History API routing, dynamic template mounting,
 * Server-Sent Events (SSE) subscriptions, and controller lifecycle
 * cleanup.
 */

/**
 * @typedef {Object} Application
 * @property {() => void} assignHrefs - Binds click navigation events.
 * @property {Object.<string, Function>} controllers - Controller registry.
 * @property {() => string} getCSRFToken - Retrieves current CSRF token.
 * @property {() => string} getPath - Returns active route path.
 * @property {(targetPath?: string) => void} loadRoute - Loads view route.
 * @property {function(string, Object): function(): void} subscribeToStream
 *     Subscribes to an SSE topic and returns a teardown function.
 */

/**
 * Creates and initializes the single-page application runtime.
 *
 * @returns {Readonly<Application>} Frozen application interface.
 */
function createApplication() {
    let activeCleanups = [];
    const controllers = Object.create(null);
    let path = "/";
    const router = {
        loadRoute: undefined
    };

    /**
     * Extracts the CSRF token from the meta tag in the document head.
     *
     * @returns {string} The CSRF token string, or an empty string if absent.
     */
    function getCSRFToken() {
        const meta = document.querySelector("meta[name=\"csrf-token\"]");
        if (meta !== null && meta.content) {
            return meta.content;
        }
        return "";
    }

    /**
     * Executes all registered teardown callbacks from active controllers.
     *
     * @returns {void}
     */
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

    /**
     * Renders the template payload into the DOM and invokes controllers.
     *
     * @param {Object} msg - Render response payload.
     * @param {string} msg.template - HTML markup for the view.
     * @param {string[]} [msg.controllers] - Names of controllers to mount.
     * @returns {void}
     */
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
                        msg.template
                    );
                    if (typeof cleanupFn === "function") {
                        activeCleanups.push(cleanupFn);
                    }
                }
            });
        }
    }

    /**
     * Displays an error message inside the primary view container.
     *
     * @param {string} [message] - Error message to render.
     * @returns {void}
     */
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

    /**
     * Fetches and renders a view template corresponding to a route path.
     *
     * @param {string} [targetPath] - Target URL path to navigate to.
     * @returns {void}
     */
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

    /**
     * Intercepts navigation clicks and delegates to client-side routing.
     *
     * @param {MouseEvent} [event] - The click event object.
     * @returns {void}
     */
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

    /**
     * Submits a logout request and reloads the current route.
     *
     * @param {Event} [event] - The triggering event.
     * @returns {void}
     */
    function handleLogout(event) {
        if (event && typeof event.preventDefault === "function") {
            event.preventDefault();
        }
        fetch("/logout", {
            headers: {
                "X-CSRF-Token": getCSRFToken(),
                "X-Requested-With": "XMLHttpRequest"
            },
            method: "POST"
        }).then(function () {
            const current = (
                (globalThis !== undefined && globalThis.location)
                ? globalThis.location.pathname
                : "/"
            );
            loadRoute(current);
        }).catch(function (err) {
            console.error("Logout request error:", err);
        });
    }

    /**
     * Binds navigation and action click handlers across the document.
     *
     * @returns {void}
     */
    function assignHrefs() {
        const oldHrefs = document.querySelectorAll("[data-href]");
        oldHrefs.forEach(function (el) {
            el.removeEventListener("click", navigate);
        });
        const newHrefs = document.querySelectorAll("[data-href]");
        newHrefs.forEach(function (el) {
            el.addEventListener("click", navigate);
        });
        const logoutBtns = document.querySelectorAll(
            "[data-action='logout']"
        );
        logoutBtns.forEach(function (el) {
            el.removeEventListener("click", handleLogout);
            el.addEventListener("click", handleLogout);
        });
    }

    /**
     * Establishes a Server-Sent Events (SSE) connection for a topic.
     *
     * @param {string} topic - Channel or board topic name.
     * @param {Object.<string, Function>} handlers - Event callback map.
     * @returns {function(): void} Teardown function to close the stream.
     */
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

    /**
     * Initializes window popstate event listeners and mounts initial route.
     *
     * @returns {void}
     */
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

/**
 * Application singleton instance.
 * @type {Readonly<Application>}
 */
const app = createApplication();
export default Object.freeze(app);
