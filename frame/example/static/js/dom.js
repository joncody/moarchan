"use strict";

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

const eventRegistry = new WeakMap();

function objectType(obj) {
    return Object.prototype.toString.call(obj).slice(8, -1).toLowerCase();
}

function isNode(value) {
    return value
        && typeof value === "object"
        && typeof value.nodeName === "string"
        && typeof value.nodeType === "number";
}

function camelCase(value) {
    if (typeof value !== "string") {
        return value;
    }
    return value.replace(/-([a-z])/g, (ignore, letter) => letter.toUpperCase());
}

function kebabCase(value) {
    if (typeof value !== "string") {
        return value;
    }
    return value.replace(/([A-Z])/g, "-$1").toLowerCase();
}

function toElements(selector) {
    const elements = [];

    // dom(...) object
    if (selector && typeof selector === "object" && typeof selector.get === "function") {
        elements.push(...selector.get());
    // CSS selector
    } else if (typeof selector === "string") {
        elements.push(...Array.from(document.querySelectorAll(selector)));
    // Node
    } else if (isNode(selector)) {
        if (objectType(selector) === "documentfragment") {
            // expand fragment to its element children
            elements.push(...Array.from(selector.children));
        } else {
            elements.push(selector);
        }
    // NodeList or HTMLCollection
    } else if (objectType(selector) === "nodelist" || objectType(selector) === "htmlcollection") {
        elements.push(...Array.from(selector));
    // Array of nodes
    } else if (Array.isArray(selector) && selector.every(isNode)) {
        elements.push(...selector);
    }
    return elements;
}

// Factory
function dom(selector) {
    const elements = toElements(selector);
    const api = {
        // === Core / Inspection ===
        get: function (index) {
            if (typeof index === "number" && index >= 0 && index < elements.length) {
                return elements[index];
            }
            return [...elements];
        },
        length: function () {
            return elements.length;
        },
        each: function (fn) {
            if (typeof fn !== "function") {
                return api;
            }
            elements.forEach(fn, api);
            return api;
        },

        // === Collection Management ===
        addItem: function (val) {
            elements.push(...toElements(val));
            return api;
        },
        removeItem: function (index) {
            if (typeof index === "number" && index >= 0 && index < elements.length) {
                elements.splice(index, 1);
            }
            return api;
        },

        // === Attributes & Classes ===
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
        data: function (name, value) {
            if (typeof name !== "string") {
                return api;
            }
            if (value === undefined) {
                return elements.map((el) => el.dataset[name]);
            }
            if (value === null) {
                elements.forEach(function (el) {
                    delete el.dataset[name];
                });
            } else {
                elements.forEach(function (el) {
                    el.dataset[name] = value;
                });
            }
            return api;
        },
        hasClass: function (token) {
            if (typeof token !== "string" || !token.trim()) {
                return false;
            }
            return elements.every((el) => el.classList.contains(token));
        },
        addClass: function (token) {
            if (typeof token !== "string" || !token.trim()) {
                return api;
            }
            elements.forEach(function (el) {
                token.trim().split(/\s+/).forEach(function (c) {
                    el.classList.add(c);
                });
            });
            return api;
        },
        removeClass: function (token) {
            if (typeof token !== "string" || !token.trim()) {
                return api;
            }
            elements.forEach(function (el) {
                token.trim().split(/\s+/).forEach(function (c) {
                    el.classList.remove(c);
                });
            });
            return api;
        },
        toggleClass: function (token, force) {
            if (typeof token !== "string" || !token.trim()) {
                return api;
            }
            elements.forEach(function (el) {
                token.trim().split(/\s+/).forEach(function (c) {
                    el.classList.toggle(c, force);
                });
            });
            return api;
        },
        getClassName: function () {
            return elements.map((el) => el.className);
        },

        // === Content ===
        text: function (value) {
            if (value === undefined) {
                return elements.map((el) => el.textContent);
            }
            if (typeof value !== "string") {
                return api;
            }
            elements.forEach(function (el) {
                el.textContent = value;
            });
            return api;
        },
        html: function (value) {
            if (value === undefined) {
                return elements.map((el) => el.innerHTML);
            }
            if (typeof value !== "string") {
                return api;
            }
            elements.forEach(function (el) {
                el.innerHTML = value;
            });
            return api;
        },

        // === DOM Traversal ===
        next: function () {
            return dom(elements.map((el) => el.nextElementSibling).filter(Boolean));
        },
        prev: function () {
            return dom(elements.map((el) => el.previousElementSibling).filter(Boolean));
        },
        children: function () {
            return dom(elements.flatMap((el) => Array.from(el.children)));
        },
        parents: function () {
            return dom(elements.map((el) => el.parentElement).filter(Boolean));
        },
        siblings: function () {
            return dom(Array.from(new Set(elements.flatMap(function (el) {
                return (
                    el.parentElement
                    ? Array.from(el.parentElement.children).filter((sib) => sib !== el)
                    : []
                );
            }))));
        },
        select: function (token) {
            if (typeof token !== "string") {
                return api;
            }
            return dom(Array.from(new Set(elements.map((el) => el.querySelector(token)).filter(Boolean))));
        },
        selectAll: function (token) {
            if (typeof token !== "string") {
                return api;
            }
            return dom(Array.from(new Set(elements.flatMap((el) => Array.from(el.querySelectorAll(token))))));
        },

        // === DOM Manipulation ===
        clone: function (deep = true) {
            return dom(elements.map((el) => el.cloneNode(deep)));
        },
        remove: function () {
            elements.forEach(function (el) {
                if (el.parentNode) {
                    el.parentNode.removeChild(el);
                }
            });
            return api;
        },

        // === Styling ===
        css: function (name, value) {
            if (typeof name !== "string") {
                return api;
            }
            const camelName = camelCase(name);
            const kebabName = kebabCase(name);
            if (value === undefined) {
                return elements.map(function (el) {
                    return global.getComputedStyle(el).getPropertyValue(kebabName) || "";
                });
            }
            if (value === null) {
                elements.forEach(function (el) {
                    el.style.removeProperty(kebabName);
                });
            } else {
                elements.forEach(function (el) {
                    el.style[camelName] = value;
                });
            }
            return api;
        },

        // === Events ===
        on: function (type, fn, capture = false) {
            if (typeof type !== "string" || typeof fn !== "function") {
                return api;
            }
            elements.forEach(function (el) {
                let register = eventRegistry.get(el);
                if (!register) {
                    register = {};
                    eventRegistry.set(el, register);
                }
                if (!register[type]) {
                    register[type] = [];
                }
                register[type].push({fn, capture});
                el.addEventListener(type, fn, capture);
            });
            return api;
        },
        off: function (type, fn, capture = false) {
            if (!type) {
                elements.forEach(function (el) {
                    const register = eventRegistry.get(el);
                    if (register) {
                        Object.keys(register).forEach(function (eventType) {
                            register[eventType].forEach(function ({fn: f, capture: c}) {
                                el.removeEventListener(eventType, f, c);
                            });
                        });
                        eventRegistry.delete(el);
                    }
                });
                return api;
            }
            if (typeof type === "string" && !fn) {
                elements.forEach(function (el) {
                    const register = eventRegistry.get(el);
                    if (register && register[type]) {
                        register[type].forEach(function ({fn: f, capture: c}) {
                            el.removeEventListener(type, f, c);
                        });
                        delete register[type];
                        if (Object.keys(register).length === 0) {
                            eventRegistry.delete(el);
                        }
                    }
                });
                return api;
            }
            if (typeof type === "string" && typeof fn === "function") {
                elements.forEach(function (el) {
                    const register = eventRegistry.get(el);
                    if (register && register[type]) {
                        register[type] = register[type].filter(function ({fn: f, capture: c}) {
                            if (f === fn && c === capture) {
                                el.removeEventListener(type, f, c);
                                return false;
                            }
                            return true;
                        });
                        if (register[type].length === 0) {
                            delete register[type];
                            if (Object.keys(register).length === 0) {
                                eventRegistry.delete(el);
                            }
                        }
                    }
                });
                return api;
            }
            return api;
        },
        once: function (type, fn, capture = false) {
            if (typeof type !== "string" || typeof fn !== "function") {
                return api;
            }
            elements.forEach(function (el) {
                const wrapper = function (event) {
                    el.removeEventListener(type, wrapper, capture);
                    const register = eventRegistry.get(el);
                    if (register && register[type]) {
                        register[type] = register[type].filter((h) => h.fn !== wrapper);
                        if (register[type].length === 0) {
                            delete register[type];
                            if (Object.keys(register).length === 0) {
                                eventRegistry.delete(el);
                            }
                        }
                    }
                    fn(event);
                };
                let register = eventRegistry.get(el);
                if (!register) {
                    register = {};
                    eventRegistry.set(el, register);
                }
                if (!register[type]) {
                    register[type] = [];
                }
                register[type].push({fn: wrapper, capture});
                el.addEventListener(type, wrapper, capture);
            });
            return api;
        }
    };
    return Object.freeze(api);
}

dom.create = function (tag) {
    if (typeof tag !== "string") {
        return dom();
    }
    tag = tag.toLowerCase();
    if (!VALID_TAGS.has(tag)) {
        return dom();
    }
    return dom(document.createElement(tag));
};

export default Object.freeze(dom);
