// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

var emailRegExp = /^\S+@\S+\.\S+$/;
var floatRegExp = /^\$?[0-9]+(?:\.[0-9][0-9])?$/;
var fullNameRegExp = /^(\S+ )+\S+$/;

// Map of nodeId to check function. These are executed on each input event.
var currChecks = {};

var getVal = function (nodeId) {
  return $('#' + nodeId).val();
};

var setErr = function (nodeId, errorMsg) {
  $('#' + nodeId + '-err').text(errorMsg);
};

var clearErr = function (nodeId) {
  $('#' + nodeId + '-err').text('');
};

var checkEmailField = function (nodeId) {
  if (!emailRegExp.test(getVal(nodeId))) {
    setErr(nodeId, 'Invalid email address');
    return false;
  }
  return true;
};

var checkAmountField = function (nodeId) {
  if (!floatRegExp.test(getVal(nodeId))) {
    setErr(nodeId, 'Invalid amount');
    return false;
  }
  return true;
};

var checkPasswordField = function (nodeId) {
  if (getVal(nodeId).length < 6) {
    setErr(nodeId, 'Password must be at least 6 characters long');
    return false;
  }
  return true;
};

var checkFullNameField = function (nodeId) {
  if (!fullNameRegExp.test(getVal(nodeId))) {
    setErr(nodeId, 'Please provide your full name');
    return false;
  }
  return true;
};

var checkDescriptionField = function (nodeId) {
  if (getVal(nodeId).length === 0) {
    setErr(nodeId, 'Description must not be empty');
    return false;
  }
  return true;
};

// TODO(sadovsky): Temporarily highlight field if check fails.
var checkField = function (nodeId, checkFn) {
  if (!checkFn(nodeId)) {
    if (!currChecks[nodeId]) {
      currChecks[nodeId] = function () { checkField(nodeId, checkFn); };
      $('#' + nodeId).get(0).addEventListener('input', currChecks[nodeId], false);
    }
    return false;
  }
  clearErr(nodeId);
  return true;
};

// TODO(sadovsky): Maybe add password confirmation field.
var checkSignupForm = function () {
  var valid = checkField('signup-name', checkFullNameField);
  valid = checkField('signup-email', checkEmailField) && valid;
  if (!$('#signup-copy-email').get(0).checked) {
    valid = checkField('signup-paypal', checkEmailField) && valid;
  }
  valid = checkField('signup-password', checkPasswordField) && valid;
  return valid;
};

var updateSignupPaypal = function () {
  var doCopy = $('#signup-copy-email').is(':checked');
  $('#signup-paypal').get(0).disabled = doCopy;
  if (doCopy) {
    // TODO(sadovsky): Maybe subscribe to value of email field.
    $('#signup-paypal').val('');
    clearErr('signup-paypal');
  } else {
    var checkFn = currChecks['signup-paypal'];
    if (checkFn) {
      checkField('signup-paypal', checkFn);
    }
  }
};

// TODO(sadovsky): We have to check for existence because the request page won't
// have this checkbox in the DOM if the user is already logged in. Ugh.
if ($('#signup-copy-email').length) {
  $('#signup-copy-email').click(updateSignupPaypal);
  updateSignupPaypal();  // if user clicked back button, box might be checked
}
