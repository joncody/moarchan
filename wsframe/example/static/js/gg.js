"use strict";

import utils from "./utils.js";

const global = (
    globalThis !== undefined
    ? globalThis
    : (
        window !== undefined
        ? window
        : this
    )
);

const VALID_TAGS = new Set([
  "a", "abbr", "address", "area", "article", "aside", "audio", "b", "base",
  "bdo", "blockquote", "body", "br", "button", "canvas", "caption", "cite",
  "code", "col", "colgroup", "dd", "del", "details", "dfn", "dialog", "div",
  "dl", "dt", "em", "embed", "fieldset", "figcaption", "figure", "footer",
  "form", "h1", "h2", "h3", "h4", "h5", "h6", "head", "header", "hr", "i",
  "iframe", "img", "input", "ins", "kbd", "label", "legend", "li", "link",
  "main", "map", "mark", "meta", "nav", "noscript", "object", "ol",
  "optgroup", "option", "p", "param", "picture", "pre", "progress", "q",
  "rp", "rt", "ruby", "s", "samp", "script", "section", "select", "small",
  "source", "span", "strong", "style", "sub", "summary", "sup", "svg",
  "table", "tbody", "td", "template", "textarea", "tfoot", "th", "thead",
  "time", "title", "tr", "track", "u", "ul", "var", "video", "wbr"
]);

// GG
const gg = function (selector) {
    const elements = [];

    if (typeof selector === "string") {
        elements.push(...document.querySelectorAll(selector));
    } else if (utils.isNode(selector)) {
        elements.push(selector);
    } else if (utils.objectType(selector) === "nodelist") {
        elements.push(...selector);
    } else if (Array.isArray(selector) && selector.every(utils.isNode)) {
        elements.push(...selector);
    } else {
        throw new TypeError("gg requires a selector string, Node, NodeList, or an array of Nodes");
    }

    const api = {
        addElement: function (val) {
            if (typeof val === "string") {
                elements.push(...document.querySelectorAll(val));
            } else if (utils.isNode(val)) {
                elements.push(val);
            } else if (utils.objectType(val) === "nodelist") {
                elements.push(...val);
            } else if (Array.isArray(val) && val.every(utils.isNode)) {
                elements.push(...val);
            }
            return api;
        },
        removeElement: function (index) {
            if (typeof index === "number" && index >= 0 && index < elements.length) {
                elements.splice(index, 1);
            }
            return api;
        },
        addClass: function (token) {
            if (typeof token !== "string" || !token.trim()) {
                return api;
            }
            elements.forEach((el) => {
                token.trim().split(/\s+/).forEach((c) => {
                    el.classList.add(c);
                });
            });
            return api;
        },
        removeClass: function (token) {
            if (typeof token !== "string" || !token.trim()) {
                return api;
            }
            elements.forEach((el) => {
                token.trim().split(/\s+/).forEach((c) => {
                    el.classList.remove(c);
                });
            });
            return api;
        },
        toggleClass: function (token, force) {
            if (typeof token !== "string" || !token.trim()) {
                return api;
            }
            elements.forEach((el) => {
                token.trim().split(/\s+/).forEach((c) => {
                    el.classList.toggle(c, force);
                });
            });
            return api;
        },
        hasClass: function (token) {
            if (typeof token !== "string" || !token.trim()) {
                return api;
            }
            return elements.every((el) => {
                return el.classList.contains(token);
            });
        },
        getClassName: function () {
            return elements.map((el) => el.className);
        },
        attr: function (name, value) {
            if (typeof name !== "string") {
                return api;
            }
            if (value === undefined) {
                return elements.map((el) => el.getAttribute(name));
            }
            if (value === null) {
                elements.forEach((el) => el.removeAttribute(name));
            } else {
                elements.forEach((el) => el.setAttribute(name, value));
            }
            return api;
        },
        children: function () {
            return gg(elements.flatMap((el) => [...el.children]));
        },
        clone: function (deep = true) {
            const clones = elements.map((el) => el.cloneNode(deep));
            return gg(clones);
        },
        data: function (name, value) {
            if (typeof name !== "string") {
                return api;
            }
            if (value === undefined) {
                return elements.map((el) => el.dataset[name]);
            }
            if (value === null) {
                elements.forEach((el) => delete el.dataset[name]);
            } else {
                elements.forEach((el) => el.dataset[name] = value);
            }
            return api;
        },
        each: function (fn) {
            elements.forEach(fn);
            return api;
        },
        get: function (index) {
            if (typeof index === "number" && index >= 0 && index < elements.length) {
                return elements[index];
            }
            return [...elements];
        },
        html: function (value) {
            if (value === undefined) {
                return elements.map((el) => el.innerHTML);
            }
            if (typeof value !== "string") {
                return api;
            }
            elements.forEach((el) => el.innerHTML = value);
            return api;
        },
        length: function () {
            return elements.length;
        },
        off: function (type, fn, capture = false) {
            if (typeof type !== "string" || typeof fn !== "function") {
                return api;
            }
            elements.forEach((el) => el.removeEventListener(type, fn, capture));
            return api;
        },
        on: function (type, fn, capture = false) {
            if (typeof type !== "string" || typeof fn !== "function") {
                return api;
            }
            elements.forEach((el) => el.addEventListener(type, fn, capture));
            return api;
        },
        once: function (type, fn, capture = false) {
            if (typeof type !== "string" || typeof fn !== "function") {
                return api;
            }
            elements.forEach((el) => {
                const onceFn = function (event) {
                    el.removeEventListener(type, onceFn, capture);
                    fn(event, el);
                };
                el.addEventListener(type, onceFn, capture);
            });
            return api;
        },
        parents: function () {
            return gg(elements.map((el) => el.parentElement));
        },
        css: function (name, value) {
            if (typeof name !== "string") {
                return api;
            }
            const camelName = utils.camelCase(name);
            const kebabName = utils.kebabCase(name);
            if (value === undefined) {
                return elements.map((el) => {
                    return global.getComputedStyle(el).getPropertyValue(kebabName) || '';
                });
            }
            if (value === null) {
                elements.forEach((el) => el.style.removeProperty(kebabName));
            } else {
                elements.forEach((el) => el.style[camelName] = value);
            }
            return api;
        },
        remove: function () {
            elements.forEach((el) => {
                if (el.parentNode) {
                    el.parentNode.removeChild(el);
                }
            });
            return api;
        },
        select: function (token) {
            if (typeof token !== "string") {
                return api;
            }
            const nodes = [];
            elements.forEach(function (el) {
                nodes.push(el.querySelector(token));
            });
            return gg(nodes);
        },
        selectAll: function (token) {
            if (typeof token !== "string") {
                return api;
            }
            const nodes = [];
            elements.forEach(function (el) {
                nodes.push(...el.querySelectorAll(token));
            });
            return gg(nodes);
        },
        text: function (value) {
            if (value === undefined) {
                return elements.map((el) => el.textContent);
            }
            if (typeof value !== "string") {
                return api;
            }
            elements.forEach((el) => {
                el.textContent = value;
            });
            return api;
        }
    };

    return Object.freeze(api);
};

gg.create = function (tag) {
    if (typeof tag !== "string") {
        return null;
    }
    tag = tag.toLowerCase();
    if (!VALID_TAGS.has(tag)) {
        return null;
    }
    return gg(document.createElement(tag));
};

export default Object.freeze(gg);
