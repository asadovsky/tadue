// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

goog.provide('tadue.changePassword');

goog.require('tadue.form');

tadue.changePassword.runChecks = function() {
  var checks = {};
  if (document.getElementById('current-password') !== null) {
    checks['#current-password'] = tadue.form.checkPasswordField;
  }
  checks['#new-password'] = tadue.form.checkPasswordField;

  var checkConfirmPasswordFieldClosure = function(node) {
    return tadue.form.checkConfirmPasswordField(node, '#new-password');
  };
  checks['#confirm-password'] = checkConfirmPasswordFieldClosure;

  return tadue.form.runChecks(checks);
};

// Run checks when button is pressed, and on every input event thereafter.
tadue.changePassword.runChecksOnEveryInputEvent = false;
tadue.changePassword.checkForm = function() {
  if (!tadue.changePassword.runChecksOnEveryInputEvent) {
    tadue.changePassword.runChecksOnEveryInputEvent = true;
    $('input').on('input', tadue.changePassword.runChecks);
  }
  return tadue.changePassword.runChecks();
};

tadue.changePassword.init = function() {
};
