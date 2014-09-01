'use strict';

goog.provide('tadue.resetPassword');

goog.require('tadue.form');

tadue.resetPassword.runChecks = function() {
  var checks = {};
  checks['#email'] = tadue.form.checkEmailField;
  return tadue.form.runChecks(checks);
};

// Run checks when button is pressed, and on every input event thereafter.
tadue.resetPassword.runChecksOnEveryInputEvent = false;
tadue.resetPassword.checkForm = function() {
  if (!tadue.resetPassword.runChecksOnEveryInputEvent) {
    tadue.resetPassword.runChecksOnEveryInputEvent = true;
    $('input').on('input', tadue.resetPassword.runChecks);
  }
  return tadue.resetPassword.runChecks();
};

tadue.resetPassword.init = function() {
};
