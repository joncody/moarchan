/**
 * @fileoverview Component managing user interactions on posts
 * (expand/collapse, hide, delete).
 */

import dom from "../dom.js";
import frame from "../frame.js";

/**
 * @typedef {Object} PostActions
 * @property {(replyEl: Object) => void} bindReplyEvents
 *     Binds interaction listeners to reply elements.
 * @property {(threadEl: Object) => void} bindThreadEvents
 *     Binds interaction listeners to thread elements.
 * @property {(hash: string) => void} initReplies
 *     Calculates and initializes omitted replies summary.
 * @property {() => void} cleanup
 *     Unbinds component-level event listeners.
 */

/**
 * Creates post action handlers.
 *
 * @param {Object} options - Component options.
 * @param {string} options.topic - Current board topic.
 * @param {Function} [options.onReplyClick] - Reply button callback.
 * @returns {Readonly<PostActions>} Frozen post actions API.
 */
export default Object.freeze(function createPostActions(options) {
    const topic = options.topic;
    const onReplyClick = options.onReplyClick;

    /**
     * Extracts the post hash identifier from event target.
     *
     * @param {Event} e - DOM event object.
     * @returns {string|undefined} The post hash.
     */
    function getPostHash(e) {
        const target = dom(e.currentTarget || e.target);
        const dataVal = target.data("post");
        return (
            Array.isArray(dataVal)
            ? dataVal[0]
            : dataVal
        );
    }

    /**
     * Toggles thumbnail image expansion between preview and full size.
     *
     * @param {Event} [e] - Triggering click event.
     * @returns {void}
     */
    function toggleImageExpansion(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const container = dom(e.currentTarget);
        const img = container.select(".post-image").get(0);
        if (!img) {
            return;
        }
        const fullSrc = (
            img.getAttribute("data-full") ||
            container.attr("href")[0]
        );
        const thumbSrc = img.getAttribute("data-thumb") || img.src;
        const parentBody = container.parents();

        if (img.classList.contains("expanded")) {
            img.classList.remove("expanded");
            img.src = thumbSrc;
            parentBody.removeClass("has-expanded");
        } else {
            img.classList.add("expanded");
            img.src = fullSrc;
            parentBody.addClass("has-expanded");
        }
    }

    /**
     * Toggles visibility of an entire thread container.
     *
     * @param {Event} [e] - Triggering click event.
     * @returns {void}
     */
    function toggleThread(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const hash = getPostHash(e);
        if (hash) {
            dom("#post-" + hash).toggleClass("hide-thread");
        }
    }

    /**
     * Toggles between summary view and all replies for a thread.
     *
     * @param {Event} [e] - Triggering click event.
     * @returns {void}
     */
    function toggleReplies(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const hash = getPostHash(e);
        if (!hash) {
            return;
        }
        const thread = dom("#post-" + hash);
        const replies = thread.select(".reply-container");
        thread.toggleClass("show-replies");

        const summaryEl = thread.select(".post-summary");
        if (!thread.hasClass("show-replies") && replies.length() > 5) {
            const omitted = replies.length() - 5;
            const href = "/" + topic + "/thread/" + hash;
            summaryEl.html(
                omitted +
                " posts omitted. " +
                "<span class=\"blue-text-link\" data-href=\"" +
                href +
                "\">Click here</span> to view."
            );
        } else {
            summaryEl.html("Showing all replies.");
        }
        frame.assignHrefs();
    }

    /**
     * Hides a specific post or thread and closes options menu.
     *
     * @param {Event} [e] - Triggering click event.
     * @returns {void}
     */
    function hidePost(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const hash = getPostHash(e);
        if (!hash) {
            return;
        }
        const post = dom("#post-" + hash);
        if (post.hasClass("thread")) {
            post.addClass("hide-thread");
        } else if (post.hasClass("reply")) {
            post.addClass("hide-reply");
        }
        dom(".post-options-menu").addClass("hide");
    }

    /**
     * Restores visibility of a hidden post or thread.
     *
     * @param {Event} [e] - Triggering click event.
     * @returns {void}
     */
    function unhidePost(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const hash = getPostHash(e);
        if (!hash) {
            return;
        }
        const post = dom("#post-" + hash);
        if (post.hasClass("thread")) {
            post.removeClass("hide-thread");
        } else if (post.hasClass("reply")) {
            post.removeClass("hide-reply");
        }
        dom(".post-options-menu").addClass("hide");
    }

    /**
     * Opens the contextual post options dropdown menu.
     *
     * @param {Event} [e] - Triggering click event.
     * @returns {void}
     */
    function showPostOptions(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const hash = getPostHash(e);
        if (!hash) {
            return;
        }
        const menu = dom("#post-menu-" + hash);
        dom(".post-options-menu").addClass("hide");
        menu.removeClass("hide");
        setTimeout(function () {
            dom(document.body).once("click", function (evt) {
                if (!dom(evt.target).hasClass("post-options-arrow")) {
                    menu.addClass("hide");
                }
            }, false);
        }, 0);
    }

    /**
     * Dispatches reply click to populate quick reply with post quote.
     *
     * @param {Event} [e] - Triggering click event.
     * @returns {void}
     */
    function handleReplyClick(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const target = dom(e.currentTarget);
        const threadVal = target.data("thread");
        const threadId = (
            Array.isArray(threadVal)
            ? threadVal[0]
            : threadVal
        );
        const postHash = target.text()[0] || "";
        if (typeof onReplyClick === "function") {
            onReplyClick(threadId, postHash);
        }
    }

    /**
     * Configures initial omitted reply counts and summary markup.
     *
     * @param {string} hash - Target thread hash.
     * @returns {void}
     */
    function initReplies(hash) {
        const threadDom = dom("#post-" + hash);
        const replies = threadDom.select(".reply-container");
        const summaryEl = threadDom.select(".post-summary");

        if (replies.length() > 0) {
            if (replies.length() > 5) {
                const omitted = replies.length() - 5;
                const href = "/" + topic + "/thread/" + hash;
                threadDom.addClass("show-summary");
                summaryEl.html(
                    omitted +
                    " posts omitted. " +
                    "<span class=\"blue-text-link\" data-href=\"" +
                    href +
                    "\">Click here</span> to view."
                );
            } else {
                threadDom.addClass("show-replies");
            }
        }
        frame.assignHrefs();
    }

    /**
     * Binds event listeners to interactive elements within a thread.
     *
     * @param {Object} threadEl - Wrapped thread DOM element.
     * @returns {void}
     */
    function bindThreadEvents(threadEl) {
        threadEl.selectAll(".post-show-hide-thread")
            .on("click", toggleThread, false);
        threadEl.selectAll(".post-show-hide-replies")
            .on("click", toggleReplies, false);
        threadEl.selectAll(".hide-post")
            .on("click", hidePost, false);
        threadEl.selectAll(".unhide-post")
            .on("click", unhidePost, false);
        threadEl.selectAll(".post-options-arrow")
            .on("click", showPostOptions, false);
        threadEl.selectAll(".post-reply-to")
            .on("click", handleReplyClick, false);
        threadEl.selectAll(".post-image-container")
            .on("click", toggleImageExpansion, false);
    }

    /**
     * Binds event listeners to interactive elements within a reply.
     *
     * @param {Object} replyEl - Wrapped reply DOM element.
     * @returns {void}
     */
    function bindReplyEvents(replyEl) {
        replyEl.selectAll(".hide-post")
            .on("click", hidePost, false);
        replyEl.selectAll(".unhide-post")
            .on("click", unhidePost, false);
        replyEl.selectAll(".post-options-arrow")
            .on("click", showPostOptions, false);
        replyEl.selectAll(".post-reply-to")
            .on("click", handleReplyClick, false);
        replyEl.selectAll(".post-image-container")
            .on("click", toggleImageExpansion, false);
    }

    /**
     * Sends a deletion request for all checked posts.
     *
     * @param {Event} [e] - Triggering click event.
     * @returns {void}
     */
    function deleteSelectedPosts(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const checkedBoxes = Array.from(
            document.querySelectorAll(".post-checkbox:checked")
        );
        if (checkedBoxes.length === 0) {
            return globalThis.alert("No posts selected for deletion.");
        }

        const fileOnlyCheckbox = dom("#delete-file-only").get(0);
        const fileOnly = fileOnlyCheckbox && fileOnlyCheckbox.checked;
        const passInput = dom("#delete-post-password").get(0);
        const password = (passInput && passInput.value) || "";

        if (
            !globalThis.confirm(
                "Are you sure you want to delete the selected item(s)?"
            )
        ) {
            return;
        }

        checkedBoxes.forEach(function (box) {
            const container = (
                box.closest(".reply") || box.closest(".thread")
            );
            if (
                container &&
                container.id &&
                container.id.startsWith("post-")
            ) {
                const hash = container.id.slice(5);
                const fd = new FormData();
                fd.append("hash", hash);
                fd.append("password", password);
                if (fileOnly) {
                    fd.append("file_only", "true");
                }
                fetch("/api/posts/delete", {
                    body: fd,
                    headers: {
                        "X-CSRF-Token": frame.getCSRFToken()
                    },
                    method: "POST"
                })
                    .then(function (res) {
                        if (!res.ok) {
                            return res.text().then(function (t) {
                                globalThis.alert(t);
                            });
                        }
                    })
                    .catch(function (err) {
                        console.error("Delete failed:", err);
                    });
            }
        });
    }

    dom("button.delete").on("click", deleteSelectedPosts, false);

    return Object.freeze({
        bindReplyEvents,
        bindThreadEvents,
        cleanup: function () {
            dom("button.delete").off("click", deleteSelectedPosts, false);
        },
        initReplies
    });
});
