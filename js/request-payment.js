// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

goog.provide('tadue.requestPayment');

goog.require('goog.ui.ac.ArrayMatcher');
goog.require('goog.ui.ac.AutoComplete');
goog.require('goog.ui.ac.InputHandler');
goog.require('goog.ui.ac.Renderer');
goog.require('tadue.form');
goog.require('tadue.login');
goog.require('tadue.signup');

tadue.requestPayment.showSignup = function() {
  $('#signup-box').css('display', 'block');
  $('#login-box').css('display', 'none');
  $('#new-user').addClass('active-tab');
  $('#existing-user').removeClass('active-tab');
  $('#do-signup').val('true');
};

tadue.requestPayment.showLogin = function() {
  $('#signup-box').css('display', 'none');
  $('#login-box').css('display', 'block');
  $('#new-user').removeClass('active-tab');
  $('#existing-user').addClass('active-tab');
  $('#do-signup').val('false');
};

tadue.requestPayment.checkEmailAndAmountFields = function(node) {
  var errorMsg = tadue.form.checkEmailField(node);
  if (errorMsg === '') {
    var amountNode = node.parent().next().children().first();
    errorMsg = tadue.form.checkAmountField(amountNode);
  }
  return errorMsg;
};

tadue.requestPayment.runChecks = function() {
  var checks = {};
  $('.payer-email-field').each(function() {
    checks['input[name="' + $(this).attr('name') + '"]'] =
      tadue.requestPayment.checkEmailAndAmountFields;
  });
  checks['#description'] = tadue.form.checkDescriptionField;

  var valid = tadue.form.runChecks(checks);
  // Always run the signup and login checks to ensure that all error messages
  // stay up to date.
  var signupChecksValid = tadue.signup.runChecks();
  var loginChecksValid = tadue.login.runChecks();
  // If not logged in, check signup/login part of the form.
  if (document.getElementById('logged-in') === null) {
    if ($('#do-signup').val() === 'true') {
      valid = signupChecksValid && valid;
    } else {
      valid = loginChecksValid && valid;
    }
  }
  return valid;
};

// Run checks when button is pressed, and on every input event thereafter.
tadue.requestPayment.runChecksOnEveryInputEvent = false;
tadue.requestPayment.checkForm = function() {
  if (!tadue.requestPayment.runChecksOnEveryInputEvent) {
    tadue.requestPayment.runChecksOnEveryInputEvent = true;
    $('input').on('input', tadue.requestPayment.runChecks);
    $('#signup-copy-email').click(function() {
      tadue.requestPayment.runChecks();
    });
  }
  return tadue.requestPayment.runChecks();
};

tadue.requestPayment.updateTotal = function() {
  var total = 0;
  $('.amount-field').each(function() {
    var val = $(this).val();
    if (val.indexOf('$') === 0) {
      val = val.substr(1);
    }
    total += Number(val);
  });
  $('#total-field').val(total.toFixed(2));
};

// Event counter, used for assigning field names.
tadue.requestPayment.addPayerEventCount = 0;

// AutoComplete input handler. Global so that we can attach inputs on demand.
tadue.requestPayment.inputHandler = null;

tadue.requestPayment.init = function() {
  tadue.signup.init();
  tadue.login.init();

  // Handles the case where user clicked the back button.
  $('#new-user').click(tadue.requestPayment.showSignup);
  $('#existing-user').click(tadue.requestPayment.showLogin);
  if ($('#do-signup').val() === 'true') {
    tadue.requestPayment.showSignup();
  } else {
    tadue.requestPayment.showLogin();
  }

  // Initialize "add payer" button.
  $('#add-payer').click(function() {
    tadue.requestPayment.addPayerEventCount++;
    var new_tr = $(this).closest('tr').clone();
    new_tr.find('input').val('');
    new_tr.find('.error-msg').text('');
    var email = new_tr.find('.payer-email-field');
    email.attr('name',
               'payer-email-' + tadue.requestPayment.addPayerEventCount);
    if (tadue.requestPayment.inputHandler !== null) {
      tadue.requestPayment.inputHandler.attachInput(email.get(0));
      email.attr('autocomplete', 'off');
    }
    var amount = new_tr.find('.amount-field');
    amount.attr('name', 'amount-' + tadue.requestPayment.addPayerEventCount);
    amount.blur(function() { tadue.requestPayment.updateTotal(); });
    var icon = new_tr.find('.icon');
    icon.addClass('remove-payer');
    icon.removeAttr('id');
    icon.attr('title', 'Remove this payer');
    icon.click(function() {
      $(this).closest('tr').remove();
      // Hide the total if there's now only one payer.
      if ($('.icon').length === 1) {
        $('#row-total').css('display', 'none');
      }
      tadue.requestPayment.updateTotal();
    });
    new_tr.insertBefore('#row-total');
    $('#row-total').css('display', 'table-row');

    if (tadue.requestPayment.runChecksOnEveryInputEvent) {
      new_tr.find('input').on('input', tadue.requestPayment.runChecks);
      tadue.requestPayment.runChecks();
    }
  });

  $('.amount-field').blur(function() { tadue.requestPayment.updateTotal(); });
  tadue.requestPayment.updateTotal();
};

tadue.requestPayment.initAutoComplete = function() {
  var request = $.ajax({
    url: '/get-contacts',
    type: 'POST',
    dataType: 'json'
  });
  request.done(function(contacts) {
    // Modeled after goog.ui.ac.createSimpleAutoComplete.
    var matcher = new goog.ui.ac.ArrayMatcher(contacts, true);
    var renderer = new goog.ui.ac.Renderer();
    tadue.requestPayment.inputHandler =
      new goog.ui.ac.InputHandler(null, null, false);
    var ac = new goog.ui.ac.AutoComplete(
      matcher, renderer, tadue.requestPayment.inputHandler);
    tadue.requestPayment.inputHandler.attachAutoComplete(ac);

    // Attach all existing email inputs.
    $.each($('.payer-email-field').get(), function(i, v) {
      tadue.requestPayment.inputHandler.attachInput(v);
      $(this).attr('autocomplete', 'off');
    });

    // FIXME(sadovsky): Allow "John Doe <email>" format and eliminate this hack.
    ac.selectionHandler_.setValue = function(value) {
      var email = value.substring(value.indexOf('<') + 1, value.length - 1);
      this.activeElement_.value = email;
    };
  });
  request.fail(function() {
  });
};

tadue.requestPayment.openAuthCodeUrl = function() {
  var url = $('#auth-code-url').attr('href');
  var popup = window.open(url, '', 'width=600,height=450');
  if (window.focus) {
    popup.focus();
  }
};

tadue.requestPayment.authDone = function(ok) {
  if (!ok) {
    return;
  }
  $('#auth-code-url').addClass('display-none');
  tadue.requestPayment.initAutoComplete();
};
