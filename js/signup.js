'use strict';

goog.provide('tadue.signup');

goog.require('tadue.form');

tadue.signup.runChecks = function() {
  var checks = {};
  checks['#signup-name'] = tadue.form.checkFullNameField;
  checks['#signup-email'] = tadue.form.checkEmailField;
  checks['#signup-password'] = tadue.form.checkPasswordField;

  var checkPayPalEmailField = function(node) {
    if ($('#signup-copy-email').get(0).checked) {
      return '';
    }
    return tadue.form.checkEmailField(node);
  };
  checks['#signup-paypal-email'] = checkPayPalEmailField;

  var checkConfirmPasswordFieldClosure = function(node) {
    return tadue.form.checkConfirmPasswordField(node, '#signup-password');
  };
  checks['#signup-confirm-password'] = checkConfirmPasswordFieldClosure;
  return tadue.form.runChecks(checks);
};

// Run checks when button is pressed, and on every input event thereafter.
tadue.signup.runChecksOnEveryInputEvent = false;
tadue.signup.checkForm = function() {
  if (!tadue.signup.runChecksOnEveryInputEvent) {
    tadue.signup.runChecksOnEveryInputEvent = true;
    $('input').on('input', tadue.signup.runChecks);
    $('#signup-copy-email').click(function() {
      tadue.signup.runChecks();
    });
  }
  return tadue.signup.runChecks();
};

tadue.signup.maybeCopyEmail = function() {
  var doCopy = $('#signup-copy-email').is(':checked');
  if (doCopy) {
    $('#signup-paypal-email').val($('#signup-email').val());
  }
  return doCopy;
};

tadue.signup.updateSignupPayPalEmail = function() {
  $('#signup-paypal-email').get(0).disabled = tadue.signup.maybeCopyEmail();
};

tadue.signup.init = function() {
  // Handles the case where user clicked the back button.
  $('#signup-copy-email').click(tadue.signup.updateSignupPayPalEmail);
  tadue.signup.updateSignupPayPalEmail();

  $('#signup-email').on('input', tadue.signup.maybeCopyEmail);
};
