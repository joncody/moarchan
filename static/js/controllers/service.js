import dom from "../dom.js";
import frame from "../frame.js";

const decoder = new TextDecoder("utf-8");

function escapeHTML(str) {
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
    if (node && typeof node.data === "function") {
        return node;
    }
    if (e && e.currentTarget) {
        return dom(e.currentTarget);
    }
    return dom();
}

frame.controllers.service = function service(global, view) {
    const topicsMap = {
        "3": "3DCG",
        "a": "Anime & Manga",
        "adv": "Advice",
        "an": "Animals & Nature",
        "asp": "Alternative Sports",
        "b": "Random",
        "biz": "Business & Finance",
        "c": "Anime/Cute",
        "cgl": "Cosplay & EGL",
        "ck": "Food & Cooking",
        "cm": "Cute/Male",
        "co": "Comics & Cartoons",
        "d": "Hentai/Alternative",
        "diy": "Do It Yourself",
        "e": "Ecchi",
        "f": "Flash",
        "fa": "Fashion",
        "fit": "Fitness",
        "g": "Technology",
        "gd": "Graphic Design",
        "gif": "Adult GIF",
        "h": "Hentai",
        "hc": "Hardcore",
        "hm": "Handsome Men",
        "hr": "High Resolution",
        "i": "Oekaki",
        "ic": "Artwork/Critique",
        "int": "International",
        "jp": "Otaku Culture",
        "k": "Weapons",
        "lgbt": "LGBT",
        "lit": "Literature",
        "m": "Mecha",
        "mlp": "Pony",
        "mu": "Music",
        "n": "Transportation",
        "o": "Auto",
        "out": "Outdoors",
        "p": "Photo",
        "po": "Papercraft & Origami",
        "pol": "Politically Incorrect",
        "r": "Request",
        "r9k": "ROBOT9001",
        "s": "Sexy Beautiful Women",
        "sMs": "Shit Moarchan Says",
        "sci": "Science & Math",
        "soc": "Cams & Meetups",
        "sp": "Sports",
        "t": "Torrents",
        "tg": "Traditional Games",
        "toy": "Toys",
        "trv": "Travel",
        "tv": "Television & Film",
        "u": "Yuri",
        "v": "Video Games",
        "vg": "Video Game Generals",
        "vp": "Pokemon",
        "vr": "Retro Games",
        "w": "Anime/Wallpapers",
        "wg": "Wallpapers/General",
        "wsg": "Worksafe GIF",
        "x": "Paranormal",
        "y": "Yaoi"
    };

    const replyBoxHeader = dom(".reply-box-header");
    const replyBoxHeaderText = dom(".reply-box-header-text");
    const replyBoxPost = dom(".reply-box-post");
    const replyBox = dom(".reply-box");
    const hashsplit = global.location.hash.split("/");
    const isThreadView = (hashsplit[2] === "thread");
    let mouseX;
    let mouseY;
    const room = frame.socket.join(hashsplit[1]);

    // Prevent default form submission page navigation
    dom(".new-post-form").on("submit", function (e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
    });

    const headerText = topicsMap[hashsplit[1]] || "Unknown";
    dom(".topic-header").html("/" + hashsplit[1] + "/ - " + headerText);

    function toggleBlotter(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
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

    function showNewPostForm(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        dom(".new-post").addClass("hide");
        dom(".new-post-form").removeClass("hide");
    }
    dom(".new-post").on("click", showNewPostForm, false);

    function toggleThread(e, node) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const targetNode = getNode(e, node);
        dom("#post-" + targetNode.data("post")).toggleClass("hide-thread");
    }
    dom(".post-show-hide-thread").on("click", toggleThread, false);

    function toggleReplies(e, node) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const targetNode = getNode(e, node);
        const hash = targetNode.data("post");
        const thread = dom("#post-" + hash);
        const replies = thread.select(".reply-container");

        thread.toggleClass("show-replies");
        const summaryEl = thread.select(".post-summary");

        if (thread.hasClass("show-replies") === false && replies.length() > 5) {
            const omitted = replies.length() - 5;
            const href = "/" + hashsplit[1] + "/thread/" + hash;
            summaryEl.html(
                omitted +
                " posts omitted. <span class=\"blue-text-link\" data-href=\"" +
                href +
                "\">Click here</span> to view."
            );
        } else {
            summaryEl.html("Showing all replies.");
        }
        frame.assignHrefs();
    }
    dom(".post-show-hide-replies").on("click", toggleReplies, false);

    function hidePost(e, node) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const targetNode = getNode(e, node);
        const hash = targetNode.data("post");
        const post = dom("#post-" + hash);
        if (post.hasClass("thread")) {
            post.addClass("hide-thread");
        } else if (post.hasClass("reply")) {
            post.addClass("hide-reply");
        }
        dom(".post-options-menu").addClass("hide");
    }
    dom(".hide-post").on("click", hidePost, false);

    function unhidePost(e, node) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const targetNode = getNode(e, node);
        const hash = targetNode.data("post");
        const post = dom("#post-" + hash);
        if (post.hasClass("thread")) {
            post.removeClass("hide-thread");
        } else if (post.hasClass("reply")) {
            post.removeClass("hide-reply");
        }
        dom(".post-options-menu").addClass("hide");
    }
    dom(".unhide-post").on("click", unhidePost, false);

    function hidePostOptions(e, node, arg) {
        const targetNode = getNode(e, node);
        if (targetNode.hasClass("post-options-arrow") === false && arg) {
            arg.addClass("hide");
        }
    }

    function showPostOptions(e, node) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const targetNode = getNode(e, node);
        const hash = targetNode.data("post");
        const menu = dom("#post-menu-" + hash);
        dom(".post-options-menu").addClass("hide");
        menu.removeClass("hide");
        setTimeout(function () {
            dom(document.body).once("click", hidePostOptions, false, menu);
        }, 0);
    }
    dom(".post-options-arrow").on("click", showPostOptions, false);

    function clearForms() {
        const fields = [
            "#reply-box-name",
            "#reply-box-options",
            "#reply-box-comment",
            "#reply-box-file",
            "#new-post-name",
            "#new-post-subject",
            "#new-post-options",
            "#new-post-comment",
            "#new-post-file"
        ];
        fields.forEach(function (selector) {
            dom(selector).each(function (el) {
                if ("value" in el) {
                    el.value = "";
                }
            });
        });
        dom(".reply-box-close").each(function (el) {
            el.click();
        });
    }

    function postThread(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const nameInput = dom("#new-post-name").get(0);
        const subjectInput = dom("#new-post-subject").get(0);
        const optionsInput = dom("#new-post-options").get(0);
        const commentInput = dom("#new-post-comment").get(0);
        const fileInput = dom("#new-post-file").get(0);

        const files = (fileInput && fileInput.files) || [];

        if (files.length === 0) {
            return global.alert("Must add a file to post a new thread.");
        }

        const file = files[0];

        if (file.size > 8192 * 1024) {
            return global.alert("File size exceeds maximum allowed limit of 8192 KB (8 MB).");
        }

        const rawName = (nameInput && nameInput.value) || "Anonymous";
        const rawSub = (subjectInput && subjectInput.value) || "";
        const rawOpt = (optionsInput && optionsInput.value) || "";
        const rawCom = (commentInput && commentInput.value) || "";

        const schema = {
            comment: escapeHTML(rawCom).replace(/\r?\n/g, "<br />"),
            file_mime: file.type,
            file_name: escapeHTML(file.name),
            name: escapeHTML(rawName),
            options: escapeHTML(rawOpt),
            replies: {},
            subject: escapeHTML(rawSub),
            taggedBy: [],
            tagging: [],
            topic: hashsplit[1],
            type: "thread"
        };

        const reader = new FileReader();
        reader.onload = function (evt) {
            schema.file = evt.target.result;
            room.send("new-thread", JSON.stringify(schema));
            clearForms();
        };
        reader.readAsDataURL(file);
    }

    function postReply(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        const replyboxVisible = dom(".reply-box").hasClass("hide") === false;
        const activeThreadList = replyBox.data("thread");
        const thread = (
            (replyboxVisible && activeThreadList)
            ? activeThreadList[0]
            : hashsplit[3]
        );

        if (!thread) {
            return global.alert("Unable to locate target thread ID for reply.");
        }

        const nameSel = replyboxVisible ? "#reply-box-name" : "#new-post-name";
        const optSel = replyboxVisible ? "#reply-box-options" : "#new-post-options";
        const comSel = replyboxVisible ? "#reply-box-comment" : "#new-post-comment";
        const fileSel = replyboxVisible ? "#reply-box-file" : "#new-post-file";

        const nameInput = dom(nameSel).get(0);
        const optionsInput = dom(optSel).get(0);
        const commentInput = dom(comSel).get(0);
        const fileInput = dom(fileSel).get(0);

        const rawComment = (commentInput && commentInput.value) || "";
        if (!rawComment) {
            return global.alert("Must write a comment to post a reply.");
        }

        const files = (fileInput && fileInput.files) || [];

        if (files.length > 0 && files[0].size > 8192 * 1024) {
            return global.alert("File size exceeds maximum allowed limit of 8192 KB (8 MB).");
        }

        const rawName = (nameInput && nameInput.value) || "Anonymous";
        const rawOpt = (optionsInput && optionsInput.value) || "";

        const schema = {
            name: escapeHTML(rawName),
            options: escapeHTML(rawOpt),
            taggedBy: [],
            tagging: [],
            thread,
            topic: hashsplit[1],
            type: "reply"
        };

        schema.comment = escapeHTML(rawComment)
            .replace(/\r?\n/g, "<br />")
            .replace(/&gt;&gt;(\w+)/g, function (match, postHash) {
                schema.tagging.push(postHash);
                return (
                    "<span class=\"post-tag blue-text-link\" data-tag=\"" +
                    postHash +
                    "\">" +
                    match +
                    "</span>"
                );
            });

        function submitReply() {
            room.send("new-reply", JSON.stringify(schema));
            clearForms();
        }

        if (files.length > 0) {
            const file = files[0];
            schema.file_name = escapeHTML(file.name);
            schema.file_mime = file.type;

            const reader = new FileReader();
            reader.onload = function (evt) {
                schema.file = evt.target.result;
                submitReply();
            };
            reader.readAsDataURL(file);
        } else {
            submitReply();
        }
    }

    if (isThreadView) {
        dom("#new-post-button").on("click", postReply, false);
    } else {
        dom("#new-post-button").on("click", postThread, false);
    }

    function dragging(e) {
        const topPx = (
            parseInt(replyBox.css("top")[0], 10) +
            e.clientY -
            mouseY
        ) + "px";
        const leftPx = (
            parseInt(replyBox.css("left")[0], 10) -
            mouseX +
            e.clientX
        ) + "px";
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

    function closeReplyBox(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        replyBox.addClass("hide");
    }

    function openReplyBox(e, node) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
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
            commentInput.value = ">>" + post;
        }
        replyBox.removeClass("hide");
    }

    replyBoxHeader.on("mousedown", startDrag, false);
    dom(".post-reply-to").on("click", openReplyBox, false);

    function initReplies(hash) {
        const thread = dom("#post-" + hash);
        const replies = thread.select(".reply-container");
        const summaryEl = thread.select(".post-summary");

        if (replies.length() > 0) {
            if (replies.length() > 5) {
                const omitted = replies.length() - 5;
                const href = "/" + hashsplit[1] + "/thread/" + hash;
                thread.addClass("show-summary");
                summaryEl.html(
                    omitted +
                    " posts omitted. <span class=\"blue-text-link\" data-href=\"" +
                    href +
                    "\">Click here</span> to view."
                );
            } else {
                thread.addClass("show-replies");
            }
        }
        frame.assignHrefs();
    }

    if (!isThreadView) {
        dom(".thread").each(function (node) {
            if (node && node.id) {
                initReplies(node.id.slice(5));
            }
        });
    }

    function goToTaggedPost(e) {
        const targetNode = dom(e.currentTarget);
        const tagList = targetNode.data("tag");
        const tag = (tagList) ? tagList[0] : null;
        if (!tag) {
            return;
        }
        const tagged = dom("#post-" + tag);
        if (tagged.length() === 0) {
            return;
        }
        dom(".highlight").removeClass("highlight");
        const el = tagged.get(0);
        if (el) {
            el.scrollIntoView({
                behavior: "smooth",
                block: "center"
            });
        }
        if (tagged.hasClass("thread") === false) {
            tagged.addClass("highlight");
        }
    }

    function hoverOutTag(e) {
        const targetNode = dom(e.currentTarget);
        const tagList = targetNode.data("tag");
        const tag = (tagList) ? tagList[0] : null;
        if (tag) {
            const tagged = dom("#post-" + tag);
            tagged.removeClass("highlight-hover");
        }
        dom(".tag-hover-clone").remove();
    }

    function hoverOverTag(e) {
        const targetNode = dom(e.currentTarget);
        const tagList = targetNode.data("tag");
        const tag = (tagList) ? tagList[0] : null;
        if (!tag) {
            return;
        }
        const tagged = dom("#post-" + tag);
        const el = tagged.get(0);
        if (!el) {
            return;
        }

        const rect = el.getBoundingClientRect();
        const vHeight = global.innerHeight || document.documentElement.clientHeight;
        const vWidth = global.innerWidth || document.documentElement.clientWidth;
        const inview = (
            rect.top >= 0 &&
            rect.left >= 0 &&
            rect.bottom <= vHeight &&
            rect.right <= vWidth
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
        dom(".post-tag").each(function (node) {
            const tagDom = dom(node);
            tagDom.off("mouseover", hoverOverTag, false);
            tagDom.off("click", goToTaggedPost, false);
            tagDom.on("mouseover", hoverOverTag, false);
            tagDom.on("click", goToTaggedPost, false);
        });
    }

    bindPostTags();

    function addThread(buffer) {
        const rawData = decoder.decode(buffer);
        let data;
        try {
            data = JSON.parse(rawData);
        } catch (err) {
            return;
        }

        if (!data.hash || document.getElementById("post-" + data.hash) !== null) {
            return;
        }

        const subject = escapeHTML(data.subject || "");
        const name = escapeHTML(data.name || "Anonymous");
        const file_name = escapeHTML(data.file_name || "");
        const file_size = data.file_size || "0";
        const file_dimensions = escapeHTML(data.file_dimensions || "???x???");
        const comment = data.comment || "";
        const timestamp = escapeHTML(data.timestamp || new Date().toISOString());

        const htmlString = [
            "<div id=\"post-", data.hash, "\" class=\"thread\">",
            "<div class=\"post-show-hide-icons op\">",
            "<img class=\"post-show-hide-thread plus\" data-post=\"", data.hash, "\" src=\"/static/images/show-hide-thread-plus-red.png\" alt=\"Plus\" title=\"Plus\" />",
            "<img class=\"post-show-hide-thread minus\" data-post=\"", data.hash, "\" src=\"/static/images/show-hide-thread-minus-red.png\" alt=\"Minus\" title=\"Minus\" />",
            "</div>",
            "<div class=\"post-image-metadata op\">",
            "File: <a class=\"post-image-link blue-text-link op\" href=\"/static/images/uploads/", file_name, "\" alt=\"", file_name, "\" title=\"", file_name, "\" target=\"_blank\">", file_name, "</a>",
            "<span class=\"post-image-dimensions op\">(", file_size, " KB, ", file_dimensions, ")</span>",
            "</div>",
            "<a class=\"post-image-container op\" href=\"/static/images/uploads/", file_name, "\" target=\"_blank\">",
            "<img class=\"post-image op\" src=\"/static/images/uploads/", file_name, "\" alt=\"", file_name, "\" title=\"", file_name, "\" />",
            "</a>",
            "<div class=\"post-header op\">",
            "<input class=\"post-checkbox op\" type=\"checkbox\" />",
            "<span class=\"post-subject op\">", subject, "</span>",
            "<span class=\"post-username op\">", name, "</span>",
            "<span class=\"post-date op\">", timestamp, "</span>",
            "<span class=\"post-link-to red-text-link op\" title=\"Link to this post\" data-href=\"/", data.topic, "/thread/", data.hash, "\">No.</span>",
            "<span class=\"post-reply-to red-text-link op\" title=\"Reply to this post\" data-thread=\"", data.hash, "\">", data.hash, "</span>",
            "<img class=\"post-thumbtack op\" src=\"/static/images/thumbtack.gif\" alt=\"Sticky\" title=\"Sticky\" />",
            "<img class=\"post-lock op\" src=\"/static/images/lock.gif\" alt=\"Closed\" title=\"Closed\" />",
            "<span class=\"post-reply-to-text op\">[<span class=\"reply-link blue-text-link\" data-href=\"/", data.topic, "/thread/", data.hash, "\">Reply</span>]</span>",
            "<div class=\"post-options op\" title=\"Post menu\">",
            "<span class=\"post-options-arrow op\" data-post=\"", data.hash, "\"></span>",
            "<ul id=\"post-menu-", data.hash, "\" class=\"post-options-menu hide op\">",
            "<li class=\"report-post op\" data-post=\"", data.hash, "\">Report Thread</li>",
            "<li class=\"hide-post op\" data-post=\"", data.hash, "\">Hide Thread</li>",
            "<li class=\"unhide-post op\" data-post=\"", data.hash, "\">Unhide Thread</li>",
            "<li class=\"image-search op\" data-post=\"", data.hash, "\">Image Search &gt;&gt;</li>",
            "</ul>",
            "</div>",
            "</div>",
            "<div class=\"thread-container\">",
            "<p class=\"post-content op\">", comment, "</p>",
            "<div class=\"post-summary-container\">",
            "<div class=\"post-show-hide-icons replies\">",
            "<img class=\"post-show-hide-replies plus\" data-post=\"", data.hash, "\" src=\"/static/images/show-hide-thread-plus-red.png\" alt=\"Plus\" title=\"Plus\" />",
            "<img class=\"post-show-hide-replies minus\" data-post=\"", data.hash, "\" src=\"/static/images/show-hide-thread-minus-red.png\" alt=\"Minus\" title=\"Minus\" />",
            "</div>",
            "<p class=\"post-summary\">0 posts omitted. <span class=\"blue-text-link\" data-href=\"/", data.topic, "/thread/", data.hash, "\">Click here</span> to view.</p>",
            "</div>",
            "</div>",
            "<div class=\"spacer\"></div>",
            "</div>"
        ].join("");

        const boardEl = dom(".board").get(0);
        if (boardEl) {
            boardEl.insertAdjacentHTML("beforeend", htmlString);
        }
        const threadEl = dom("#post-" + data.hash);
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

        if (!data.hash || document.getElementById("post-" + data.hash) !== null) {
            return;
        }

        if (Array.isArray(data.tagging)) {
            data.tagging.forEach(function (tag) {
                const isOp = (tag === data.thread);
                const opClass = isOp ? " op" : "";
                const tagEl = (
                    "<span class=\"post-tag blue-text-link" +
                    opClass +
                    "\" data-tag=\"" +
                    data.hash +
                    "\">&gt;&gt;" +
                    data.hash +
                    "</span>"
                );
                const header = dom("#post-" + tag + " .post-header");
                if (header.length() > 0) {
                    header.get(0).insertAdjacentHTML("beforeend", tagEl);
                }
            });
        }

        let fileNameEscaped = "";
        let fileBlock = "";
        if (data.file_name) {
            fileNameEscaped = escapeHTML(data.file_name);
            const fileSizeKb = data.file_size || "0";
            const dims = escapeHTML(data.file_dimensions || "???x???");
            fileBlock = [
                "<div class=\"post-image-metadata\">",
                "File: <a class=\"post-image-link blue-text-link\" href=\"/static/images/uploads/", fileNameEscaped, "\" alt=\"", fileNameEscaped, "\" title=\"", fileNameEscaped, "\" target=\"_blank\">", fileNameEscaped, "</a>",
                "<span class=\"post-image-dimensions\">(", fileSizeKb, " KB, ", dims, ")</span>",
                "</div>",
                "<a class=\"post-image-container\" href=\"/static/images/uploads/", fileNameEscaped, "\" target=\"_blank\">",
                "<img class=\"post-image\" src=\"/static/images/uploads/", fileNameEscaped, "\" title=\"", fileNameEscaped, "\" alt=\"", fileNameEscaped, "\" />",
                "</a>"
            ].join("");
        }

        let imgSearchBlock = "";
        if (data.file_name) {
            imgSearchBlock = "<li class=\"image-search\" data-post=\"" + data.hash + "\">Image Search &gt;&gt;</li>";
        }

        const replyHtml = [
            "<div class=\"reply-container\">",
            "<div class=\"reply-wrapper\">",
            "<span class=\"post-side-arrows\">&gt;&gt;</span>",
            "<div id=\"post-", data.hash, "\" class=\"reply\">",
            "<div class=\"post-header\">",
            "<input class=\"post-checkbox\" type=\"checkbox\" />",
            "<span class=\"post-username\">", escapeHTML(data.name || "Anonymous"), "</span>",
            "<span class=\"post-date\">", escapeHTML(data.timestamp || new Date().toISOString()), "</span>",
            "<span class=\"post-link-to red-text-link\" title=\"Link to this post\">No.</span>",
            "<span class=\"post-reply-to red-text-link\" title=\"Reply to this post\" data-thread=\"", data.thread, "\">", data.hash, "</span>",
            "<div class=\"post-options\" title=\"Post menu\">",
            "<span class=\"post-options-arrow\" data-post=\"", data.hash, "\"></span>",
            "<ul id=\"post-menu-", data.hash, "\" class=\"post-options-menu hide\">",
            "<li class=\"report-post\" data-post=\"", data.hash, "\">Report Post</li>",
            "<li class=\"hide-post\" data-post=\"", data.hash, "\">Hide Post</li>",
            "<li class=\"unhide-post\" data-post=\"", data.hash, "\">Unhide Post</li>",
            imgSearchBlock,
            "</ul>",
            "</div>",
            "</div>",
            fileBlock,
            "<p class=\"post-content\">", (data.comment || ""), "</p>",
            "</div>",
            "</div>",
            "</div>"
        ].join("");

        const threadContainer = dom("#post-" + data.thread + " .thread-container");
        if (threadContainer.length() > 0) {
            threadContainer.get(0).insertAdjacentHTML("beforeend", replyHtml);
        }

        const replyEl = dom("#post-" + data.hash);
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
