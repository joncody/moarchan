const VALID_TAGS = new Set([
    "a", "abbr", "address", "area", "article", "aside", "audio", "b",
    "base", "bdo", "blockquote", "body", "br", "button", "canvas",
    "caption", "cite", "code", "col", "colgroup", "dd", "del",
    "details", "dfn", "dialog", "div", "dl", "dt", "em", "embed",
    "fieldset", "figcaption", "figure", "footer", "form", "h1",
    "h2", "h3", "h4", "h5", "h6", "head", "header", "hr", "i",
    "iframe", "img", "input", "ins", "kbd", "label", "legend",
    "li", "link", "main", "map", "mark", "meta", "nav", "noscript",
    "object", "ol", "optgroup", "option", "p", "param", "picture",
    "pre", "progress", "q", "rp", "rt", "ruby", "s", "samp",
    "script", "section", "select", "small", "source", "span",
    "strong", "style", "sub", "summary", "sup", "svg", "table",
    "tbody", "td", "template", "textarea", "tfoot", "th", "thead",
    "time", "title", "tr", "track", "u", "ul", "var", "video", "wbr"
]);

const eventRegistry = new WeakMap();

function objectType(obj) {
    if (obj === null) {
        return "null";
    }
    if (obj === undefined) {
        return "undefined";
    }
    return Object.prototype.toString.call(obj).slice(8, -1).toLowerCase();
}

function isNode(value) {
    return (
        value !== null
        && typeof value === "object"
        && typeof value.nodeName === "string"
        && typeof value.nodeType === "number"
    );
}

function camelCase(value) {
    if (typeof value !== "string") {
        return value;
    }
    return value.replace(/-([a-z])/g, function (ignore, letter) {
        return letter.toUpperCase();
    });
}

function kebabCase(value) {
    if (typeof value !== "string") {
        return value;
    }
    return value.replace(/([A-Z])/g, "-$1").toLowerCase();
}

function toElements(selector) {
    if (selector === null || selector === undefined) {
        return [];
    }
    if (
        typeof selector === "object"
        && typeof selector.get === "function"
    ) {
        return selector.get();
    }
    if (typeof selector === "string") {
        if (document !== undefined) {
            return Array.from(document.querySelectorAll(selector));
        }
        return [];
    }
    if (isNode(selector)) {
        if (objectType(selector) === "documentfragment") {
            return Array.from(selector.children);
        }
        return [selector];
    }
    const type = objectType(selector);
    if (type === "nodelist" || type === "htmlcollection") {
        return Array.from(selector);
    }
    if (Array.isArray(selector) && selector.every(isNode)) {
        return Array.from(selector);
    }
    return [];
}

function dom(selector) {
    let api;
    let elements = toElements(selector);

    api = Object.freeze({
        addClass: function (token) {
            if (typeof token !== "string" || token.trim() === "") {
                return api;
            }
            const classes = token.trim().split(/\s+/);
            elements.forEach(function (el) {
                classes.forEach(function (c) {
                    el.classList.add(c);
                });
            });
            return api;
        },
        addItem: function (val) {
            const newElements = toElements(val);
            elements = elements.concat(newElements);
            return api;
        },
        attr: function (name, value) {
            if (typeof name !== "string") {
                return api;
            }
            if (value === undefined) {
                return elements.map(function (el) {
                    return el.getAttribute(name);
                });
            }
            if (value === null) {
                elements.forEach(function (el) {
                    el.removeAttribute(name);
                });
            } else {
                elements.forEach(function (el) {
                    el.setAttribute(name, String(value));
                });
            }
            return api;
        },
        children: function () {
            const childElements = elements.flatMap(function (el) {
                return Array.from(el.children);
            });
            return dom(childElements);
        },
        clone: function (deep) {
            const isDeep = (deep !== false);
            const cloned = elements.map(function (el) {
                return el.cloneNode(isDeep);
            });
            return dom(cloned);
        },
        css: function (name, value) {
            if (typeof name !== "string") {
                return api;
            }
            const camelName = camelCase(name);
            const kebabName = kebabCase(name);
            if (value === undefined) {
                return elements.map(function (el) {
                    if (
                        globalThis !== undefined
                        && typeof globalThis.getComputedStyle === "function"
                    ) {
                        const computed = globalThis.getComputedStyle(el);
                        return computed.getPropertyValue(kebabName) || "";
                    }
                    return "";
                });
            }
            if (value === null) {
                elements.forEach(function (el) {
                    el.style.removeProperty(kebabName);
                });
            } else {
                elements.forEach(function (el) {
                    el.style[camelName] = String(value);
                });
            }
            return api;
        },
        data: function (name, value) {
            if (typeof name !== "string") {
                return api;
            }
            if (value === undefined) {
                return elements.map(function (el) {
                    return el.dataset[name];
                });
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
        each: function (fn) {
            if (typeof fn !== "function") {
                return api;
            }
            elements.forEach(function (el, index) {
                fn(el, index);
            });
            return api;
        },
        get: function (index) {
            if (typeof index === "number") {
                if (index >= 0 && index < elements.length) {
                    return elements[index];
                }
                return null;
            }
            return Array.from(elements);
        },
        getClassName: function () {
            return elements.map(function (el) {
                return el.className;
            });
        },
        hasClass: function (token) {
            if (typeof token !== "string" || token.trim() === "") {
                return false;
            }
            return elements.every(function (el) {
                return el.classList.contains(token);
            });
        },
        html: function (value) {
            if (value === undefined) {
                return elements.map(function (el) {
                    return el.innerHTML;
                });
            }
            if (typeof value !== "string") {
                return api;
            }
            elements.forEach(function (el) {
                el.innerHTML = value;
            });
            return api;
        },
        length: function () {
            return elements.length;
        },
        next: function () {
            const nextElements = elements.map(function (el) {
                return el.nextElementSibling;
            }).filter(Boolean);
            return dom(nextElements);
        },
        off: function (type, fn, capture) {
            const isCapture = (capture === true);
            if (!type) {
                elements.forEach(function (el) {
                    const register = eventRegistry.get(el);
                    if (register) {
                        Object.keys(register).forEach(function (eventType) {
                            register[eventType].forEach(function (item) {
                                el.removeEventListener(
                                    eventType,
                                    item.fn,
                                    item.capture
                                );
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
                        register[type].forEach(function (item) {
                            el.removeEventListener(
                                type,
                                item.fn,
                                item.capture
                            );
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
                        register[type] = register[type].filter(
                            function (item) {
                                if (
                                    item.fn === fn
                                    && item.capture === isCapture
                                ) {
                                    el.removeEventListener(
                                        type,
                                        item.fn,
                                        item.capture
                                    );
                                    return false;
                                }
                                return true;
                            }
                        );
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
        on: function (type, fn, capture) {
            const isCapture = (capture === true);
            if (typeof type !== "string" || typeof fn !== "function") {
                return api;
            }
            elements.forEach(function (el) {
                let register = eventRegistry.get(el);
                if (!register) {
                    register = Object.create(null);
                    eventRegistry.set(el, register);
                }
                if (!register[type]) {
                    register[type] = [];
                }
                register[type].push({
                    capture: isCapture,
                    fn
                });
                el.addEventListener(type, fn, isCapture);
            });
            return api;
        },
        once: function (type, fn, capture) {
            const isCapture = (capture === true);
            if (typeof type !== "string" || typeof fn !== "function") {
                return api;
            }
            elements.forEach(function (el) {
                let wrapper;
                wrapper = function (event) {
                    el.removeEventListener(type, wrapper, isCapture);
                    const register = eventRegistry.get(el);
                    if (register && register[type]) {
                        register[type] = register[type].filter(
                            function (item) {
                                return item.fn !== wrapper;
                            }
                        );
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
                    register = Object.create(null);
                    eventRegistry.set(el, register);
                }
                if (!register[type]) {
                    register[type] = [];
                }
                register[type].push({
                    capture: isCapture,
                    fn: wrapper
                });
                el.addEventListener(type, wrapper, isCapture);
            });
            return api;
        },
        parents: function () {
            const parentElements = elements.map(function (el) {
                return el.parentElement;
            }).filter(Boolean);
            return dom(parentElements);
        },
        prev: function () {
            const prevElements = elements.map(function (el) {
                return el.previousElementSibling;
            }).filter(Boolean);
            return dom(prevElements);
        },
        remove: function () {
            elements.forEach(function (el) {
                if (el.parentNode !== null) {
                    el.parentNode.removeChild(el);
                }
            });
            return api;
        },
        removeClass: function (token) {
            if (typeof token !== "string" || token.trim() === "") {
                return api;
            }
            const classes = token.trim().split(/\s+/);
            elements.forEach(function (el) {
                classes.forEach(function (c) {
                    el.classList.remove(c);
                });
            });
            return api;
        },
        removeItem: function (index) {
            if (
                typeof index === "number"
                && index >= 0
                && index < elements.length
            ) {
                elements.splice(index, 1);
            }
            return api;
        },
        select: function (token) {
            if (typeof token !== "string") {
                return api;
            }
            const found = [];
            elements.forEach(function (el) {
                const target = el.querySelector(token);
                if (target !== null && !found.includes(target)) {
                    found.push(target);
                }
            });
            return dom(found);
        },
        selectAll: function (token) {
            if (typeof token !== "string") {
                return api;
            }
            const found = [];
            elements.forEach(function (el) {
                const targets = Array.from(el.querySelectorAll(token));
                targets.forEach(function (target) {
                    if (!found.includes(target)) {
                        found.push(target);
                    }
                });
            });
            return dom(found);
        },
        siblings: function () {
            const sibElements = [];
            elements.forEach(function (el) {
                if (
                    el.parentElement !== null
                    && el.parentElement !== undefined
                ) {
                    const children = Array.from(el.parentElement.children);
                    children.forEach(function (sib) {
                        if (sib !== el && !sibElements.includes(sib)) {
                            sibElements.push(sib);
                        }
                    });
                }
            });
            return dom(sibElements);
        },
        text: function (value) {
            if (value === undefined) {
                return elements.map(function (el) {
                    return el.textContent;
                });
            }
            if (typeof value !== "string") {
                return api;
            }
            elements.forEach(function (el) {
                el.textContent = value;
            });
            return api;
        },
        toggleClass: function (token, force) {
            if (typeof token !== "string" || token.trim() === "") {
                return api;
            }
            const classes = token.trim().split(/\s+/);
            const hasForce = (typeof force === "boolean");
            elements.forEach(function (el) {
                classes.forEach(function (c) {
                    if (hasForce) {
                        el.classList.toggle(c, force);
                    } else {
                        el.classList.toggle(c);
                    }
                });
            });
            return api;
        }
    });

    return api;
}

dom.create = Object.freeze(function (tag) {
    if (typeof tag !== "string") {
        return dom();
    }
    const cleanTag = tag.toLowerCase();
    if (!VALID_TAGS.has(cleanTag)) {
        return dom();
    }
    if (document !== undefined) {
        return dom(document.createElement(cleanTag));
    }
    return dom();
});

export default Object.freeze(dom);
