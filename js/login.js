'use strict';

goog.provide('tadue.login');

goog.require('tadue.form');

tadue.login.runChecks = function() {
  var checks = {};
  checks['#login-email'] = tadue.form.checkEmailField;
  checks['#login-password'] = tadue.form.checkPasswordField;
  return tadue.form.runChecks(checks);
};

// Run checks when button is pressed, and on every input event thereafter.
tadue.login.runChecksOnEveryInputEvent = false;
tadue.login.checkForm = function() {
  if (!tadue.login.runChecksOnEveryInputEvent) {
    tadue.login.runChecksOnEveryInputEvent = true;
    $('input').on('input', tadue.login.runChecks);
  }
  return tadue.login.runChecks();
};

tadue.login.init = function() {
};
