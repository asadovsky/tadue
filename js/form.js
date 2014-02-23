'use strict';

goog.provide('tadue.form');

tadue.form.emailRegExp = /^\S+@\S+\.\S+$/;
tadue.form.floatRegExp = /^\$?[0-9]+(?:\.[0-9][0-9])?$/;
tadue.form.fullNameRegExp = /^(?:\S+ )+\S+$/;

tadue.form.checkEmailField = function(node) {
  if (!tadue.form.emailRegExp.test(node.val())) {
    return 'Invalid email address';
  }
  return '';
};

tadue.form.checkAmountField = function(node) {
  if (!tadue.form.floatRegExp.test(node.val())) {
    return 'Invalid amount';
  }
  return '';
};

tadue.form.checkPasswordField = function(node) {
  if (node.val().length < 6) {
    return 'Password must be at least 6 characters long';
  }
  return '';
};

tadue.form.checkConfirmPasswordField = function(node, passwordNodeSelector) {
  var passwordNode = $(passwordNodeSelector);
  if (node.val() !== passwordNode.val()) {
    return 'Passwords do not match';
  }
  return '';
};

tadue.form.checkFullNameField = function(node) {
  if (!tadue.form.fullNameRegExp.test(node.val())) {
    return 'Please provide your full name';
  }
  return '';
};

tadue.form.checkDescriptionField = function(node) {
  if (node.val().length === 0) {
    return 'Description must not be empty';
  }
  return '';
};

tadue.form.runChecks = function(checks) {
  var valid = true;
  $.each(checks, function(nodeSelector, check) {
    var node = $(nodeSelector);
    var errorMsg = check(node);
    node.closest('tr').find('.error-msg').text(errorMsg);
    valid = (errorMsg === '') && valid;
  });
  return valid;
};
