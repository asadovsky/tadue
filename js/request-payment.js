// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

// Trick JSLint. These vars are defined elsewhere.
// TODO(sadovsky): Refactor to more cleanly share common components.
var checkEmailField, checkAmountField, checkDescriptionField, runChecks;
var runSignupChecks, runLoginChecks;

// If true, the signup/login part of the form will be hidden and should not be
// checked.
var loggedIn = $('#logged-in').length === 1;

var showSignup = function () {
  $('#signup-box').css('display', 'block');
  $('#login-box').css('display', 'none');
  $('#new-user').addClass('active-tab');
  $('#existing-user').removeClass('active-tab');
  $('#do-signup').val('true');
};

var showLogin = function () {
  $('#signup-box').css('display', 'none');
  $('#login-box').css('display', 'block');
  $('#new-user').removeClass('active-tab');
  $('#existing-user').addClass('active-tab');
  $('#do-signup').val('false');
};

// Maps element id to check function.
var requestPaymentChecks = {};
(function () {
  requestPaymentChecks['payer-email'] = checkEmailField;
  requestPaymentChecks['amount'] = checkAmountField;
  requestPaymentChecks['description'] = checkDescriptionField;
}());

var runAllChecks = function () {
  var valid = runChecks(requestPaymentChecks);
  // Always run the signup and login checks to ensure that all error messages
  // stay up to date.
  var signupChecksValid = runSignupChecks();
  var loginChecksValid = runLoginChecks();
  if (!loggedIn) {
    if ($('#do-signup').val() === 'true') {
      valid = signupChecksValid && valid;
    } else {
      valid = loginChecksValid && valid;
    }
  }
  return valid;
};

// Run checks when button is pressed, and on every input event thereafter.
var runAllChecksOnEveryInputEvent = false;
var checkRequestPaymentForm = function () {
  if (!runAllChecksOnEveryInputEvent) {
    runAllChecksOnEveryInputEvent = true;
    $('input').each(function (index, el) {
      el.addEventListener('input', runAllChecks, false);
    });
    $('#signup-copy-email').click(function () { runAllChecks(); });
  }
  return runAllChecks();
};

// Initialize the view. Handles the case where user clicked the back button.
$('#new-user').click(showSignup);
$('#existing-user').click(showLogin);
if ($('#do-signup').val() === 'true') {
  showSignup();
} else {
  showLogin();
}
