// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

var emailRegExp = /^\S+@\S+\.\S+$/;
var floatRegExp = /^\$?[0-9]+(?:\.[0-9][0-9])?$/;
var fullNameRegExp = /^(?:\S+ )+\S+$/;

var setErrorMsg = function (node, errorMsg) {
  node.closest('tr').find('.error-msg').text(errorMsg);
};

var checkEmailField = function (node) {
  if (!emailRegExp.test(node.val())) {
    return 'Invalid email address';
  }
  return '';
};

var checkAmountField = function (node) {
  if (!floatRegExp.test(node.val())) {
    return 'Invalid amount';
  }
  return '';
};

var checkPasswordField = function (node) {
  if (node.val().length < 6) {
    return 'Password must be at least 6 characters long';
  }
  return '';
};

var checkConfirmPasswordField = function (node, passwordNodeSelector) {
  var passwordNode = $(passwordNodeSelector);
  if (node.val() !== passwordNode.val()) {
    return 'Passwords do not match';
  }
  return '';
};

var checkFullNameField = function (node) {
  if (!fullNameRegExp.test(node.val())) {
    return 'Please provide your full name';
  }
  return '';
};

var checkDescriptionField = function (node) {
  if (node.val().length === 0) {
    return 'Description must not be empty';
  }
  return '';
};

var runChecks = function (checks) {
  var valid = true;
  $.each(checks, function (nodeSelector, check) {
    var node = $(nodeSelector);
    var errorMsg = check(node);
    setErrorMsg(node, errorMsg);
    valid = (errorMsg === '') && valid;
  });
  return valid;
};
