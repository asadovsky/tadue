// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

// Trick JSLint. These vars are defined elsewhere.
// TODO(sadovsky): Refactor to more cleanly share common components.
var checkEmailField, checkPasswordField, runChecks;

// Maps element id to check function.
var loginChecks = {};
(function () {
  loginChecks['login-email'] = checkEmailField;
  loginChecks['login-password'] = checkPasswordField;
}());

var runLoginChecks = function () {
  return runChecks(loginChecks);
};

// Run checks when button is pressed, and on every input event thereafter.
var runLoginChecksOnEveryInputEvent = false;
var checkLoginForm = function () {
  if (!runLoginChecksOnEveryInputEvent) {
    runLoginChecksOnEveryInputEvent = true;
    $('input').each(function (index, el) {
      el.addEventListener('input', runLoginChecks, false);
    });
  }
  return runLoginChecks();
};
