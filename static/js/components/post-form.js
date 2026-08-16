/**
 * @fileoverview Post submission form component for creating new threads
 * and replies.
 */

import dom from "../dom.js";
import frame from "../frame.js";

/**
 * @typedef {Object} PostForm
 * @property {() => void} clearForms
 *     Clears input fields across post and reply forms.
 * @property {(e: Event) => void} submitReplyQuick
 *     Submits reply data from floating quick-reply modal.
 * @property {() => void} cleanup
 *     Unbinds form event listeners.
 */

/**
 * Creates the post form handler for thread and reply creation.
 *
 * @param {Object} config - Component configuration parameters.
 * @param {string} config.topic - The current board slug (e.g., 'g').
 * @param {boolean} config.isThreadView - True if rendering inside a thread.
 * @param {string} [config.threadHash] - Target thread hash if in thread view.
 * @param {Function} [config.onSubmitted] - Post-submission callback.
 * @returns {Readonly<PostForm>} Frozen post form controller.
 */
export default Object.freeze(function createPostForm(config) {
    const topic = config.topic;
    const isThreadView = config.isThreadView;
    const threadHash = config.threadHash;
    const onSubmitted = config.onSubmitted;

    /**
     * Toggles visibility of the top blotter banner section.
     *
     * @param {Event} [e] - Triggering click event.
     * @returns {void}
     */
    function toggleBlotter(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        dom(".blotter").toggleClass("hide");
        dom(".hide-blotter-container").toggleClass("hide");
        dom(".show-all-blotter-container").toggleClass("hide");
        dom(".show-blotter-container").toggleClass("hide");
    }

    /**
     * Displays the new thread submission form.
     *
     * @param {Event} [e] - Triggering click event.
     * @returns {void}
     */
    function showNewPostForm(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        dom(".new-post").addClass("hide");
        dom(".new-post-form").removeClass("hide");
    }

    /**
     * Resets all post and reply form input values to empty strings.
     *
     * @returns {void}
     */
    function clearForms() {
        [
            "#new-post-name",
            "#new-post-subject",
            "#new-post-options",
            "#new-post-password",
            "#new-post-comment",
            "#new-post-file"
        ].forEach(function (sel) {
            dom(sel).each(function (el) {
                if (el && el.value !== undefined) {
                    el.value = "";
                }
            });
        });
        if (typeof onSubmitted === "function") {
            onSubmitted();
        }
    }

    /**
     * Validates and submits a new thread creation request.
     *
     * @param {Event} [e] - Triggering form submission event.
     * @returns {void}
     */
    function submitThread(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const nameInput = dom("#new-post-name").get(0);
        const subjectInput = dom("#new-post-subject").get(0);
        const optionsInput = dom("#new-post-options").get(0);
        const passInput = dom("#new-post-password").get(0);
        const commentInput = dom("#new-post-comment").get(0);
        const fileInput = dom("#new-post-file").get(0);
        const files = (fileInput && fileInput.files) || [];

        if (files.length === 0) {
            return globalThis.alert("File required to create a new thread.");
        }
        if (files[0].size > 32 * 1024 * 1024) {
            return globalThis.alert(
                "File exceeds maximum limit of 32 MB."
            );
        }

        const fd = new FormData();
        fd.append("topic", topic);
        fd.append("name", (nameInput && nameInput.value) || "Anonymous");
        fd.append("subject", (subjectInput && subjectInput.value) || "");
        fd.append("options", (optionsInput && optionsInput.value) || "");
        fd.append("password", (passInput && passInput.value) || "");
        fd.append("comment", (commentInput && commentInput.value) || "");
        fd.append("file", files[0]);

        fetch("/api/threads", {
            body: fd,
            headers: {
                "X-CSRF-Token": frame.getCSRFToken()
            },
            method: "POST"
        })
            .then(function (res) {
                if (!res.ok) {
                    return res.text().then(function (msg) {
                        globalThis.alert(msg);
                    });
                }
                clearForms();
            })
            .catch(function (err) {
                globalThis.alert("Network error: " + err.message);
            });
    }

    /**
     * Validates and submits a reply to an existing thread.
     *
     * @param {Event} [e] - Triggering event.
     * @param {boolean} [fromQuickReply] - Whether submitted via quick reply.
     * @returns {void}
     */
    function submitReply(e, fromQuickReply) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        let targetThread = threadHash;
        let prefix = "#new-post-";
        if (fromQuickReply) {
            prefix = "#reply-box-";
            const boxThread = dom(".reply-box").data("thread");
            targetThread = (
                Array.isArray(boxThread)
                ? boxThread[0]
                : boxThread
            );
        }
        if (!targetThread) {
            return globalThis.alert("Unable to locate target thread ID.");
        }

        const nameInput = dom(prefix + "name").get(0);
        const optionsInput = dom(prefix + "options").get(0);
        const passInput = dom(prefix + "password").get(0);
        const commentInput = dom(prefix + "comment").get(0);
        const fileInput = dom(prefix + "file").get(0);
        const comment = (commentInput && commentInput.value) || "";

        if (!comment.trim()) {
            return globalThis.alert("Comment is required to post a reply.");
        }
        const files = (fileInput && fileInput.files) || [];
        if (files.length > 0 && files[0].size > 8 * 1024 * 1024) {
            return globalThis.alert(
                "File exceeds maximum allowed limit of 8 MB."
            );
        }

        const fd = new FormData();
        fd.append("topic", topic);
        fd.append("thread", targetThread);
        fd.append("name", (nameInput && nameInput.value) || "Anonymous");
        fd.append("options", (optionsInput && optionsInput.value) || "");
        fd.append("password", (passInput && passInput.value) || "");
        fd.append("comment", comment);
        if (files.length > 0) {
            fd.append("file", files[0]);
        }

        fetch("/api/replies", {
            body: fd,
            headers: {
                "X-CSRF-Token": frame.getCSRFToken()
            },
            method: "POST"
        })
            .then(function (res) {
                if (!res.ok) {
                    return res.text().then(function (msg) {
                        globalThis.alert(msg);
                    });
                }
                clearForms();
            })
            .catch(function (err) {
                globalThis.alert("Network error: " + err.message);
            });
    }

    dom(".hide-blotter").on("click", toggleBlotter, false);
    dom(".show-blotter").on("click", toggleBlotter, false);
    dom(".new-post").on("click", showNewPostForm, false);

    if (isThreadView) {
        dom("#new-post-button").on("click", function (e) {
            submitReply(e, false);
        }, false);
    } else {
        dom("#new-post-button").on("click", submitThread, false);
    }

    return Object.freeze({
        cleanup: function () {
            dom(".hide-blotter").off("click", toggleBlotter, false);
            dom(".show-blotter").off("click", toggleBlotter, false);
            dom(".new-post").off("click", showNewPostForm, false);
            dom("#new-post-button").off("click");
        },
        clearForms,
        submitReplyQuick: function (e) {
            submitReply(e, true);
        }
    });
});
