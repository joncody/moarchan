"use strict";

import gg from "../gg.js";
import wsframe from "../wsframe.js";

wsframe.controllers.service = function service(global, view) {
    'use strict';

    var topicsMap = {
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
        },
        replyBoxHeader = gg('.reply-box-header'),
        replyBoxHeaderText = gg('.reply-box-header-text'),
        replyBoxPost = gg('.reply-box-post'),
        replyBox = gg('.reply-box'),
        hashsplit = global.location.hash.split('/'),
        mouseX,
        mouseY,
        room;

    gg('.topic-header').html('/' + hashsplit[1] + '/ - ' + topicsMap[hashsplit[1]]);
    wsframe.hrefListener();
    room = wsframe.socket.join(hashsplit[1]);

    function toggleBlotter(e) {
        var blotter = gg('.blotter');

        if (blotter.hasClass('hide')) {
            blotter.remClass('hide');
            gg('.hide-blotter-container').remClass('hide');
            gg('.show-all-blotter-container').remClass('hide');
            gg('.show-blotter-container').addClass('hide');
        } else {
            blotter.addClass('hide');
            gg('.hide-blotter-container').addClass('hide');
            gg('.show-all-blotter-container').addClass('hide');
            gg('.show-blotter-container').remClass('hide');
        }
    }
    gg('.hide-blotter').on('click', toggleBlotter, false);
    gg('.show-blotter').on('click', toggleBlotter, false);

    function showNewPostForm(e) {
        gg('.new-post').addClass('hide');
        gg('.new-post-form').remClass('hide');
    }
    gg('.new-post').on('click', showNewPostForm, false);

    function toggleThread(e, node) {
        gg('#post-' + node.data('post')).togClass('hide-thread');
    }
    gg('.post-show-hide-thread').on('click', toggleThread, false);

    function toggleReplies(e, node) {
        var hash = node.data('post'),
            thread = gg('#post-' + hash),
            replies = thread.select('.reply-container'),
            summary = '{omitted} posts omitted. <span class="blue-text-link" data-href="{href}">Click here</span> to view.',
            data = {};

        thread.togClass('show-replies');
        if (!thread.hasClass('show-replies') && replies.length > 5) {
            data.omitted = replies.length - 5;
            data.href = '/' + hashsplit[1] + '/thread/' + hash;
            thread.select('.post-summary').html(gg.utils.supplant(summary, data));
        } else {
            thread.select('.post-summary').html('Showing all replies.');
        }
        wsframe.assignHrefs();
    }
    gg('.post-show-hide-replies').on('click', toggleReplies, false);

    function hidePost(e, node) {
        var hash = node.data('post'),
            post = gg('#post-' + hash);

        if (post.hasClass('thread')) {
            post.addClass('hide-thread');
        } else if (post.hasClass('reply')) {
            post.addClass('hide-reply');
        }
        gg('.post-options-menu').addClass('hide');
    }
    gg('.hide-post').on('click', hidePost, false);

    function unhidePost(e, node) {
        var hash = node.data('post'),
            post = gg('#post-' + hash);

        if (post.hasClass('thread')) {
            post.remClass('hide-thread');
        } else if (post.hasClass('reply')) {
            post.remClass('hide-reply');
        }
        gg('.post-options-menu').addClass('hide');
    }
    gg('.unhide-post').on('click', unhidePost, false);

    function hidePostOptions(e, node, arg) {
        var menu = arg;

        if (!node.hasClass('post-options-arrow')) {
            menu.addClass('hide');
        }
    }

    function showPostOptions(e, node) {
        var hash = node.data('post'),
            menu = gg('#post-menu-' + hash);

        gg('.post-options-menu').addClass('hide');
        menu.remClass('hide');
        setTimeout(function () {
            gg(document.body).once('click', hidePostOptions, false, menu);
        }, 0);
    }
    gg('.post-options-arrow').on('click', showPostOptions, false);

    function clearForms() {
        gg('#reply-box-name').remData('value');
        gg('#reply-box-options').remData('value');
        gg('#reply-box-comment').remData('value');
        gg('#reply-box-file').remData('value');
        gg('#new-post-name').remData('value');
        if (gg.utils.getById('new-post-subject')) {
            gg('#new-post-subject').remData('value');
        }
        gg('#new-post-options').remData('value');
        gg('#new-post-comment').remData('value');
        gg('#new-post-file').remData('value');
        gg.utils.select('.reply-box-close').click();
    }

    function postThread(e) {
        var schema = {
                type: 'thread',
                topic: hashsplit[1],
                name: gg('#new-post-name').attr('value') || 'Anonymous',
                subject: gg('#new-post-subject').attr('value'),
                options: gg('#new-post-options').attr('value'),
                comment: gg('#new-post-comment').attr('value').replace(/\r?\n/g, '<br />'),
                replies: {},
                taggedBy: [],
                tagging: []
            },
            files = gg('#new-post-file').attr('files'),
            reader = new FileReader(),
            file;

        function fileLoaded(s) {
            return function (e) {
                s.file = e.target.result;
                room.send('new-thread', JSON.stringify(s));
                clearForms();
            };
        }

        if (files.length > 0) {
            file = files[0];
            schema.file_name = file.name;
            schema.file_mime = file.type;
            schema.file_size = file.size.toString();
            reader.onload = fileLoaded(schema);
            reader.readAsDataURL(file);
        } else {
            global.alert('Must add a file to post a new thread.');
        }
    }

    function postReply(e, node, arg) {
        var thread = arg,
            replybox = replyBox.hasClass('hide') ? false : true,
            schema = {
                type: 'reply',
                topic: hashsplit[1],
                thread: thread,
                name: replybox ? gg('#reply-box-name').attr('value') : gg('#new-post-name').attr('value') || 'Anonymous',
                options: replybox ? gg('#reply-box-options').attr('value') : gg('#new-post-options').attr('value'),
                taggedBy: [],
                tagging: []
            },
            comment = replybox ? gg('#reply-box-comment').attr('value') : gg('#new-post-comment').attr('value'),
            files = replybox ? gg('#reply-box-file').attr('files') : gg('#new-post-file').attr('files'),
            reader = new FileReader(),
            file;

        function fileLoaded(s) {
            return function (e) {
                s.file = e.target.result;
                room.send('new-reply', JSON.stringify(s));
                clearForms();
            };
        }

        if (!comment) {
            return global.alert('Must write a comment to post a reply.');
        }
        schema.comment = comment.replace(/\r?\n/g, '<br />').replace(/>>(\w+)/g, function (a, b) {
            schema.tagging.push(b);
            return '<span class="post-tag blue-text-link" data-tag="' + b + '">' + a + '</span>';
        });
        if (files.length > 0) {
            file = files[0];
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

    if (view === 'thread') {
        gg('#new-post-button').on('click', postReply, false, hashsplit[3]);
    } else {
        gg('#new-post-button').on('click', postThread, false);
    }

    function dragging(e) {
        var top = parseInt(replyBox.css('top'), 10) + e.clientY - mouseY + 'px',
            left = parseInt(replyBox.css('left'), 10) - mouseX + e.clientX + 'px';

        replyBox.css('top', top);
        replyBox.css('left', left);
        mouseX = e.clientX;
        mouseY = e.clientY;
    }

    function stopDrag(e) {
        gg(document.body).off('mousemove', dragging, false);
    }

    function startDrag(e) {
        mouseX = e.clientX;
        mouseY = e.clientY;
        gg(document.body).on('mousemove', dragging, false);
        gg(document.body).once('mouseup', stopDrag, false);
    }

    function closeReplyBox(e, node, arg) {
        var postFunc = arg.postFunc,
            thread = arg.thread;

        replyBoxPost.off('click', postFunc, false, thread);
        replyBox.addClass('hide');
    }

    function openReplyBox(e, node) {
        var thread = node.data('thread'),
            post = node.html();

        replyBoxHeaderText.html(thread);
        replyBoxHeaderText.attr('title', thread);
        replyBoxPost.on('click', postReply, false, thread);
        gg('.reply-box-close').on('click', closeReplyBox, false, {postFunc: postReply, thread: thread});
        gg('#reply-box-comment').attr('value', '>>' + post);
        replyBox.remClass('hide');
    }
    replyBoxHeader.on('mousedown', startDrag, false);
    gg('.post-reply-to').on('click', openReplyBox, false);

    function initReplies(hash) {
        var thread = gg('#post-' + hash),
            replies = thread.select('.reply-container'),
            summary = '{omitted} posts omitted. <span class="blue-text-link" data-href="{href}">Click here</span> to view.',
            data = {};

        if (replies && replies.length) {
            if (replies.length > 5) {
                data.omitted = replies.length - 5;
                data.href = '/' + hashsplit[1] + '/thread/' + hash;
                thread.addClass('show-summary');
                thread.select('.post-summary').html(gg.utils.supplant(summary, data));
            } else {
                thread.addClass('show-replies');
            }
        }
        wsframe.assignHrefs();
    }
    if (view === 'topic') {
        gg('.thread').each(function (node) {
            initReplies(node.id.slice(5));
        });
    }

    function isElementInViewport(el) {
        var rect = el.getBoundingClientRect();

        return rect.top >= 0 &&
            rect.left >= 0 &&
            rect.bottom <= (global.innerHeight || document.documentElement.clientHeight) &&
            rect.right <= (global.innerWidth || document.documentElement.clientWidth);
    }

    function goToTaggedPost(e, node, arg) {
        var tagged = arg;

        gg('.highlight').remClass('highlight');
        tagged.raw(0).scrollIntoView();
        if (!tagged.hasClass('thread')) {
            tagged.addClass('highlight');
        }
    }

    function hoverOutTag(e, node, arg) {
        var tagged = arg.tagged,
            clone = arg.clone;

        if (clone) {
            clone.remove();
        } else {
            tagged.remClass('highlight-hover');
        }
    }

    function hoverOverTag(e, node, arg) {
        var tagged = arg,
            inview = isElementInViewport(tagged.raw(0)),
            replyHeight = global.parseInt(tagged.css('height'), 10),
            replyWidth = global.parseInt(tagged.css('width'), 10),
            top = e.clientY - 20 > replyHeight ? e.pageY - replyHeight - 20 : e.pageY + 20,
            left = global.screen.width - e.pageX - 20 > replyWidth ? e.pageX + 20 : 10,
            clone;

        if (inview) {
            tagged.addClass('highlight-hover');
        } else {
            clone = tagged.clone(true)
                .css('position', 'absolute')
                .css('top', top + 'px')
                .css('left', left + 'px')
                .css('width', tagged.css('width'))
                .css('height', tagged.css('height'))
                .css('box-shadow', '1px 1px 6px 0 rgba(0, 0, 0, 0.6)')
                .appendTo(document.body);
        }
        node.once('mouseout', hoverOutTag, false, {tagged: tagged, clone: clone});
    }

    gg('.post-tag').each(function (node) {
        var tagged = gg('#post-' + node.data('tag'));

        if (!tagged.hasClass('thread')) {
            node.on('mouseover', hoverOverTag, false, tagged);
        }
        node.on('click', goToTaggedPost, false, tagged);
    });

    function addThread(buffer) {
        var data = JSON.parse(gg.utils.toStringFromCodes(buffer)),
            htmlString = '<div id="post-{hash}" class="thread">' +
                '<div class="post-show-hide-icons op">' +
                    '<img class="post-show-hide-thread plus" data-post="{hash}" src="/static/images/show-hide-thread-plus-red.png" alt="Plus" title="Plus" />' +
                    '<img class="post-show-hide-thread minus" data-post="{hash}" src="/static/images/show-hide-thread-minus-red.png" alt="Minus" title="Minus" />' +
                '</div>' +
                '<div class="post-image-metadata op">' +
                    'File: <a class="post-image-link blue-text-link op" href="/static/images/uploads/{file_name}" alt="{file_name}" title="{file_name}" target="_blank">{file_name}</a>' +
                    '<span class="post-image-dimensions op">({file_size} KB, {file_dimensions})</span>' +
                '</div>' +
                '<a class="post-image-container op" href="/static/images/uploads/{file_name}" target="_blank">' +
                    '<img class="post-image op" src="/static/images/uploads/{file_name}" alt="{file_name}" title="{file_name}" />' +
                '</a>' +
                '<div class="post-header op">' +
                    '<input class="post-checkbox op" type="checkbox" />' +
                    '<span class="post-subject op">{subject}</span>' +
                    '<span class="post-username op">{name}</span>' +
                    '<span class="post-date op">{timestamp}</span>' +
                    '<span class="post-link-to red-text-link op" title="Link to this post" data-href="/{topic}/thread/{hash}">No.</span>' +
                    '<span class="post-reply-to red-text-link op" title="Reply to this post" data-thread="{hash}">{hash}</span>' +
                    '<img class="post-thumbtack op" src="/static/images/thumbtack.gif" alt="Sticky" title="Sticky" />' +
                    '<img class="post-lock op" src="/static/images/lock.gif" alt="Closed" title="Closed" />' +
                    '<span class="post-reply-to-text op">[<span class="reply-link blue-text-link" data-href="/{topic}/thread/{hash}">Reply</span>]</span> ' +
                    '<div class="post-options op" title="Post menu">' +
                        '<span class="post-options-arrow op" data-post="{hash}"></span>' +
                        '<ul id="post-menu-{hash}" class="post-options-menu hide op">' +
                            '<li class="report-post op" data-post="{hash}">Report Thread</li>' +
                            '<li class="hide-post op" data-post="{hash}">Hide Thread</li>' +
                            '<li class="unhide-post op" data-post="{hash}">Unhide Thread</li>' +
                            '<li class="image-search op" data-post="{hash}">Image Search &gt;&gt;</li>' +
                        '</ul>' +
                    '</div>' +
                '</div>' +
                '<div class="thread-container">' +
                    '<p class="post-content op">{comment}</p>' +
                    '<div class="post-summary-container">' +
                        '<div class="post-show-hide-icons replies">' +
                            '<img class="post-show-hide-replies plus" data-post="{hash}" src="/static/images/show-hide-thread-plus-red.png" alt="Plus" title="Plus" />' +
                            '<img class="post-show-hide-replies minus" data-post="{hash}" src="/static/images/show-hide-thread-minus-red.png" alt="Minus" title="Minus" />' +
                        '</div>' +
                        '<p class="post-summary">0 posts omitted. <span class="blue-text-link" data-href="/{topic}/thread/{hash}">Click here</span> to view.</p>' +
                    '</div>' +
                '</div>' +
                '<div class="spacer"></div>' +
            '</div>',
            thread,
            threadEl;

        if (data.hash && !gg.utils.getById('post-' + data.hash)) {
            thread = gg.utils.supplant(htmlString, data);
            gg('.board').insert('beforeend', thread);
            threadEl = gg('#post-' + data.hash);
            threadEl.selectAll('.post-show-hide-thread').on('click', toggleThread, false);
            threadEl.selectAll('.post-show-hide-replies').on('click', toggleReplies, false);
            threadEl.selectAll('.post-show-replies').on('click', toggleReplies, false);
            threadEl.selectAll('.hide-post').on('click', hidePost, false);
            threadEl.selectAll('.unhide-post').on('click', unhidePost, false);
            threadEl.selectAll('.post-options-arrow').on('click', showPostOptions, false);
            threadEl.selectAll('.post-reply-to').on('click', openReplyBox, false);
            wsframe.assignHrefs();
        }
    }

    function addReply(buffer) {
        var data = JSON.parse(gg.utils.toStringFromCodes(buffer)),
            htmlString =
            '<div class="reply-container">' +
                '<div class="reply-wrapper">' +
                    '<span class="post-side-arrows">&gt;&gt;</span>' +
                    '<div id="post-{hash}" class="reply">' +
                        '<div class="post-header">' +
                            '<input class="post-checkbox" type="checkbox" />' +
                            '<span class="post-username">{name}</span>' +
                            '<span class="post-date">{timestamp}</span>' +
                            '<span class="post-link-to red-text-link" title="Link to this post">No.</span>' +
                            '<span class="post-reply-to red-text-link" title="Reply to this post" data-thread="{thread}">{hash}</span>' +
                            '<div class="post-options" title="Post menu">' +
                                '<span class="post-options-arrow" data-post="{hash}"></span>' +
                                '<ul id="post-menu-{hash}" class="post-options-menu hide">' +
                                    '<li class="report-post" data-post="{hash}">Report Post</li>' +
                                    '<li class="hide-post" data-post="{hash}">Hide Post</li>' +
                                    '<li class="unhide-post" data-post="{hash}">Unhide Post</li>',
            imgSearchString =       '<li class="image-search" data-post="{hash}">Image Search &gt;&gt;</li>',
            midString =         '</ul>' +
                            '</div>' +
                        '</div>',
            imgString = '<div class="post-image-metadata">' +
                            'File: <a class="post-image-link blue-text-link" href="/static/images/uploads/{file_name}" alt="{file_name}" title="{file_name}" target="_blank">{file_name}</a> ' +
                            '<span class="post-image-dimensions">({file_size} KB, {file_dimensions})</span>' +
                        '</div>' +
                        '<a class="post-image-container" href="/static/images/uploads/{file_name}" target="_blank">' +
                            '<img class="post-image" src="/static/images/uploads/{file_name}" title="{file_name}" alt="{file_name}" />' +
                        '</a>',
            endString =
                        '<p class="post-content">{comment}</p>' +
                    '</div>' +
                '</div>' +
            '</div>',
            thread,
            replies,
            reply,
            replyEl;

        if (data.hash && !gg.utils.getById('post-' + data.hash)) {
            data.tagging.forEach(function (tag) {
                var op = tag === data.thread ? ' op' : '',
                    el = gg.utils.supplant('<span class="post-tag blue-text-link' + op + '" data-tag="{hash}">&gt;&gt;{hash}</span>', data);

                gg('#post-' + tag + ' .post-header').insert('beforeend', el);
            });
            if (data.file_name) {
                htmlString += imgSearchString;
                htmlString += midString;
                htmlString += imgString;
            } else {
                htmlString += midString;
            }
            htmlString += endString;
            reply = gg.utils.supplant(htmlString, data);
            gg('#post-' + data.thread + ' .thread-container').insert('beforeend', reply);
            replyEl = gg('#post-' + data.hash);
            replyEl.selectAll('.hide-post').on('click', hidePost, false);
            replyEl.selectAll('.unhide-post').on('click', unhidePost, false);
            replyEl.selectAll('.post-options-arrow').on('click', showPostOptions, false);
            replyEl.selectAll('.post-reply-to').on('click', openReplyBox, false);
            gg('#post-' + data.thread).selectAll('.post-tag').each(function (node) {
                var tagged = gg('#post-' + node.data('tag'));

                if (!tagged.hasClass('thread')) {
                    node.on('mouseover', hoverOverTag, false, tagged);
                }
                node.on('click', goToTaggedPost, false, tagged);
            });
            initReplies(data.thread);
        }
    }

    room.on('new-reply', addReply);
    room.on('new-thread', addThread);
};
