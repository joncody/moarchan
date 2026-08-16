/**
 * @fileoverview Floating, draggable Quick Reply modal component.
 */

import dom from "../dom.js";

/**
 * @typedef {Object} ReplyBox
 * @property {() => void} clear - Clears form inputs and closes modal.
 * @property {(e?: Event) => void} close - Closes the quick reply modal.
 * @property {() => boolean} isOpen - Returns true if modal is visible.
 * @property {(threadId: string, postHash?: string) => void} open
 *     Opens the quick reply box for a thread with optional quote.
 * @property {() => void} cleanup - Unbinds drag and modal event listeners.
 */

/**
 * Creates the floating Quick Reply modal.
 *
 * @param {Function} [onPostReply] - Callback when user clicks 'Post'.
 * @returns {Readonly<ReplyBox>} Frozen quick reply modal controller.
 */
export default Object.freeze(function createReplyBox(onPostReply) {
    const replyBox = dom(".reply-box");
    const replyBoxHeader = dom(".reply-box-header");
    const replyBoxHeaderText = dom(".reply-box-header-text");
    const replyBoxPost = dom(".reply-box-post");
    const replyBoxClose = dom(".reply-box-close");
    let mouseX = 0;
    let mouseY = 0;

    /**
     * Handles modal dragging on mouse move.
     *
     * @param {MouseEvent} e - Mouse move event object.
     * @returns {void}
     */
    function dragging(e) {
        const topVal = parseInt(replyBox.css("top")[0], 10) || 0;
        const leftVal = parseInt(replyBox.css("left")[0], 10) || 0;
        const topPx = (topVal + e.clientY - mouseY) + "px";
        const leftPx = (leftVal - mouseX + e.clientX) + "px";

        replyBox.css("top", topPx).css("left", leftPx);
        mouseX = e.clientX;
        mouseY = e.clientY;
    }

    /**
     * Unbinds mousemove drag listener on mouse release.
     *
     * @returns {void}
     */
    function stopDrag() {
        dom(document.body).off("mousemove", dragging, false);
    }

    /**
     * Initiates modal dragging on header mousedown.
     *
     * @param {MouseEvent} e - Mousedown event object.
     * @returns {void}
     */
    function startDrag(e) {
        mouseX = e.clientX;
        mouseY = e.clientY;
        dom(document.body).on("mousemove", dragging, false);
        dom(document.body).once("mouseup", stopDrag, false);
    }

    /**
     * Hides the quick reply modal.
     *
     * @param {Event} [e] - Triggering event.
     * @returns {void}
     */
    function close(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        replyBox.addClass("hide");
    }

    /**
     * Opens the quick reply modal and pre-fills quote tags if provided.
     *
     * @param {string} threadId - Target thread identifier.
     * @param {string} [postHash] - Post hash to reference in comment.
     * @returns {void}
     */
    function open(threadId, postHash) {
        replyBox.data("thread", threadId);
        replyBoxHeaderText.html(threadId).attr("title", threadId);

        const commentInput = dom("#reply-box-comment").get(0);
        if (commentInput && postHash) {
            const currentVal = commentInput.value || "";
            commentInput.value = (
                currentVal
                ? currentVal + "\n"
                : ""
            ) + ">>" + postHash + "\n";
            commentInput.focus();
        }
        replyBox.removeClass("hide");
    }

    /**
     * Resets quick reply form inputs and hides the modal.
     *
     * @returns {void}
     */
    function clear() {
        [
            "#reply-box-name",
            "#reply-box-options",
            "#reply-box-password",
            "#reply-box-comment",
            "#reply-box-file"
        ].forEach(function (sel) {
            dom(sel).each(function (el) {
                if (el && el.value !== undefined) {
                    el.value = "";
                }
            });
        });
        close();
    }

    replyBoxHeader.on("mousedown", startDrag, false);
    replyBoxClose.on("click", close, false);
    if (typeof onPostReply === "function") {
        replyBoxPost.on("click", onPostReply, false);
    }

    return Object.freeze({
        cleanup: function () {
            replyBoxHeader.off("mousedown", startDrag, false);
            replyBoxClose.off("click", close, false);
            if (typeof onPostReply === "function") {
                replyBoxPost.off("click", onPostReply, false);
            }
            dom(document.body).off("mousemove", dragging, false);
        },
        clear,
        close,
        isOpen: function () {
            return !replyBox.hasClass("hide");
        },
        open
    });
});
