// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

// Trick JSLint. These vars are defined elsewhere.
// TODO(sadovsky): Refactor to more cleanly share common components.
var checkEmailField, runChecks;

// Maps element id to check function.
var checks = {};
(function () {
  checks['#email'] = checkEmailField;
}());

var shouldRunAllChecks = false;
var maybeRunAllChecks = function () {
  if (shouldRunAllChecks) {
    return runChecks(checks);
  }
  return true;
};

// Run checks when button is pressed, and on every input event thereafter.
var checkForm = function () {
  shouldRunAllChecks = true;
  return maybeRunAllChecks();
};

$('input').each(function (index, el) {
  el.addEventListener('input', maybeRunAllChecks, false);
});
