// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

goog.provide('tadue.settings');

goog.require('tadue.form');

tadue.settings.runChecks = function() {
  var checks = {};
  checks['#name'] = tadue.form.checkFullNameField;
  checks['#paypal-email'] = tadue.form.checkEmailField;
  return tadue.form.runChecks(checks);
};

// Run checks when button is pressed, and on every input event thereafter.
tadue.settings.runChecksOnEveryInputEvent = false
tadue.settings.checkForm = function() {
  if (!tadue.settings.runChecksOnEveryInputEvent) {
    tadue.settings.runChecksOnEveryInputEvent = true;
    $('input').on('input', tadue.settings.runChecks);
  }
  return tadue.settings.runChecks();
};

tadue.settings.init = function() {
  $('input').on('input', function() {
    $('#save').prop('disabled', false);
    $('#cancel').prop('disabled', false);
  });

  $('#cancel').click(function() { window.location.reload(); });
};
