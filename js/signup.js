// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

// Trick JSLint. These vars are defined elsewhere.
// TODO(sadovsky): Refactor to more cleanly share common components.
var checkFullNameField, checkEmailField, checkPasswordField, checkConfirmPasswordField, runChecks;

// Maps element id to check function.
var signupChecks = {};
(function () {
  signupChecks['signup-name'] = checkFullNameField;
  signupChecks['signup-email'] = checkEmailField;
  signupChecks['signup-password'] = checkPasswordField;

  var checkPayPalEmailField = function (node) {
    if ($('#signup-copy-email').get(0).checked) {
      return true;
    }
    return checkEmailField(node);
  };
  signupChecks['signup-paypal-email'] = checkPayPalEmailField;

  var checkConfirmPasswordFieldClosure = function (node) {
    return checkConfirmPasswordField(node, 'signup-password');
  };
  signupChecks['signup-confirm-password'] = checkConfirmPasswordFieldClosure;
}());

var runSignupChecks = function () {
  return runChecks(signupChecks);
};

// Run checks when button is pressed, and on every input event thereafter.
var runSignupChecksOnEveryInputEvent = false;
var checkSignupForm = function () {
  if (!runSignupChecksOnEveryInputEvent) {
    runSignupChecksOnEveryInputEvent = true;
    $('input').each(function (index, el) {
      el.addEventListener('input', runSignupChecks, false);
    });
    $('#signup-copy-email').click(function () { runSignupChecks(); });
  }
  return runSignupChecks();
};

var updateSignupPayPalEmail = function () {
  var doCopy = $('#signup-copy-email').is(':checked');
  $('#signup-paypal-email').get(0).disabled = doCopy;
  if (doCopy) {
    $('#signup-paypal-email').val($('#signup-email').val());
  }
};

var maybeCopyEmail = function () {
  var doCopy = $('#signup-copy-email').is(':checked');
  if (doCopy) {
    $('#signup-paypal-email').val($('#signup-email').val());
  }
};

// Initialize the view. Handles the case where user clicked the back button.
$('#signup-copy-email').click(updateSignupPayPalEmail);
updateSignupPayPalEmail();

$('#signup-email').get(0).addEventListener('input', maybeCopyEmail, false);
