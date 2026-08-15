import dom from "../dom.js";
import frame from "../frame.js";
import topicsMap from "../components/topics-map.js";
import renderer from "../components/post-renderer.js";
import createTagHover from "../components/tag-hover.js";
import createReplyBox from "../components/reply-box.js";
import createPostActions from "../components/post-actions.js";
import createPostForm from "../components/post-form.js";

frame.controllers.service = function service(global) {
    const pathParts = global.location.pathname.split("/").filter(Boolean);
    const topic = pathParts[0] || "";
    const isThreadView = (pathParts[1] === "thread");
    const threadHash = pathParts[2] || "";

    // Set topic header
    const headerText = topicsMap[topic] || "Unknown";
    dom(".topic-header").html("/" + topic + "/ - " + headerText);

    let replyBox;
    let postForm;

    // 1. Initialize Tag Hover Component
    const tagHover = createTagHover(global);

    // 2. Initialize Post Form Component
    postForm = createPostForm(global, {
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
    replyBox = createReplyBox(global, function (e) {
        postForm.submitReplyQuick(e);
    });

    // 4. Initialize Post Actions Component
    const postActions = createPostActions(global, {
        onReplyClick: function (threadId, postHash) {
            replyBox.open(threadId, postHash);
        },
        topic
    });

    // 5. SSE Real-Time Event Handlers
    function handleNewThread(data) {
        if (!data || !data.hash || document.getElementById("post-" + data.hash) !== null) {
            return;
        }
        const boardEl = dom(".board").get(0);
        if (boardEl) {
            const html = data.html || renderer.renderThread(data);
            boardEl.insertAdjacentHTML("beforeend", html);
            const threadEl = dom("#post-" + data.hash);
            postActions.bindThreadEvents(threadEl);
            tagHover.bindTags();
            frame.assignHrefs();
        }
    }

    function handleNewReply(data) {
        if (!data || !data.hash || document.getElementById("post-" + data.hash) !== null) {
            return;
        }
        const threadContainer = dom("#post-" + data.thread + " .thread-container");
        if (threadContainer.length() > 0) {
            const html = data.html || renderer.renderReply(data);
            threadContainer.get(0).insertAdjacentHTML("beforeend", html);
            const replyEl = dom("#post-" + data.hash);
            postActions.bindReplyEvents(replyEl);
            tagHover.bindTags();
            postActions.initReplies(data.thread);
        }
    }

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
