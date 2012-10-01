// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

// Trick JSLint. These vars are defined elsewhere.
// TODO(sadovsky): Refactor to more cleanly share common components.
var checkEmailField, runChecks;

// Maps element id to check function.
var checks = {};
(function () {
  checks['email'] = checkEmailField;
}());

var runAllChecks = function () {
  return runChecks(checks);
};

// Run checks when submit is pressed, and on every input event thereafter.
var runAllChecksOnEveryInputEvent = false;
var checkForm = function () {
  if (!runAllChecksOnEveryInputEvent) {
    runAllChecksOnEveryInputEvent = true;
    $('input').each(function (index, el) {
      el.addEventListener('input', runAllChecks, false);
    });
  }
  return runAllChecks();
};
