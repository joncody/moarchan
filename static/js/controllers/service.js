/**
 * @fileoverview Service controller coordinating imageboard feeds,
 * active threads, and real-time SSE updates.
 */

import dom from "../dom.js";
import frame from "../frame.js";
import topicsMap from "../components/topics-map.js";
import createTagHover from "../components/tag-hover.js";
import createReplyBox from "../components/reply-box.js";
import createPostActions from "../components/post-actions.js";
import createPostForm from "../components/post-form.js";

/**
 * Controller for board index and single-thread views.
 *
 * @returns {function(): void} Unified teardown function to clean up
 *     SSE streams and components.
 */
frame.controllers.service = function service() {
    const pathParts = globalThis.location.pathname.split("/").filter(Boolean);
    const topic = pathParts[0] || "";
    const isThreadView = (pathParts[1] === "thread");
    const threadHash = pathParts[2] || "";

    // Set topic header
    const headerText = topicsMap[topic] || "Unknown";
    dom(".topic-header").html("/" + topic + "/ - " + headerText);

    let replyBox;
    let postForm;

    // 1. Initialize Tag Hover Component
    const tagHover = createTagHover();

    // 2. Initialize Post Form Component
    postForm = createPostForm({
        isThreadView,
        onSubmitted: function () {
            if (replyBox) {
                replyBox.clear();
            }
        },
        threadHash,
        topic
    });

    // 3. Initialize Reply Box Component
    replyBox = createReplyBox(function (e) {
        postForm.submitReplyQuick(e);
    });

    // 4. Initialize Post Actions Component
    const postActions = createPostActions({
        onReplyClick: function (threadId, postHash) {
            replyBox.open(threadId, postHash);
        },
        topic
    });

    /**
     * Prepends a new thread element to the board view from SSE stream.
     *
     * @param {Object} data - SSE thread payload.
     * @param {string} data.hash - Thread identifier hash.
     * @param {string} data.html - Rendered thread HTML markup.
     * @returns {void}
     */
    function handleNewThread(data) {
        if (
            !data ||
            !data.hash ||
            document.getElementById("post-" + data.hash) !== null
        ) {
            return;
        }
        const boardEl = dom(".board").get(0);
        if (boardEl && data.html) {
            boardEl.insertAdjacentHTML("afterbegin", data.html);
            const threadEl = dom("#post-" + data.hash);
            postActions.bindThreadEvents(threadEl);
            tagHover.bindTags();
            frame.assignHrefs();
        }
    }

    /**
     * Appends a new reply to its target thread from SSE stream.
     *
     * @param {Object} data - SSE reply payload.
     * @param {string} data.hash - Reply identifier hash.
     * @param {string} data.thread - Target parent thread hash.
     * @param {string} data.html - Rendered reply HTML markup.
     * @param {string} [data.options] - Post options (e.g., 'sage').
     * @returns {void}
     */
    function handleNewReply(data) {
        if (
            !data ||
            !data.hash ||
            document.getElementById("post-" + data.hash) !== null
        ) {
            return;
        }
        const threadContainer = dom(
            "#post-" + data.thread + " .thread-container"
        );
        if (threadContainer.length() > 0 && data.html) {
            threadContainer.get(0).insertAdjacentHTML("beforeend", data.html);
            const replyEl = dom("#post-" + data.hash);
            postActions.bindReplyEvents(replyEl);
            tagHover.bindTags();
            postActions.initReplies(data.thread);

            // In topic board view, bump thread to top of feed
            // unless 'sage' is specified
            if (!isThreadView) {
                const isSage = (
                    data.options &&
                    data.options.toLowerCase().indexOf("sage") !== -1
                );
                if (!isSage) {
                    const boardEl = dom(".board").get(0);
                    const threadEl = document.getElementById(
                        "post-" + data.thread
                    );
                    if (boardEl && threadEl) {
                        boardEl.insertBefore(threadEl, boardEl.firstChild);
                    }
                }
            }
        }
    }

    /**
     * Removes or updates a deleted post element in response to SSE event.
     *
     * @param {Object} data - SSE delete-post payload.
     * @param {string} data.hash - Target post hash to delete.
     * @param {boolean} [data.file_only] - Whether only file was deleted.
     * @returns {void}
     */
    function handleDeletePost(data) {
        if (!data || !data.hash) {
            return;
        }
        const postEl = dom("#post-" + data.hash);
        if (postEl.length() === 0) {
            return;
        }
        if (data.file_only) {
            postEl.select(".post-image-metadata").remove();
            postEl.select(".post-image-container").remove();
        } else {
            postEl.remove();
        }
    }

    // 6. Bind Existing Server-Rendered Content
    postActions.bindThreadEvents(dom(".board"));
    postActions.bindReplyEvents(dom(".board"));
    tagHover.bindTags();

    if (!isThreadView) {
        dom(".thread").each(function (node) {
            if (node && node.id) {
                postActions.initReplies(node.id.slice(5));
            }
        });
    }

    // 7. Subscribe to Real-Time SSE Stream
    const streamCleanup = frame.subscribeToStream(topic, {
        "delete-post": handleDeletePost,
        "new-reply": handleNewReply,
        "new-thread": handleNewThread
    });

    // 8. Unified Teardown Lifecycle
    return function cleanup() {
        if (typeof streamCleanup === "function") {
            streamCleanup();
        }
        tagHover.cleanup();
        replyBox.cleanup();
        postActions.cleanup();
        postForm.cleanup();
    };
};
