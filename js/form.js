// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

var emailRegExp = /^\S+@\S+\.\S+$/;
var floatRegExp = /^\$?[0-9]+(?:\.[0-9][0-9])?$/;
var fullNameRegExp = /^(\S+ )+\S+$/;

var getNode = function (nodeId) {
  return $('#' + nodeId);
};

var getErrSpan = function (node) {
  return node.closest('tr').find('.error-msg');
};

var setErr = function (node, errorMsg) {
  getErrSpan(node).text(errorMsg);
};

var clearErr = function (node) {
  getErrSpan(node).text('');
};

var checkEmailField = function (node) {
  if (!emailRegExp.test(node.val())) {
    setErr(node, 'Invalid email address');
    return false;
  }
  return true;
};

var checkAmountField = function (node) {
  if (!floatRegExp.test(node.val())) {
    setErr(node, 'Invalid amount');
    return false;
  }
  return true;
};

var checkPasswordField = function (node) {
  if (node.val().length < 6) {
    setErr(node, 'Password must be at least 6 characters long');
    return false;
  }
  return true;
};

var checkConfirmPasswordField = function (node, passwordNodeId) {
  var passwordNode = getNode(passwordNodeId);
  if (node.val() !== passwordNode.val()) {
    setErr(node, 'Passwords do not match');
    return false;
  }
  return true;
};

var checkFullNameField = function (node) {
  if (!fullNameRegExp.test(node.val())) {
    setErr(node, 'Please provide your full name');
    return false;
  }
  return true;
};

var checkDescriptionField = function (node) {
  if (node.val().length === 0) {
    setErr(node, 'Description must not be empty');
    return false;
  }
  return true;
};

var runChecks = function (checks) {
  var valid = true;
  $.each(checks, function (nodeId, check) {
    var node = getNode(nodeId);
    var checkPassed = check(node);
    if (checkPassed) {
      clearErr(node);
    }
    valid = checkPassed && valid;
  });
  return valid;
};
