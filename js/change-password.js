// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

// Trick JSLint. These vars are defined elsewhere.
// TODO(sadovsky): Refactor to more cleanly share common components.
var checkPasswordField, checkConfirmPasswordField, runChecks;

// If true, the current-password field will be missing and should not be
// checked.
var isPasswordResetRequest = $('#key').length === 1;

// Maps element id to check function.
var checks = {};
(function () {
  if (!isPasswordResetRequest) {
    checks['#current-password'] = checkPasswordField;
  }
  checks['#new-password'] = checkPasswordField;

  var checkConfirmPasswordFieldClosure = function (node) {
    return checkConfirmPasswordField(node, '#new-password');
  };
  checks['#confirm-password'] = checkConfirmPasswordFieldClosure;
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
