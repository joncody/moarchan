import dom from "../dom.js";
import frame from "../frame.js";

frame.controllers.main = function main(global) {
    function agreeDisclaimer(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        dom(".disclaimer-container").addClass("hide");
    }

    function closeDisclaimer(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        dom(".disclaimer-container").addClass("hide");
    }

    function handleLogout(e) {
        if (e && typeof e.preventDefault === "function") {
            e.preventDefault();
        }
        fetch("/logout", {
            method: "POST",
            headers: {
                "X-Requested-With": "XMLHttpRequest",
                "X-CSRF-Token": frame.getCSRFToken()
            }
        }).then(function () {
            global.location.reload();
        }).catch(function (err) {
            console.error("Logout request failed:", err);
        });
    }

    dom("#agree-to-disclaimer").on("click", agreeDisclaimer, false);
    dom(".close-disclaimer").on("click", closeDisclaimer, false);
    dom("[data-action='logout']").on("click", handleLogout, false);

    return function cleanup() {
        dom("#agree-to-disclaimer").off("click");
        dom(".close-disclaimer").off("click");
        dom("[data-action='logout']").off("click");
    };
};
