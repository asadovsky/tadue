// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

// If true, the signup/login part of the form will be missing.
var loggedIn = $('#account-box').length === 0;

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

if (!loggedIn) {
  // Note: We initialize default state in html and css.
  $('#new-user').click(showSignup);
  $('#existing-user').click(showLogin);
}

// Trick JSLint. These vars are actually defined in signup.js.
// TODO(sadovsky): Refactor to more cleanly share common components.
var checkField, checkEmailField, checkAmountField, checkDescriptionField, checkSignupForm;

var checkForm = function () {
  var valid = checkField('payer-email', checkEmailField);
  valid = checkField('amount', checkAmountField) && valid;
  valid = checkField('description', checkDescriptionField) && valid;
  // If user is not logged in and the signup form (as opposed to the login form)
  // is showing, check the signup form.
  if (!loggedIn && $('#do-signup').val() === 'true') {
    valid = checkSignupForm() && valid;
  }
  return valid;
};
