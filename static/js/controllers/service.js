"use strict";

import dom from "../dom.js";
import frame from "../frame.js";

const decoder = new TextDecoder("utf-8");

function escapeHTML(str) {
    "use strict";
    if (typeof str !== "string") {
        return "";
    }
    return str
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

function getNode(e, node) {
    "use strict";
    if (node && typeof node.data === "function") {
        return node;
    }
    if (e && e.currentTarget) {
        return dom(e.currentTarget);
    }
    return dom();
}

frame.controllers.service = function service(global, view) {
    "use strict";

    const topicsMap = {
        "a": "Anime & Manga",
        "b": "Random",
        "c": "Anime/Cute",
        "d": "Hentai/Alternative",
        "e": "Ecchi",
        "f": "Flash",
        "g": "Technology",
        "gif": "Adult GIF",
        "h": "Hentai",
        "hr": "High Resolution",
        "k": "Weapons",
        "m": "Mecha",
        "o": "Auto",
        "p": "Photo",
        "r": "Request",
        "s": "Sexy Beautiful Women",
        "t": "Torrents",
        "u": "Yuri",
        "v": "Video Games",
        "vg": "Video Game Generals",
        "vr": "Retro Games",
        "w": "Anime/Wallpapers",
        "wg": "Wallpapers/General",
        "i": "Oekaki",
        "ic": "Artwork/Critique",
        "r9k": "ROBOT9001",
        "sMs": "Shit Moarchan Says",
        "cm": "Cute/Male",
        "hm": "Handsome Men",
        "lgbt": "LGBT",
        "y": "Yaoi",
        "3": "3DCG",
        "adv": "Advice",
        "an": "Animals & Nature",
        "asp": "Alternative Sports",
        "biz": "Business & Finance",
        "cgl": "Cosplay & EGL",
        "ck": "Food & Cooking",
        "co": "Comics & Cartoons",
        "diy": "Do It Yourself",
        "fa": "Fashion",
        "fit": "Fitness",
        "gd": "Graphic Design",
        "hc": "Hardcore",
        "int": "International",
        "jp": "Otaku Culture",
        "lit": "Literature",
        "mlp": "Pony",
        "mu": "Music",
        "n": "Transportation",
        "out": "Outdoors",
        "po": "Papercraft & Origami",
        "pol": "Politically Incorrect",
        "sci": "Science & Math",
        "soc": "Cams & Meetups",
        "sp": "Sports",
        "tg": "Traditional Games",
        "toy": "Toys",
        "trv": "Travel",
        "tv": "Television & Film",
        "vp": "Pokemon",
        "wsg": "Worksafe GIF",
        "x": "Paranormal"
    };

    const replyBoxHeader = dom(".reply-box-header");
    const replyBoxHeaderText = dom(".reply-box-header-text");
    const replyBoxPost = dom(".reply-box-post");
    const replyBox = dom(".reply-box");
    const hashsplit = global.location.hash.split("/");
    let mouseX;
    let mouseY;
    const room = frame.socket.join(hashsplit[1]);

    // Set board header
    dom(".topic-header").html(`/${hashsplit[1]}/ - ${topicsMap[hashsplit[1]] || "Unknown"}`);

    // Blotter toggle
    function toggleBlotter(e) {
        const blotter = dom(".blotter");
        if (blotter.hasClass("hide")) {
            blotter.removeClass("hide");
            dom(".hide-blotter-container").removeClass("hide");
            dom(".show-all-blotter-container").removeClass("hide");
            dom(".show-blotter-container").addClass("hide");
        } else {
            blotter.addClass("hide");
            dom(".hide-blotter-container").addClass("hide");
            dom(".show-all-blotter-container").addClass("hide");
            dom(".show-blotter-container").removeClass("hide");
        }
    }
    dom(".hide-blotter").on("click", toggleBlotter, false);
    dom(".show-blotter").on("click", toggleBlotter, false);

    // New post form
    function showNewPostForm(e) {
        dom(".new-post").addClass("hide");
        dom(".new-post-form").removeClass("hide");
    }
    dom(".new-post").on("click", showNewPostForm, false);

    // Thread show/hide
    function toggleThread(e, node) {
        const targetNode = getNode(e, node);
        dom(`#post-${targetNode.data("post")}`).toggleClass("hide-thread");
    }
    dom(".post-show-hide-thread").on("click", toggleThread, false);

    // Replies toggle
    function toggleReplies(e, node) {
        const targetNode = getNode(e, node);
        const hash = targetNode.data("post");
        const thread = dom(`#post-${hash}`);
        const replies = thread.select(".reply-container");
        const data = {};

        thread.toggleClass("show-replies");
        const summaryEl = thread.select(".post-summary");

        if (!thread.hasClass("show-replies") && replies.length > 5) {
            data.omitted = replies.length - 5;
            data.href = `/${hashsplit[1]}/thread/${hash}`;
            summaryEl.html(`${data.omitted} posts omitted. <span class="blue-text-link" data-href="${data.href}">Click here</span> to view.`);
        } else {
            summaryEl.html("Showing all replies.");
        }
        frame.assignHrefs();
    }
    dom(".post-show-hide-replies").on("click", toggleReplies, false);

    // Hide post
    function hidePost(e, node) {
        const targetNode = getNode(e, node);
        const hash = targetNode.data("post");
        const post = dom(`#post-${hash}`);
        if (post.hasClass("thread")) {
            post.addClass("hide-thread");
        } else if (post.hasClass("reply")) {
            post.addClass("hide-reply");
        }
        dom(".post-options-menu").addClass("hide");
    }
    dom(".hide-post").on("click", hidePost, false);

    // Unhide post
    function unhidePost(e, node) {
        const targetNode = getNode(e, node);
        const hash = targetNode.data("post");
        const post = dom(`#post-${hash}`);
        if (post.hasClass("thread")) {
            post.removeClass("hide-thread");
        } else if (post.hasClass("reply")) {
            post.removeClass("hide-reply");
        }
        dom(".post-options-menu").addClass("hide");
    }
    dom(".unhide-post").on("click", unhidePost, false);

    // Post options menu
    function hidePostOptions(e, node, arg) {
        const targetNode = getNode(e, node);
        if (!targetNode.hasClass("post-options-arrow") && arg) {
            arg.addClass("hide");
        }
    }

    function showPostOptions(e, node) {
        const targetNode = getNode(e, node);
        const hash = targetNode.data("post");
        const menu = dom(`#post-menu-${hash}`);
        dom(".post-options-menu").addClass("hide");
        menu.removeClass("hide");
        setTimeout(() => {
            dom(document.body).once("click", hidePostOptions, false, menu);
        }, 0);
    }
    dom(".post-options-arrow").on("click", showPostOptions, false);

    // Clear forms using existing .each() method
    function clearForms() {
        dom("#reply-box-name, #reply-box-options, #reply-box-comment, #reply-box-file").each(function (el) {
            if ("value" in el) {
                el.value = "";
            }
        });
        dom("#new-post-name, #new-post-subject, #new-post-options, #new-post-comment, #new-post-file").each(function (el) {
            if ("value" in el) {
                el.value = "";
            }
        });
        dom(".reply-box-close").each(function (el) {
            el.click();
        });
    }

    // Post new thread
    function postThread(e) {
        const nameInput = dom("#new-post-name").get(0);
        const subjectInput = dom("#new-post-subject").get(0);
        const optionsInput = dom("#new-post-options").get(0);
        const commentInput = dom("#new-post-comment").get(0);
        const fileInput = dom("#new-post-file").get(0);

        const schema = {
            type: "thread",
            topic: hashsplit[1],
            name: escapeHTML(nameInput?.value || "Anonymous"),
            subject: escapeHTML(subjectInput?.value || ""),
            options: escapeHTML(optionsInput?.value || ""),
            comment: escapeHTML(commentInput?.value || "").replace(/\r?\n/g, "<br />"),
            replies: {},
            taggedBy: [],
            tagging: []
        };

        const files = fileInput?.files || [];
        const reader = new FileReader();

        function fileLoaded(s) {
            return function (e) {
                s.file = e.target.result;
                room.send("new-thread", JSON.stringify(s));
                clearForms();
            };
        }

        if (files.length > 0) {
            const file = files[0];
            schema.file_name = escapeHTML(file.name);
            schema.file_mime = file.type;
            schema.file_size = (file.size / 1024).toFixed(1);
            reader.onload = fileLoaded(schema);
            reader.readAsDataURL(file);
        } else {
            global.alert("Must add a file to post a new thread.");
        }
    }

    // Post reply
    function postReply(e) {
        const replyboxVisible = !replyBox.hasClass("hide");
        const activeThreadList = replyBox.data("thread");
        const thread = replyboxVisible && activeThreadList ? activeThreadList[0] : hashsplit[3];

        if (!thread) {
            return global.alert("Unable to locate target thread ID for reply.");
        }

        const nameInput = replyboxVisible ? dom("#reply-box-name").get(0) : dom("#new-post-name").get(0);
        const optionsInput = replyboxVisible ? dom("#reply-box-options").get(0) : dom("#new-post-options").get(0);
        const commentInput = replyboxVisible ? dom("#reply-box-comment").get(0) : dom("#new-post-comment").get(0);
        const fileInput = replyboxVisible ? dom("#reply-box-file").get(0) : dom("#new-post-file").get(0);

        const rawComment = commentInput?.value || "";
        if (!rawComment) {
            return global.alert("Must write a comment to post a reply.");
        }

        const schema = {
            type: "reply",
            topic: hashsplit[1],
            thread: thread,
            name: escapeHTML(nameInput?.value || "Anonymous"),
            options: escapeHTML(optionsInput?.value || ""),
            taggedBy: [],
            tagging: []
        };

        schema.comment = escapeHTML(rawComment)
            .replace(/\r?\n/g, "<br />")
            .replace(/&gt;&gt;(\w+)/g, function (match, postHash) {
                schema.tagging.push(postHash);
                return `<span class="post-tag blue-text-link" data-tag="${postHash}">${match}</span>`;
            });

        const files = fileInput?.files || [];
        const reader = new FileReader();

        function fileLoaded(s) {
            return function (e) {
                s.file = e.target.result;
                room.send("new-reply", JSON.stringify(s));
                clearForms();
            };
        }

        if (files.length > 0) {
            const file = files[0];
            schema.file_name = escapeHTML(file.name);
            schema.file_mime = file.type;
            schema.file_size = (file.size / 1024).toFixed(1);
            reader.onload = fileLoaded(schema);
            reader.readAsDataURL(file);
        } else {
            room.send("new-reply", JSON.stringify(schema));
            clearForms();
        }
    }

    // Attach post button
    if (view === "thread") {
        dom("#new-post-button").on("click", postReply, false);
    } else {
        dom("#new-post-button").on("click", postThread, false);
    }

    // Reply box drag
    function dragging(e) {
        const top = parseInt(replyBox.css("top")[0], 10) + e.clientY - mouseY + "px";
        const left = parseInt(replyBox.css("left")[0], 10) - mouseX + e.clientX + "px";
        replyBox.css("top", top).css("left", left);
        mouseX = e.clientX;
        mouseY = e.clientY;
    }

    function stopDrag(e) {
        dom(document.body).off("mousemove", dragging, false);
    }

    function startDrag(e) {
        mouseX = e.clientX;
        mouseY = e.clientY;
        dom(document.body).on("mousemove", dragging, false);
        dom(document.body).once("mouseup", stopDrag, false);
    }

    function closeReplyBox(e) {
        replyBox.addClass("hide");
    }

    function openReplyBox(e, node) {
        const targetNode = getNode(e, node);
        const thread = targetNode.data("thread");
        const post = targetNode.html()[0] || "";

        replyBox.data("thread", thread);
        replyBoxHeaderText.html(thread).attr("title", thread);
        replyBoxPost.off("click", postReply, false);
        replyBoxPost.on("click", postReply, false);
        dom(".reply-box-close").on("click", closeReplyBox, false);
        const commentInput = dom("#reply-box-comment").get(0);
        if (commentInput && "value" in commentInput) {
            commentInput.value = `>>${post}`;
        }
        replyBox.removeClass("hide");
    }

    replyBoxHeader.on("mousedown", startDrag, false);
    dom(".post-reply-to").on("click", openReplyBox, false);

    // Initialize thread replies visibility
    function initReplies(hash) {
        const thread = dom(`#post-${hash}`);
        const replies = thread.select(".reply-container");
        const summaryEl = thread.select(".post-summary");
        const data = {};

        if (replies.length > 0) {
            if (replies.length > 5) {
                data.omitted = replies.length - 5;
                data.href = `/${hashsplit[1]}/thread/${hash}`;
                thread.addClass("show-summary");
                summaryEl.html(`${data.omitted} posts omitted. <span class="blue-text-link" data-href="${data.href}">Click here</span> to view.`);
            } else {
                thread.addClass("show-replies");
            }
        }
        frame.assignHrefs();
    }

    if (view === "topic") {
        dom(".thread").each(node => {
            if (node && node.id) {
                initReplies(node.id.slice(5));
            }
        });
    }

    // Scroll to tagged post dynamically
    function goToTaggedPost(e) {
        const targetNode = dom(e.currentTarget);
        const tagList = targetNode.data("tag");
        const tag = tagList ? tagList[0] : null;
        if (!tag) {
            return;
        }
        const tagged = dom(`#post-${tag}`);
        if (tagged.length() === 0) {
            return;
        }
        dom(".highlight").removeClass("highlight");
        const el = tagged.get(0);
        if (el) {
            el.scrollIntoView({ behavior: "smooth", block: "center" });
        }
        if (!tagged.hasClass("thread")) {
            tagged.addClass("highlight");
        }
    }

    // Hover over >>link dynamically
    function hoverOutTag(e) {
        const targetNode = dom(e.currentTarget);
        const tagList = targetNode.data("tag");
        const tag = tagList ? tagList[0] : null;
        if (tag) {
            const tagged = dom(`#post-${tag}`);
            tagged.removeClass("highlight-hover");
        }
        dom(".tag-hover-clone").remove();
    }

    function hoverOverTag(e) {
        const targetNode = dom(e.currentTarget);
        const tagList = targetNode.data("tag");
        const tag = tagList ? tagList[0] : null;
        if (!tag) {
            return;
        }
        const tagged = dom(`#post-${tag}`);
        const el = tagged.get(0);
        if (!el) {
            return;
        }

        const rect = el.getBoundingClientRect();
        const inview = (
            rect.top >= 0 &&
            rect.left >= 0 &&
            rect.bottom <= (global.innerHeight || document.documentElement.clientHeight) &&
            rect.right <= (global.innerWidth || document.documentElement.clientWidth)
        );

        if (inview) {
            tagged.addClass("highlight-hover");
            targetNode.once("mouseout", hoverOutTag, false);
        } else {
            dom(".tag-hover-clone").remove();

            const cloneDom = tagged.clone(true);
            cloneDom.addClass("tag-hover-clone")
                .css("position", "absolute")
                .css("top", (e.pageY - rect.height - 20) + "px")
                .css("left", (e.pageX + 20) + "px")
                .css("width", rect.width + "px")
                .css("height", rect.height + "px")
                .css("box-shadow", "1px 1px 6px 0 rgba(0, 0, 0, 0.6)");
            const cloneEl = cloneDom.get(0);
            if (cloneEl) {
                document.body.appendChild(cloneEl);
            }
            targetNode.once("mouseout", hoverOutTag, false);
        }
    }

    function bindPostTags() {
        dom(".post-tag").each(node => {
            const tagDom = dom(node);
            tagDom.off("mouseover", hoverOverTag, false);
            tagDom.off("click", goToTaggedPost, false);
            tagDom.on("mouseover", hoverOverTag, false);
            tagDom.on("click", goToTaggedPost, false);
        });
    }

    bindPostTags();

    // === Real-time updates ===

    function addThread(buffer) {
        const rawData = decoder.decode(buffer);
        let data;
        try {
            data = JSON.parse(rawData);
        } catch (err) {
            return;
        }

        if (!data.hash || document.getElementById(`post-${data.hash}`) !== null) return;

        // Sanitized fallbacks for missing/untrusted fields
        const subject = escapeHTML(data.subject || "");
        const name = escapeHTML(data.name || "Anonymous");
        const file_name = escapeHTML(data.file_name || "");
        const file_size = data.file_size ? (parseInt(data.file_size, 10) / 1024).toFixed(1) : "0";
        const file_dimensions = escapeHTML(data.file_dimensions || "???x???");
        const comment = data.comment || "";
        const timestamp = escapeHTML(data.timestamp || new Date().toISOString());

        const htmlString = `
<div id="post-${data.hash}" class="thread">
    <div class="post-show-hide-icons op">
        <img class="post-show-hide-thread plus" data-post="${data.hash}" src="/static/images/show-hide-thread-plus-red.png" alt="Plus" title="Plus" />
        <img class="post-show-hide-thread minus" data-post="${data.hash}" src="/static/images/show-hide-thread-minus-red.png" alt="Minus" title="Minus" />
    </div>
    <div class="post-image-metadata op">
        File: <a class="post-image-link blue-text-link op" href="/static/images/uploads/${file_name}" alt="${file_name}" title="${file_name}" target="_blank">${file_name}</a>
        <span class="post-image-dimensions op">(${file_size} KB, ${file_dimensions})</span>
    </div>
    <a class="post-image-container op" href="/static/images/uploads/${file_name}" target="_blank">
        <img class="post-image op" src="/static/images/uploads/${file_name}" alt="${file_name}" title="${file_name}" />
    </a>
    <div class="post-header op">
        <input class="post-checkbox op" type="checkbox" />
        <span class="post-subject op">${subject}</span>
        <span class="post-username op">${name}</span>
        <span class="post-date op">${timestamp}</span>
        <span class="post-link-to red-text-link op" title="Link to this post" data-href="/${data.topic}/thread/${data.hash}">No.</span>
        <span class="post-reply-to red-text-link op" title="Reply to this post" data-thread="${data.hash}">${data.hash}</span>
        <img class="post-thumbtack op" src="/static/images/thumbtack.gif" alt="Sticky" title="Sticky" />
        <img class="post-lock op" src="/static/images/lock.gif" alt="Closed" title="Closed" />
        <span class="post-reply-to-text op">[<span class="reply-link blue-text-link" data-href="/${data.topic}/thread/${data.hash}">Reply</span>]</span>
        <div class="post-options op" title="Post menu">
            <span class="post-options-arrow op" data-post="${data.hash}"></span>
            <ul id="post-menu-${data.hash}" class="post-options-menu hide op">
                <li class="report-post op" data-post="${data.hash}">Report Thread</li>
                <li class="hide-post op" data-post="${data.hash}">Hide Thread</li>
                <li class="unhide-post op" data-post="${data.hash}">Unhide Thread</li>
                <li class="image-search op" data-post="${data.hash}">Image Search &gt;&gt;</li>
            </ul>
        </div>
    </div>
    <div class="thread-container">
        <p class="post-content op">${comment}</p>
        <div class="post-summary-container">
            <div class="post-show-hide-icons replies">
                <img class="post-show-hide-replies plus" data-post="${data.hash}" src="/static/images/show-hide-thread-plus-red.png" alt="Plus" title="Plus" />
                <img class="post-show-hide-replies minus" data-post="${data.hash}" src="/static/images/show-hide-thread-minus-red.png" alt="Minus" title="Minus" />
            </div>
            <p class="post-summary">0 posts omitted. <span class="blue-text-link" data-href="/${data.topic}/thread/${data.hash}">Click here</span> to view.</p>
        </div>
    </div>
    <div class="spacer"></div>
</div>`;

        const boardEl = dom(".board").get(0);
        if (boardEl) {
            boardEl.insertAdjacentHTML("beforeend", htmlString);
        }
        const threadEl = dom(`#post-${data.hash}`);
        threadEl.selectAll(".post-show-hide-thread").on("click", toggleThread, false);
        threadEl.selectAll(".post-show-hide-replies").on("click", toggleReplies, false);
        threadEl.selectAll(".hide-post").on("click", hidePost, false);
        threadEl.selectAll(".unhide-post").on("click", unhidePost, false);
        threadEl.selectAll(".post-options-arrow").on("click", showPostOptions, false);
        threadEl.selectAll(".post-reply-to").on("click", openReplyBox, false);
        bindPostTags();
        frame.assignHrefs();
    }

    function addReply(buffer) {
        const rawData = decoder.decode(buffer);
        let data;
        try {
            data = JSON.parse(rawData);
        } catch (err) {
            return;
        }

        if (!data.hash || document.getElementById(`post-${data.hash}`) !== null) return;

        // Process tagging
        data.tagging.forEach(tag => {
            const isOp = tag === data.thread;
            const opClass = isOp ? " op" : "";
            const tagEl = `<span class="post-tag blue-text-link${opClass}" data-tag="${data.hash}">&gt;&gt;${data.hash}</span>`;
            const header = dom(`#post-${tag} .post-header`);
            if (header.length() > 0) {
                header.get(0).insertAdjacentHTML("beforeend", tagEl);
            }
        });

        // Build HTML
        let html = `
<div class="reply-container">
    <div class="reply-wrapper">
        <span class="post-side-arrows">&gt;&gt;</span>
        <div id="post-${data.hash}" class="reply">
            <div class="post-header">
                <input class="post-checkbox" type="checkbox" />
                <span class="post-username">${escapeHTML(data.name || "Anonymous")}</span>
                <span class="post-date">${escapeHTML(data.timestamp || new Date().toISOString())}</span>
                <span class="post-link-to red-text-link" title="Link to this post">No.</span>
                <span class="post-reply-to red-text-link" title="Reply to this post" data-thread="${data.thread}">${data.hash}</span>
                <div class="post-options" title="Post menu">
                    <span class="post-options-arrow" data-post="${data.hash}"></span>
                    <ul id="post-menu-${data.hash}" class="post-options-menu hide">`;

        // Report/Hide/Unhide always
        html += `
                        <li class="report-post" data-post="${data.hash}">Report Post</li>
                        <li class="hide-post" data-post="${data.hash}">Hide Post</li>
                        <li class="unhide-post" data-post="${data.hash}">Unhide Post</li>`;

        // Image search if image exists
        if (data.file_name) {
            html += `<li class="image-search" data-post="${data.hash}">Image Search &gt;&gt;</li>`;
        }

        html += `
                    </ul>
                </div>
            </div>`;

        // Image block if exists
        if (data.file_name) {
            const fileNameEscaped = escapeHTML(data.file_name);
            const file_size_kb = data.file_size ? (parseInt(data.file_size, 10) / 1024).toFixed(1) : "0";
            const dims = escapeHTML(data.file_dimensions || "???x???");
            html += `
            <div class="post-image-metadata">
                File: <a class="post-image-link blue-text-link" href="/static/images/uploads/${fileNameEscaped}" alt="${fileNameEscaped}" title="${fileNameEscaped}" target="_blank">${fileNameEscaped}</a>
                <span class="post-image-dimensions">(${file_size_kb} KB, ${dims})</span>
            </div>
            <a class="post-image-container" href="/static/images/uploads/${fileNameEscaped}" target="_blank">
                <img class="post-image" src="/static/images/uploads/${fileNameEscaped}" title="${fileNameEscaped}" alt="${fileNameEscaped}" />
            </a>`;
        }

        // Comment
        html += `
            <p class="post-content">${data.comment || ""}</p>
        </div>
    </div>
</div>`;

        const threadContainer = dom(`#post-${data.thread} .thread-container`);
        if (threadContainer.length() > 0) {
            threadContainer.get(0).insertAdjacentHTML("beforeend", html);
        }

        const replyEl = dom(`#post-${data.hash}`);
        replyEl.selectAll(".hide-post").on("click", hidePost, false);
        replyEl.selectAll(".unhide-post").on("click", unhidePost, false);
        replyEl.selectAll(".post-options-arrow").on("click", showPostOptions, false);
        replyEl.selectAll(".post-reply-to").on("click", openReplyBox, false);

        bindPostTags();
        initReplies(data.thread);
    }

    room.on("new-reply", addReply);
    room.on("new-thread", addThread);
};
