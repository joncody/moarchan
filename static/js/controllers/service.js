"use strict";

import dom from "../dom.js";
import wsframe from "../wsframe.js";

const decoder = new TextDecoder("utf-8");

wsframe.controllers.service = function service(global, view) {
    'use strict';

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

    const replyBoxHeader = dom('.reply-box-header');
    const replyBoxHeaderText = dom('.reply-box-header-text');
    const replyBoxPost = dom('.reply-box-post');
    const replyBox = dom('.reply-box');
    const hashsplit = global.location.hash.split('/');
    let mouseX, mouseY;
    const room = wsframe.socket.join(hashsplit[1]);

    // Set board header
    dom('.topic-header').html(`/${hashsplit[1]}/ - ${topicsMap[hashsplit[1]] || 'Unknown'}`);

    // Blotter toggle
    function toggleBlotter(e) {
        const blotter = dom('.blotter');
        if (blotter.hasClass('hide')) {
            blotter.removeClass('hide');
            dom('.hide-blotter-container').removeClass('hide');
            dom('.show-all-blotter-container').removeClass('hide');
            dom('.show-blotter-container').addClass('hide');
        } else {
            blotter.addClass('hide');
            dom('.hide-blotter-container').addClass('hide');
            dom('.show-all-blotter-container').addClass('hide');
            dom('.show-blotter-container').removeClass('hide');
        }
    }
    dom('.hide-blotter').on('click', toggleBlotter, false);
    dom('.show-blotter').on('click', toggleBlotter, false);

    // New post form
    function showNewPostForm(e) {
        dom('.new-post').addClass('hide');
        dom('.new-post-form').removeClass('hide');
    }
    dom('.new-post').on('click', showNewPostForm, false);

    // Thread show/hide
    function toggleThread(e, node) {
        dom(`#post-${node.data('post')}`).toggleClass('hide-thread');
    }
    dom('.post-show-hide-thread').on('click', toggleThread, false);

    // Replies toggle
    function toggleReplies(e, node) {
        const hash = node.data('post');
        const thread = dom(`#post-${hash}`);
        const replies = thread.select('.reply-container');
        const data = {};

        thread.toggleClass('show-replies');
        const summaryEl = thread.select('.post-summary');

        if (!thread.hasClass('show-replies') && replies.length > 5) {
            data.omitted = replies.length - 5;
            data.href = `/${hashsplit[1]}/thread/${hash}`;
            summaryEl.html(`${data.omitted} posts omitted. <span class="blue-text-link" data-href="${data.href}">Click here</span> to view.`);
        } else {
            summaryEl.html('Showing all replies.');
        }
        wsframe.assignHrefs();
    }
    dom('.post-show-hide-replies').on('click', toggleReplies, false);

    // Hide post
    function hidePost(e, node) {
        const hash = node.data('post');
        const post = dom(`#post-${hash}`);
        if (post.hasClass('thread')) {
            post.addClass('hide-thread');
        } else if (post.hasClass('reply')) {
            post.addClass('hide-reply');
        }
        dom('.post-options-menu').addClass('hide');
    }
    dom('.hide-post').on('click', hidePost, false);

    // Unhide post
    function unhidePost(e, node) {
        const hash = node.data('post');
        const post = dom(`#post-${hash}`);
        if (post.hasClass('thread')) {
            post.removeClass('hide-thread');
        } else if (post.hasClass('reply')) {
            post.removeClass('hide-reply');
        }
        dom('.post-options-menu').addClass('hide');
    }
    dom('.unhide-post').on('click', unhidePost, false);

    // Post options menu
    function hidePostOptions(e, node, arg) {
        if (!node.hasClass('post-options-arrow')) {
            arg.addClass('hide');
        }
    }

    function showPostOptions(e, node) {
        const hash = node.data('post');
        const menu = dom(`#post-menu-${hash}`);
        dom('.post-options-menu').addClass('hide');
        menu.removeClass('hide');
        setTimeout(() => {
            dom(document.body).once('click', hidePostOptions, false, menu);
        }, 0);
    }
    dom('.post-options-arrow').on('click', showPostOptions, false);

    // Clear forms
    function clearForms() {
        dom('#reply-box-name').data('value', null);
        dom('#reply-box-options').data('value', null);
        dom('#reply-box-comment').data('value', null);
        dom('#reply-box-file').data('value', null);
        dom('#new-post-name').data('value', null);
        if (document.getElementById('new-post-subject')) {
            dom('#new-post-subject').data('value', null);
        }
        dom('#new-post-options').data('value', null);
        dom('#new-post-comment').data('value', null);
        dom('#new-post-file').data('value', null);
        // Trigger close button click safely
        dom('.reply-box-close').each(el => el.click());
    }

    // Post new thread
    function postThread(e) {
        const nameInput = dom('#new-post-name').get(0);
        const subjectInput = dom('#new-post-subject').get(0);
        const optionsInput = dom('#new-post-options').get(0);
        const commentInput = dom('#new-post-comment').get(0);
        const fileInput = dom('#new-post-file').get(0);

        const schema = {
            type: 'thread',
            topic: hashsplit[1],
            name: nameInput?.value || 'Anonymous',
            subject: subjectInput?.value || '',
            options: optionsInput?.value || '',
            comment: (commentInput?.value || '').replace(/\r?\n/g, '<br />'),
            replies: {},
            taggedBy: [],
            tagging: []
        };

        const files = fileInput?.files || [];
        const reader = new FileReader();

        function fileLoaded(s) {
            return function (e) {
                s.file = e.target.result;
                room.send('new-thread', JSON.stringify(s));
                clearForms();
            };
        }

        if (files.length > 0) {
            const file = files[0];
            schema.file_name = file.name;
            schema.file_mime = file.type;
            schema.file_size = file.size.toString();
            reader.onload = fileLoaded(schema);
            reader.readAsDataURL(file);
        } else {
            global.alert('Must add a file to post a new thread.');
        }
    }

    // Post reply
    function postReply(e, node, arg) {
        const thread = arg;
        const replyboxVisible = !replyBox.hasClass('hide');
        const nameInput = replyboxVisible ? dom('#reply-box-name').get(0) : dom('#new-post-name').get(0);
        const optionsInput = replyboxVisible ? dom('#reply-box-options').get(0) : dom('#new-post-options').get(0);
        const commentInput = replyboxVisible ? dom('#reply-box-comment').get(0) : dom('#new-post-comment').get(0);
        const fileInput = replyboxVisible ? dom('#reply-box-file').get(0) : dom('#new-post-file').get(0);

        const comment = commentInput?.value || '';
        if (!comment) {
            return global.alert('Must write a comment to post a reply.');
        }

        const schema = {
            type: 'reply',
            topic: hashsplit[1],
            thread: thread,
            name: nameInput?.value || 'Anonymous',
            options: optionsInput?.value || '',
            taggedBy: [],
            tagging: []
        };

        schema.comment = comment
            .replace(/\r?\n/g, '<br />')
            .replace(/>>(\w+)/g, (match, postHash) => {
                schema.tagging.push(postHash);
                return `<span class="post-tag blue-text-link" data-tag="${postHash}">${match}</span>`;
            });

        const files = fileInput?.files || [];
        const reader = new FileReader();

        function fileLoaded(s) {
            return function (e) {
                s.file = e.target.result;
                room.send('new-reply', JSON.stringify(s));
                clearForms();
            };
        }

        if (files.length > 0) {
            const file = files[0];
            schema.file_name = file.name;
            schema.file_mime = file.type;
            schema.file_size = file.size.toString();
            reader.onload = fileLoaded(schema);
            reader.readAsDataURL(file);
        } else {
            room.send('new-reply', JSON.stringify(schema));
            clearForms();
        }
    }

    // Attach post button
    if (view === 'thread') {
        dom('#new-post-button').on('click', postReply, false, hashsplit[3]);
    } else {
        dom('#new-post-button').on('click', postThread, false);
    }

    // Reply box drag
    function dragging(e) {
        const top = parseInt(replyBox.css('top')[0], 10) + e.clientY - mouseY + 'px';
        const left = parseInt(replyBox.css('left')[0], 10) - mouseX + e.clientX + 'px';
        replyBox.css('top', top).css('left', left);
        mouseX = e.clientX;
        mouseY = e.clientY;
    }

    function stopDrag(e) {
        dom(document.body).off('mousemove', dragging, false);
    }

    function startDrag(e) {
        mouseX = e.clientX;
        mouseY = e.clientY;
        dom(document.body).on('mousemove', dragging, false);
        dom(document.body).once('mouseup', stopDrag, false);
    }

    function closeReplyBox(e, node, arg) {
        const { postFunc, thread } = arg;
        replyBoxPost.off('click', postFunc, false, thread);
        replyBox.addClass('hide');
    }

    function openReplyBox(e, node) {
        const thread = node.data('thread');
        const post = node.html()[0] || '';

        replyBoxHeaderText.html(thread).attr('title', thread);
        replyBoxPost.on('click', postReply, false, thread);
        dom('.reply-box-close').on('click', closeReplyBox, false, { postFunc: postReply, thread });
        dom('#reply-box-comment').attr('value', `>>${post}`);
        replyBox.removeClass('hide');
    }

    replyBoxHeader.on('mousedown', startDrag, false);
    dom('.post-reply-to').on('click', openReplyBox, false);

    // Initialize thread replies visibility
    function initReplies(hash) {
        const thread = dom(`#post-${hash}`);
        const replies = thread.select('.reply-container');
        const summaryEl = thread.select('.post-summary');
        const data = {};

        if (replies.length > 0) {
            if (replies.length > 5) {
                data.omitted = replies.length - 5;
                data.href = `/${hashsplit[1]}/thread/${hash}`;
                thread.addClass('show-summary');
                summaryEl.html(`${data.omitted} posts omitted. <span class="blue-text-link" data-href="${data.href}">Click here</span> to view.`);
            } else {
                thread.addClass('show-replies');
            }
        }
        wsframe.assignHrefs();
    }

    if (view === 'topic') {
        dom('.thread').each(node => {
            initReplies(node.id.slice(5));
        });
    }

    // Scroll to tagged post
    function goToTaggedPost(e, node, arg) {
        const tagged = arg;
        dom('.highlight').removeClass('highlight');
        const el = tagged.get(0);
        if (el) el.scrollIntoView();
        if (!tagged.hasClass('thread')) {
            tagged.addClass('highlight');
        }
    }

    // Hover over >>link
    function hoverOutTag(e, node, arg) {
        const { tagged, clone } = arg;
        if (clone) {
            clone.remove();
        } else {
            tagged.removeClass('highlight-hover');
        }
    }

    function hoverOverTag(e, node, arg) {
        const tagged = arg;
        const el = tagged.get(0);
        if (!el) return;

        const rect = el.getBoundingClientRect();
        const inview = (
            rect.top >= 0 &&
            rect.left >= 0 &&
            rect.bottom <= (global.innerHeight || document.documentElement.clientHeight) &&
            rect.right <= (global.innerWidth || document.documentElement.clientWidth)
        );

        if (inview) {
            tagged.addClass('highlight-hover');
            node.once('mouseout', hoverOutTag, false, { tagged, clone: null });
        } else {
            const clone = tagged.clone(true)
                .css('position', 'absolute')
                .css('top', (e.pageY - rect.height - 20) + 'px')
                .css('left', (e.pageX + 20) + 'px')
                .css('width', rect.width + 'px')
                .css('height', rect.height + 'px')
                .css('box-shadow', '1px 1px 6px 0 rgba(0, 0, 0, 0.6)')
                .appendTo(document.body);
            node.once('mouseout', hoverOutTag, false, { tagged, clone });
        }
    }

    dom('.post-tag').each(node => {
        const tag = node.data('tag');
        const tagged = dom(`#post-${tag}`);
        if (tagged.length() > 0 && !tagged.hasClass('thread')) {
            node.on('mouseover', hoverOverTag, false, tagged);
        }
        node.on('click', goToTaggedPost, false, tagged);
    });

    // === Real-time updates ===

    function addThread(buffer) {
        const rawData = decoder.decode(buffer);
        let data;
        try {
            data = JSON.parse(rawData);
        } catch (err) {
            return;
        }

        if (!data.hash || document.getElementById(`post-${data.hash}`)) return;

        // Fallbacks for missing fields
        const subject = data.subject || '';
        const name = data.name || 'Anonymous';
        const file_name = data.file_name || '';
        const file_size = data.file_size ? (parseInt(data.file_size, 10) / 1024).toFixed(1) : '0';
        const file_dimensions = data.file_dimensions || '???x???';
        const comment = data.comment || '';
        const timestamp = data.timestamp || new Date().toISOString();

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

        dom('.board').insert('beforeend', htmlString);
        const threadEl = dom(`#post-${data.hash}`);
        threadEl.selectAll('.post-show-hide-thread').on('click', toggleThread, false);
        threadEl.selectAll('.post-show-hide-replies').on('click', toggleReplies, false);
        threadEl.selectAll('.hide-post').on('click', hidePost, false);
        threadEl.selectAll('.unhide-post').on('click', unhidePost, false);
        threadEl.selectAll('.post-options-arrow').on('click', showPostOptions, false);
        threadEl.selectAll('.post-reply-to').on('click', openReplyBox, false);
        wsframe.assignHrefs();
    }

    function addReply(buffer) {
        const rawData = decoder.decode(buffer);
        let data;
        try {
            data = JSON.parse(rawData);
        } catch (err) {
            return;
        }

        if (!data.hash || document.getElementById(`post-${data.hash}`)) return;

        // Process tagging
        data.tagging.forEach(tag => {
            const isOp = tag === data.thread;
            const opClass = isOp ? ' op' : '';
            const tagEl = `<span class="post-tag blue-text-link${opClass}" data-tag="${data.hash}">&gt;&gt;${data.hash}</span>`;
            const header = dom(`#post-${tag} .post-header`);
            if (header.length() > 0) {
                header.get(0).insertAdjacentHTML('beforeend', tagEl);
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
                <span class="post-username">${data.name || 'Anonymous'}</span>
                <span class="post-date">${data.timestamp || new Date().toISOString()}</span>
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
            const file_size_kb = data.file_size ? (parseInt(data.file_size, 10) / 1024).toFixed(1) : '0';
            const dims = data.file_dimensions || '???x???';
            html += `
            <div class="post-image-metadata">
                File: <a class="post-image-link blue-text-link" href="/static/images/uploads/${data.file_name}" alt="${data.file_name}" title="${data.file_name}" target="_blank">${data.file_name}</a>
                <span class="post-image-dimensions">(${file_size_kb} KB, ${dims})</span>
            </div>
            <a class="post-image-container" href="/static/images/uploads/${data.file_name}" target="_blank">
                <img class="post-image" src="/static/images/uploads/${data.file_name}" title="${data.file_name}" alt="${data.file_name}" />
            </a>`;
        }

        // Comment
        html += `
            <p class="post-content">${data.comment || ''}</p>
        </div>
    </div>
</div>`;

        const threadContainer = dom(`#post-${data.thread} .thread-container`);
        if (threadContainer.length() > 0) {
            threadContainer.get(0).insertAdjacentHTML('beforeend', html);
        }

        const replyEl = dom(`#post-${data.hash}`);
        replyEl.selectAll('.hide-post').on('click', hidePost, false);
        replyEl.selectAll('.unhide-post').on('click', unhidePost, false);
        replyEl.selectAll('.post-options-arrow').on('click', showPostOptions, false);
        replyEl.selectAll('.post-reply-to').on('click', openReplyBox, false);

        // Rebind tags in this thread
        dom(`#post-${data.thread} .post-tag`).each(node => {
            const tag = node.data('tag');
            const tagged = dom(`#post-${tag}`);
            if (tagged.length() > 0 && !tagged.hasClass('thread')) {
                node.off('mouseover').off('click'); // prevent duplicates
                node.on('mouseover', hoverOverTag, false, tagged);
                node.on('click', goToTaggedPost, false, tagged);
            }
        });

        initReplies(data.thread);
    }

    room.on('new-reply', addReply);
    room.on('new-thread', addThread);
};
