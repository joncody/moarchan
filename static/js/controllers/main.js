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

    function agreed() {
        global.location.hash = arg.path.replace(/\#/g, "").replace(/\/\//g, "");
    }
    function disagreed() {
        disclaimerContainer.addClass("hide");
    }

    function popup(e) {
        var path = gg(e.currentTarget).data("href") || gg(e.currentTarget.parentNode).data("href");

        console.log(path);
        if (defaultRoutes.indexOf(path) !== -1) {
            global.location.hash = path.replace(/\#/g, "").replace(/\/\//g, "");
        } else {
            disclaimerContainer.remClass("hide");
        }
    }
    gg(".close-disclaimer").once("click", disagreed);
    gg("#agree-to-disclaimer").once("click", agreed);
    gg("[data-href]").on("click", popup);
};
