// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

// Trick JSLint. These vars are defined elsewhere.
// TODO(sadovsky): Refactor to more cleanly share common components.
var checkEmailField, checkAmountField, checkDescriptionField, runChecks;
var runSignupChecks, runLoginChecks;

// If true, the signup/login part of the form will be hidden and should not be
// checked.
var LOGGED_IN = $('#logged-in').length === 1;

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

var checkEmailAndAmountFields = function (node) {
  var errorMsg = checkEmailField(node);
  if (errorMsg === '') {
    var amountNode = node.parent().next().children().first();
    errorMsg = checkAmountField(amountNode);
  }
  return errorMsg;
};

var runAllChecks = function () {
  // Maps element id to check function.
  var checks = {};
  $('.payer-email-field').each(function () {
    checks['input[name="' + $(this).attr('name') + '"]'] = checkEmailAndAmountFields;
  });
  checks['#description'] = checkDescriptionField;

  var valid = runChecks(checks);
  // Always run the signup and login checks to ensure that all error messages
  // stay up to date.
  var signupChecksValid = runSignupChecks();
  var loginChecksValid = runLoginChecks();
  if (!LOGGED_IN) {
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

var updateTotal = function () {
  var total = 0;
  $('.amount-field').each(function () {
    var val = $(this).val();
    if (val.indexOf('$') === 0) {
      val = val.substr(1);
    }
    total += Number(val);
  });
  $('#total-field').val(total.toFixed(2));
};

// Event counter, used for assigning field names.
var addPayerEventCount = 0;

// Initialize "add payer" button.
$('#add-payer').click(function () {
  addPayerEventCount++;
  var tr = $(this).closest('tr');
  var clone = tr.clone();
  clone.find('input').val('');
  clone.find('.payer-email-field').attr('name', 'payer-email-' + addPayerEventCount);
  var amount = clone.find('.amount-field');
  amount.attr('name', 'amount-' + addPayerEventCount);
  amount.blur(function () { updateTotal(); });
  var icon = clone.find('.icon');
  icon.addClass('remove-payer');
  icon.removeAttr('id');
  icon.attr('title', 'Remove payer');
  icon.click(function () {
    $(this).closest('tr').remove();
    // Hide the total if there's now only one payer.
    if ($('.icon').length === 1) {
      $('#row-total').css('display', 'none');
    }
    updateTotal();
  });
  clone.insertBefore('#row-total');
  $('#row-total').css('display', 'table-row');
});

$('.amount-field').blur(function () { updateTotal(); });
updateTotal();
