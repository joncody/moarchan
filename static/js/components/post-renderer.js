function escapeHtml(str) {
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

function renderImageBlock(data) {
    if (!data.file_name) {
        return "";
    }
    const safeFile = escapeHtml(data.file_name);
    const size = escapeHtml(String(data.file_size || "0"));
    const dims = escapeHtml(String(data.file_dimensions || "???x???"));

    return [
        "<div class=\"post-image-metadata\">",
        "File: <a class=\"post-image-link blue-text-link\" href=\"/static/images/uploads/", safeFile, "\" alt=\"", safeFile, "\" title=\"", safeFile, "\" target=\"_blank\">", safeFile, "</a>",
        "<span class=\"post-image-dimensions\">(", size, " KB, ", dims, ")</span>",
        "</div>",
        "<a class=\"post-image-container\" href=\"/static/images/uploads/", safeFile, "\" target=\"_blank\">",
        "<img class=\"post-image\" src=\"/static/images/uploads/thumb_", safeFile, "\" title=\"", safeFile, "\" alt=\"", safeFile, "\" />",
        "</a>"
    ].join("");
}

function renderThread(data) {
    const safeHash = escapeHtml(data.hash);
    const safeTopic = escapeHtml(data.topic);
    const safeSubject = escapeHtml(data.subject || "");
    const safeName = escapeHtml(data.name || "Anonymous");
    const safeTime = escapeHtml(data.timestamp || "");
    const comment = data.comment || ""; // Pre-sanitized by server

    let fileMetadata = "";
    let fileImage = "";
    if (data.file_name) {
        const safeFile = escapeHtml(data.file_name);
        const size = escapeHtml(String(data.file_size || "0"));
        const dims = escapeHtml(String(data.file_dimensions || "???x???"));
        fileMetadata = [
            "<div class=\"post-image-metadata op\">",
            "File: <a class=\"post-image-link blue-text-link op\" href=\"/static/images/uploads/", safeFile, "\" target=\"_blank\">", safeFile, "</a>",
            "<span class=\"post-image-dimensions op\">(", size, " KB, ", dims, ")</span>",
            "</div>"
        ].join("");
        fileImage = [
            "<a class=\"post-image-container op\" href=\"/static/images/uploads/", safeFile, "\" target=\"_blank\">",
            "<img class=\"post-image op\" src=\"/static/images/uploads/thumb_", safeFile, "\" alt=\"", safeFile, "\" />",
            "</a>"
        ].join("");
    }

    return [
        "<div id=\"post-", safeHash, "\" class=\"thread\">",
        "<div class=\"post-show-hide-icons op\">",
        "<img class=\"post-show-hide-thread plus\" data-post=\"", safeHash, "\" src=\"/static/images/show-hide-thread-plus-red.png\" alt=\"+\" />",
        "<img class=\"post-show-hide-thread minus\" data-post=\"", safeHash, "\" src=\"/static/images/show-hide-thread-minus-red.png\" alt=\"-\" />",
        "</div>",
        fileMetadata,
        fileImage,
        "<div class=\"post-header op\">",
        "<input class=\"post-checkbox op\" type=\"checkbox\" />",
        "<span class=\"post-subject op\">", safeSubject, "</span>",
        "<span class=\"post-username op\">", safeName, "</span>",
        "<span class=\"post-date op\">", safeTime, "</span>",
        "<span class=\"post-link-to red-text-link op\" title=\"Link to this post\" data-href=\"/", safeTopic, "/thread/", safeHash, "\">No.</span>",
        "<span class=\"post-reply-to red-text-link op\" title=\"Reply to this post\" data-thread=\"", safeHash, "\">", safeHash, "</span>",
        "<img class=\"post-thumbtack op\" src=\"/static/images/thumbtack.gif\" alt=\"Sticky\" />",
        "<img class=\"post-lock op\" src=\"/static/images/lock.gif\" alt=\"Closed\" />",
        "<span class=\"post-reply-to-text op\">[<span class=\"reply-link blue-text-link\" data-href=\"/", safeTopic, "/thread/", safeHash, "\">Reply</span>]</span>",
        "<div class=\"post-options op\" title=\"Post menu\">",
        "<span class=\"post-options-arrow op\" data-post=\"", safeHash, "\"></span>",
        "<ul id=\"post-menu-", safeHash, "\" class=\"post-options-menu hide op\">",
        "<li class=\"report-post op\" data-post=\"", safeHash, "\">Report Thread</li>",
        "<li class=\"hide-post op\" data-post=\"", safeHash, "\">Hide Thread</li>",
        "<li class=\"unhide-post op\" data-post=\"", safeHash, "\">Unhide Thread</li>",
        "<li class=\"image-search op\" data-post=\"", safeHash, "\">Image Search &gt;&gt;</li>",
        "</ul>",
        "</div>",
        "</div>",
        "<div class=\"thread-container\">",
        "<p class=\"post-content op\">", comment, "</p>",
        "<div class=\"post-summary-container\">",
        "<div class=\"post-show-hide-icons replies\">",
        "<img class=\"post-show-hide-replies plus\" data-post=\"", safeHash, "\" src=\"/static/images/show-hide-thread-plus-red.png\" alt=\"+\" />",
        "<img class=\"post-show-hide-replies minus\" data-post=\"", safeHash, "\" src=\"/static/images/show-hide-thread-minus-red.png\" alt=\"-\" />",
        "</div>",
        "<p class=\"post-summary\">0 posts omitted. <span class=\"blue-text-link\" data-href=\"/", safeTopic, "/thread/", safeHash, "\">Click here</span> to view.</p>",
        "</div>",
        "</div>",
        "<div class=\"spacer\"></div>",
        "</div>"
    ].join("");
}

function renderReply(data) {
    const safeHash = escapeHtml(data.hash);
    const safeThread = escapeHtml(data.thread);
    const safeName = escapeHtml(data.name || "Anonymous");
    const safeTime = escapeHtml(data.timestamp || "");
    const imgSearch = data.file_name
        ? "<li class=\"image-search\" data-post=\"" + safeHash + "\">Image Search &gt;&gt;</li>"
        : "";

    return [
        "<div class=\"reply-container\">",
        "<div class=\"reply-wrapper\">",
        "<span class=\"post-side-arrows\">&gt;&gt;</span>",
        "<div id=\"post-", safeHash, "\" class=\"reply\">",
        "<div class=\"post-header\">",
        "<input class=\"post-checkbox\" type=\"checkbox\" />",
        "<span class=\"post-username\">", safeName, "</span>",
        "<span class=\"post-date\">", safeTime, "</span>",
        "<span class=\"post-link-to red-text-link\" title=\"Link to this post\">No.</span>",
        "<span class=\"post-reply-to red-text-link\" title=\"Reply to this post\" data-thread=\"", safeThread, "\">", safeHash, "</span>",
        "<div class=\"post-options\" title=\"Post menu\">",
        "<span class=\"post-options-arrow\" data-post=\"", safeHash, "\"></span>",
        "<ul id=\"post-menu-", safeHash, "\" class=\"post-options-menu hide\">",
        "<li class=\"report-post\" data-post=\"", safeHash, "\">Report Post</li>",
        "<li class=\"hide-post\" data-post=\"", safeHash, "\">Hide Post</li>",
        "<li class=\"unhide-post\" data-post=\"", safeHash, "\">Unhide Post</li>",
        imgSearch,
        "</ul>",
        "</div>",
        "</div>",
        renderImageBlock(data),
        "<p class=\"post-content\">", (data.comment || ""), "</p>",
        "</div>",
        "</div>",
        "</div>"
    ].join("");
}

export default Object.freeze({
    escapeHtml,
    renderReply,
    renderThread
});
