"use strict";

import gg from "../gg.js";
import wsframe from "../wsframe.js";

const global = globalThis || window || this;

wsframe.controllers.main = function main(global, view) {
    "use strict";

    var disclaimerContainer = gg(".disclaimer-container"),
        defaultRoutes = [
            "/",
            "/news",
            "/blog",
            "/faq",
            "/rules",
            "/advertise",
            "/press",
            "/about",
            "/feedback",
            "/legal",
            "/contact"
        ];

    function disclaimerCallback(e, node, arg) {
        if (arg.choice === "agree") {
            global.location.hash = arg.path.replace(/\#/g, "").replace(/\/\//g, "");
        }
        disclaimerContainer.addClass("hide");
    }

    function popup(e, node, arg) {
        var path = gg(e.target).data("href") || gg(e.target.parentNode).data("href");

        console.log(path);
        if (defaultRoutes.indexOf(path) !== -1) {
            global.location.hash = path.replace(/\#/g, "").replace(/\/\//g, "");
        } else {
            disclaimerContainer.remClass("hide");
            gg(".close-disclaimer").once("click", disclaimerCallback, false, {choice: "disagree", path: path});
            gg("#agree-to-disclaimer").once("click", disclaimerCallback, false, {choice: "agree", path: path});
        }
    }
    wsframe.hrefListener(popup);
};
