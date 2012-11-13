// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

// Trick JSLint. These vars are defined elsewhere.
// TODO(sadovsky): Refactor to more cleanly share common components.
var checkFullNameField, checkEmailField, runChecks;

// Maps element id to check function.
var checks = {};
(function () {
  checks['name'] = checkFullNameField;
  checks['paypal-email'] = checkEmailField;
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
  el.addEventListener('input', function () {
    $('#save').prop('disabled', false);
    $('#cancel').prop('disabled', false);
  }, false);
});

$('#cancel').click(function () { window.location.reload(); });
