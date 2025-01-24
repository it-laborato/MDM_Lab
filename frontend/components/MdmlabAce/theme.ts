/* eslint-disable */
// @ts-nocheck
ace.define(
  "ace/theme/mdmlab",
  ["require", "exports", "module", "ace/lib/dom"],
  function (acequire, exports, module) {
    exports.isDark = false;
    exports.cssClass = "ace-mdmlab";
    exports.cssText = require("./theme.css");

    var dom = acequire("../lib/dom");
    dom.importCssString(exports.cssText, exports.cssClass);
  }
);
