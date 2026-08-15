import dom from "../dom.js";
import frame from "../frame.js";

export default function createPostActions(global, options) {
    const topic = options.topic;
    const onReplyClick = options.onReplyClick;

    function getPostHash(e) {
        const target = dom(e.currentTarget || e.target);
        const dataVal = target.data("post");
        return Array.isArray(dataVal) ? dataVal[0] : dataVal;
    }

    function toggleImageExpansion(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const container = dom(e.currentTarget);
        const img = container.select(".post-image").get(0);
        if (!img) {
            return;
        }

        const fullSrc = img.getAttribute("data-full") || container.attr("href")[0];
        const thumbSrc = img.getAttribute("data-thumb") || img.src;

        if (img.classList.contains("expanded")) {
            img.classList.remove("expanded");
            img.src = thumbSrc;
        } else {
            img.classList.add("expanded");
            img.src = fullSrc;
        }
    }

    function toggleThread(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const hash = getPostHash(e);
        if (hash) {
            dom("#post-" + hash).toggleClass("hide-thread");
        }
    }

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
                omitted + " posts omitted. <span class=\"blue-text-link\" data-href=\"" +
                href + "\">Click here</span> to view."
            );
        } else {
            summaryEl.html("Showing all replies.");
        }
        frame.assignHrefs();
    }

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

    function handleReplyClick(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const target = dom(e.currentTarget);
        const threadVal = target.data("thread");
        const threadId = Array.isArray(threadVal) ? threadVal[0] : threadVal;
        const postHash = target.text()[0] || "";

        if (typeof onReplyClick === "function") {
            onReplyClick(threadId, postHash);
        }
    }

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
                    omitted + " posts omitted. <span class=\"blue-text-link\" data-href=\"" +
                    href + "\">Click here</span> to view."
                );
            } else {
                threadDom.addClass("show-replies");
            }
        }
        frame.assignHrefs();
    }

    function bindThreadEvents(threadEl) {
        threadEl.selectAll(".post-show-hide-thread").on("click", toggleThread, false);
        threadEl.selectAll(".post-show-hide-replies").on("click", toggleReplies, false);
        threadEl.selectAll(".hide-post").on("click", hidePost, false);
        threadEl.selectAll(".unhide-post").on("click", unhidePost, false);
        threadEl.selectAll(".post-options-arrow").on("click", showPostOptions, false);
        threadEl.selectAll(".post-reply-to").on("click", handleReplyClick, false);
        threadEl.selectAll(".post-image-container").on("click", toggleImageExpansion, false);
    }

    function bindReplyEvents(replyEl) {
        replyEl.selectAll(".hide-post").on("click", hidePost, false);
        replyEl.selectAll(".unhide-post").on("click", unhidePost, false);
        replyEl.selectAll(".post-options-arrow").on("click", showPostOptions, false);
        replyEl.selectAll(".post-reply-to").on("click", handleReplyClick, false);
        replyEl.selectAll(".post-image-container").on("click", toggleImageExpansion, false);
    }

    function deleteSelectedPosts(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const checkedBoxes = Array.from(document.querySelectorAll(".post-checkbox:checked"));
        if (checkedBoxes.length === 0) {
            return global.alert("No posts selected for deletion.");
        }

        const fileOnlyCheckbox = dom("#delete-file-only").get(0);
        const fileOnly = fileOnlyCheckbox && fileOnlyCheckbox.checked;
        const passInput = dom("#delete-post-password").get(0);
        const password = (passInput && passInput.value) || "";

        if (!global.confirm("Are you sure you want to delete the selected item(s)?")) {
            return;
        }

        checkedBoxes.forEach(function (box) {
            const container = box.closest(".reply") || box.closest(".thread");
            if (container && container.id && container.id.startsWith("post-")) {
                const hash = container.id.slice(5);
                const fd = new FormData();
                fd.append("hash", hash);
                fd.append("password", password);
                if (fileOnly) {
                    fd.append("file_only", "true");
                }

                fetch("/api/posts/delete", { method: "POST", body: fd })
                    .then(function (res) {
                        if (!res.ok) {
                            return res.text().then(function (t) { global.alert(t); });
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
        initReplies,
        cleanup: function () {
            dom("button.delete").off("click", deleteSelectedPosts, false);
        }
    });
}
