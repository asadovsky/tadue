// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

// Trick JSLint. These vars are defined elsewhere.
// TODO(sadovsky): Refactor to more cleanly share common components.
var checkFullNameField, checkEmailField, runChecks;

// Maps element id to check function.
var checks = {};
(function () {
  checks['name'] = checkFullNameField;
  checks['paypal-email'] = checkEmailField;
}());

var shouldRunAllChecks = false;
var maybeRunAllChecks = function () {
  if (shouldRunAllChecks) {
    return runChecks(checks);
  }
  return true;
};

var enableSave = function (enabled) {
  $('#save').prop('disabled', !enabled);
};

// Run checks when button is pressed, and on every input event thereafter.
var save = function () {
  shouldRunAllChecks = true;
  if (!maybeRunAllChecks()) {
    return;
  }
  // Inputs are valid. Send ajax request to update info.
  var data = {
    'name': $('#name').val(),
    'paypal-email': $('#paypal-email').val()
  };
  var request = $.ajax({
    url: '/settings/update-info',
    type: 'POST',
    data: data,
    dataType: 'html'
  });
  // TODO(sadovsky): Handle ajax failure.
  request.fail(function (jqXHR, textStatus) {
    console.log(jqXHR.responseText);
  });
  shouldRunAllChecks = false;
  enableSave(false);
};

$('#save').click(save);

$('input').each(function (index, el) {
  el.addEventListener('input', maybeRunAllChecks, false);
  el.addEventListener('input', function () { enableSave(true); }, false);
});
