import dom from "../dom.js";

export default function createReplyBox(global, onPostReply) {
    const replyBox = dom(".reply-box");
    const replyBoxHeader = dom(".reply-box-header");
    const replyBoxHeaderText = dom(".reply-box-header-text");
    const replyBoxPost = dom(".reply-box-post");
    const replyBoxClose = dom(".reply-box-close");
    let mouseX = 0;
    let mouseY = 0;

    function dragging(e) {
        const topVal = parseInt(replyBox.css("top")[0], 10) || 0;
        const leftVal = parseInt(replyBox.css("left")[0], 10) || 0;
        const topPx = (topVal + e.clientY - mouseY) + "px";
        const leftPx = (leftVal - mouseX + e.clientX) + "px";

        replyBox.css("top", topPx).css("left", leftPx);
        mouseX = e.clientX;
        mouseY = e.clientY;
    }

    function stopDrag() {
        dom(document.body).off("mousemove", dragging, false);
    }

    function startDrag(e) {
        mouseX = e.clientX;
        mouseY = e.clientY;
        dom(document.body).on("mousemove", dragging, false);
        dom(document.body).once("mouseup", stopDrag, false);
    }

    function close(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        replyBox.addClass("hide");
    }

    function open(threadId, postHash) {
        replyBox.data("thread", threadId);
        replyBoxHeaderText.html(threadId).attr("title", threadId);

        const commentInput = dom("#reply-box-comment").get(0);
        if (commentInput && postHash) {
            const currentVal = commentInput.value || "";
            commentInput.value = (currentVal ? currentVal + "\n" : "") + ">>" + postHash + "\n";
            commentInput.focus();
        }
        replyBox.removeClass("hide");
    }

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
        clear,
        close,
        isOpen: function () {
            return !replyBox.hasClass("hide");
        },
        open,
        cleanup: function () {
            replyBoxHeader.off("mousedown", startDrag, false);
            replyBoxClose.off("click", close, false);
            if (typeof onPostReply === "function") {
                replyBoxPost.off("click", onPostReply, false);
            }
            dom(document.body).off("mousemove", dragging, false);
        }
    });
}
