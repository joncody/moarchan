import dom from "../dom.js";

export default function createTagHover(global) {
    function getDataTag(e) {
        const target = dom(e.currentTarget || e.target);
        const dataVal = target.data("tag");
        return Array.isArray(dataVal) ? dataVal[0] : dataVal;
    }

    function goToTaggedPost(e) {
        const tag = getDataTag(e);
        if (!tag) {
            return;
        }
        const tagged = dom("#post-" + tag);
        if (tagged.length() === 0) {
            return;
        }
        dom(".highlight").removeClass("highlight");
        const el = tagged.get(0);
        if (el && typeof el.scrollIntoView === "function") {
            el.scrollIntoView({ behavior: "smooth", block: "center" });
        }
        if (!tagged.hasClass("thread")) {
            tagged.addClass("highlight");
        }
    }

    function hoverOutTag(e) {
        const tag = getDataTag(e);
        if (tag) {
            dom("#post-" + tag).removeClass("highlight-hover");
        }
        dom(".tag-hover-clone").remove();
    }

    function hoverOverTag(e) {
        const tag = getDataTag(e);
        if (!tag) {
            return;
        }
        const tagged = dom("#post-" + tag);
        if (tagged.length() === 0) {
            return;
        }
        const el = tagged.get(0);
        if (!el || typeof el.getBoundingClientRect !== "function") {
            return;
        }

        const rect = el.getBoundingClientRect();
        const docEl = document.documentElement;
        const vHeight = global.innerHeight || docEl.clientHeight;
        const vWidth = global.innerWidth || docEl.clientWidth;
        const inview = (
            rect.top >= 0 &&
            rect.left >= 0 &&
            rect.bottom <= vHeight &&
            rect.right <= vWidth
        );

        if (inview) {
            tagged.addClass("highlight-hover");
            dom(e.currentTarget).once("mouseout", hoverOutTag, false);
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
            dom(e.currentTarget).once("mouseout", hoverOutTag, false);
        }
    }

    function bindTags() {
        dom(".post-tag").each(function (node) {
            const tagDom = dom(node);
            tagDom.off("mouseover", hoverOverTag, false);
            tagDom.off("click", goToTaggedPost, false);
            tagDom.on("mouseover", hoverOverTag, false);
            tagDom.on("click", goToTaggedPost, false);
        });
    }

    return Object.freeze({
        bindTags,
        cleanup: function () {
            dom(".post-tag").off("mouseover", hoverOverTag, false);
            dom(".post-tag").off("click", goToTaggedPost, false);
            dom(".tag-hover-clone").remove();
        }
    });
}
