/**
 * @fileoverview Lightweight, Crockfordian DOM manipulation library.
 * Implements a chainable query selector wrapper adhering to strict
 * functional conventions.
 */

/**
 * Set of valid standard HTML tag names.
 * @type {Set<string>}
 */
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

/**
 * @typedef {Object} EventRegistryEntry
 * @property {boolean} capture - Whether the listener uses capture.
 * @property {Function} fn - Event listener callback function.
 */

/**
 * WeakMap tracking registered event listeners per DOM element.
 * @type {WeakMap<Element, Object.<string, EventRegistryEntry[]>>}
 */
const eventRegistry = new WeakMap();

/**
 * Returns the lowercase object type string.
 * @param {*} obj - Value to check.
 * @returns {string} The lowercase object type representation.
 */
function objectType(obj) {
    if (obj === null) {
        return "null";
    }
    if (obj === undefined) {
        return "undefined";
    }
    return Object.prototype.toString.call(obj).slice(8, -1).toLowerCase();
}

/**
 * Checks if a value is a DOM Node.
 * @param {*} value - Value to check.
 * @returns {boolean} True if the value is a DOM Node, false otherwise.
 */
function isNode(value) {
    return (
        value !== null
        && typeof value === "object"
        && typeof value.nodeName === "string"
        && typeof value.nodeType === "number"
    );
}

/**
 * Converts a hyphen-separated string to camelCase.
 * @param {string} value - Hyphenated string.
 * @returns {string} CamelCase formatted string.
 */
function camelCase(value) {
    if (typeof value !== "string") {
        return value;
    }
    return value.replace(/-([a-z])/g, function (ignore, letter) {
        return letter.toUpperCase();
    });
}

/**
 * Converts a camelCase string to kebab-case.
 * @param {string} value - camelCase string.
 * @returns {string} Kebab-case formatted string.
 */
function kebabCase(value) {
    if (typeof value !== "string") {
        return value;
    }
    return value.replace(/([A-Z])/g, "-$1").toLowerCase();
}

/**
 * Normalizes selectors, nodes, and arrays into an array of DOM Elements.
 * @param {*} selector - Input selector, DOM node, or element collection.
 * @returns {Element[]} Array of normalized DOM Elements.
 */
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

/**
 * @typedef {Object} DomApi
 * @property {(token: string) => DomApi} addClass
 *     Adds one or more space-separated classes to all matched elements.
 * @property {(val: *) => DomApi} addItem
 *     Appends normalized elements to the internal element collection.
 * @property {(name: string, value?: *) => ((string | null)[] | DomApi)} attr
 *     Gets or sets an attribute across all matched elements.
 * @property {() => DomApi} children
 *     Returns a wrapper containing all child elements.
 * @property {(deep?: boolean) => DomApi} clone
 *     Clones all elements in the collection.
 * @property {(name: string, value?: *) => (string[] | DomApi)} css
 *     Gets or sets inline CSS styles on all matched elements.
 * @property {(name: string, value?: *) => (*[] | DomApi)} data
 *     Gets or sets dataset attributes on all matched elements.
 * @property {(fn: (el: Element, index: number) => void) => DomApi} each
 *     Iterates over each element in the collection.
 * @property {(index?: number) => (Element | null | Element[])} get
 *     Retrieves an element by index or returns all elements as an array.
 * @property {() => string[]} getClassName
 *     Returns an array of className values for all matched elements.
 * @property {(token: string) => boolean} hasClass
 *     Determines whether all elements contain the specified class.
 * @property {(value?: string) => (string[] | DomApi)} html
 *     Gets or sets the inner HTML for all matched elements.
 * @property {() => number} length
 *     Returns the number of elements in the collection.
 * @property {() => DomApi} next
 *     Returns next element siblings wrapped in a new DomApi instance.
 * @property {(type?: string, fn?: Function, cap?: boolean) => DomApi} off
 *     Removes registered event listeners from all matched elements.
 * @property {(type: string, fn: Function, cap?: boolean) => DomApi} on
 *     Attaches an event listener to all matched elements.
 * @property {(type: string, fn: Function, cap?: boolean) => DomApi} once
 *     Attaches a one-time event listener to all matched elements.
 * @property {() => DomApi} parents
 *     Returns parent elements wrapped in a new DomApi instance.
 * @property {() => DomApi} prev
 *     Returns previous element siblings wrapped in a new DomApi instance.
 * @property {() => DomApi} remove
 *     Removes all matched elements from the DOM tree.
 * @property {(token: string) => DomApi} removeClass
 *     Removes one or more space-separated classes from all elements.
 * @property {(index: number) => DomApi} removeItem
 *     Removes an element at a given index from the collection.
 * @property {(token: string) => DomApi} select
 *     Finds first descendant matching selector for each element.
 * @property {(token: string) => DomApi} selectAll
 *     Finds all descendants matching selector for each element.
 * @property {() => DomApi} siblings
 *     Returns all sibling elements wrapped in a new DomApi instance.
 * @property {(value?: string) => (*[] | DomApi)} text
 *     Gets or sets the text content for all matched elements.
 * @property {(token: string, force?: boolean) => DomApi} toggleClass
 *     Toggles one or more space-separated classes on all elements.
 */

/**
 * Wraps DOM elements in a chainable API interface.
 * @param {*} [selector] - CSS selector, DOM node, or element collection.
 * @returns {DomApi} Frozen DOM manipulation API.
 */
function dom(selector) {
    let api;
    let elements = toElements(selector);

    api = Object.freeze({
        /**
         * Adds one or more space-separated classes to all elements.
         * @param {string} token - Space-separated class names.
         * @returns {DomApi}
         */
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
        /**
         * Appends normalized elements to the internal collection.
         * @param {*} val - Selector, node, or element array to add.
         * @returns {DomApi}
         */
        addItem: function (val) {
            const newElements = toElements(val);
            elements = elements.concat(newElements);
            return api;
        },
        /**
         * Gets or sets an attribute on all elements in the collection.
         * @param {string} name - Attribute name.
         * @param {*} [value] - Attribute value (removes if null).
         * @returns {(string | null)[] | DomApi}
         */
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
        /**
         * Returns immediate child elements wrapped in a new DomApi.
         * @returns {DomApi}
         */
        children: function () {
            const childElements = elements.flatMap(function (el) {
                return Array.from(el.children);
            });
            return dom(childElements);
        },
        /**
         * Clones all elements in the collection.
         * @param {boolean} [deep=true] - Whether to perform a deep clone.
         * @returns {DomApi}
         */
        clone: function (deep) {
            const isDeep = (deep !== false);
            const cloned = elements.map(function (el) {
                return el.cloneNode(isDeep);
            });
            return dom(cloned);
        },
        /**
         * Gets computed style or sets inline styles on all elements.
         * @param {string} name - CSS property name.
         * @param {*} [value] - Style value (removes if null).
         * @returns {string[] | DomApi}
         */
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
        /**
         * Gets or sets dataset attributes on all elements.
         * @param {string} name - Data attribute key in camelCase.
         * @param {*} [value] - Data value (deletes if null).
         * @returns {(string | undefined)[] | DomApi}
         */
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
        /**
         * Iterates over each element in the collection.
         * @param {function(Element, number): void} fn - Callback function.
         * @returns {DomApi}
         */
        each: function (fn) {
            if (typeof fn !== "function") {
                return api;
            }
            elements.forEach(function (el, index) {
                fn(el, index);
            });
            return api;
        },
        /**
         * Retrieves an element by index or returns all elements.
         * @param {number} [index] - 0-based element index.
         * @returns {Element | null | Element[]}
         */
        get: function (index) {
            if (typeof index === "number") {
                if (index >= 0 && index < elements.length) {
                    return elements[index];
                }
                return null;
            }
            return Array.from(elements);
        },
        /**
         * Returns an array of className strings for all elements.
         * @returns {string[]}
         */
        getClassName: function () {
            return elements.map(function (el) {
                return el.className;
            });
        },
        /**
         * Checks if every element in collection contains a class.
         * @param {string} token - Single class name to check.
         * @returns {boolean}
         */
        hasClass: function (token) {
            if (typeof token !== "string" || token.trim() === "") {
                return false;
            }
            return elements.every(function (el) {
                return el.classList.contains(token);
            });
        },
        /**
         * Gets or sets inner HTML for all elements in collection.
         * @param {string} [value] - HTML string to assign.
         * @returns {string[] | DomApi}
         */
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
        /**
         * Returns the total count of matched elements.
         * @returns {number}
         */
        length: function () {
            return elements.length;
        },
        /**
         * Returns next element siblings wrapped in a new DomApi.
         * @returns {DomApi}
         */
        next: function () {
            const nextElements = elements.map(function (el) {
                return el.nextElementSibling;
            }).filter(Boolean);
            return dom(nextElements);
        },
        /**
         * Removes registered event listeners from elements.
         * @param {string} [type] - Event type name.
         * @param {Function} [fn] - Specific listener callback.
         * @param {boolean} [capture=false] - Whether listener captures.
         * @returns {DomApi}
         */
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
        /**
         * Attaches an event listener to all matched elements.
         * @param {string} type - Event type name.
         * @param {Function} fn - Event listener callback.
         * @param {boolean} [capture=false] - Use capture phase.
         * @returns {DomApi}
         */
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
        /**
         * Attaches a one-time event listener to matched elements.
         * @param {string} type - Event type name.
         * @param {Function} fn - Event listener callback.
         * @param {boolean} [capture=false] - Use capture phase.
         * @returns {DomApi}
         */
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
        /**
         * Returns parent elements wrapped in a new DomApi.
         * @returns {DomApi}
         */
        parents: function () {
            const parentElements = elements.map(function (el) {
                return el.parentElement;
            }).filter(Boolean);
            return dom(parentElements);
        },
        /**
         * Returns previous element siblings wrapped in a new DomApi.
         * @returns {DomApi}
         */
        prev: function () {
            const prevElements = elements.map(function (el) {
                return el.previousElementSibling;
            }).filter(Boolean);
            return dom(prevElements);
        },
        /**
         * Removes all matched elements from their parent nodes.
         * @returns {DomApi}
         */
        remove: function () {
            elements.forEach(function (el) {
                if (el.parentNode !== null) {
                    el.parentNode.removeChild(el);
                }
            });
            return api;
        },
        /**
         * Removes one or more space-separated classes from elements.
         * @param {string} token - Space-separated class names.
         * @returns {DomApi}
         */
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
        /**
         * Removes element at a given index from internal collection.
         * @param {number} index - 0-based index to remove.
         * @returns {DomApi}
         */
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
        /**
         * Finds first descendant matching selector for each node.
         * @param {string} token - CSS selector.
         * @returns {DomApi}
         */
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
        /**
         * Finds all descendants matching selector for each node.
         * @param {string} token - CSS selector.
         * @returns {DomApi}
         */
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
        /**
         * Returns sibling elements wrapped in a new DomApi instance.
         * @returns {DomApi}
         */
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
        /**
         * Gets or sets textContent for all elements in collection.
         * @param {string} [value] - Text string to assign.
         * @returns {(string | null)[] | DomApi}
         */
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
        /**
         * Toggles one or more space-separated classes on elements.
         * @param {string} token - Space-separated class names.
         * @param {boolean} [force] - Force add (true) or remove (false).
         * @returns {DomApi}
         */
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

/**
 * Creates and wraps a newly constructed HTML element.
 * @param {string} tag - Valid HTML element tag name.
 * @returns {DomApi} Frozen DOM manipulation API.
 */
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
