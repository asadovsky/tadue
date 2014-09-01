'use strict';

goog.provide('tadue.payments');

// Called when user clicks a checkbox or performs an action (e.g. delete), and
// at initialization time.
tadue.payments.updateVisibleState = function() {
  $('.checkbox').each(function() {
    $(this).closest('tr').toggleClass('highlight', $(this).is(':checked'));
  });
  $('.action-button').prop('disabled', $('.checkbox:checked').size() === 0);
  if ($('.checkbox').size() === 0) {
    $('#master-checkbox').prop('disabled', true);
  } else {
    $('#master-checkbox').prop(
      'checked', $('.checkbox:checked').size() === $('.checkbox').size());
  }
  // Make unpaid rows link to their associated payment request pages.
  $('.unpaid').click(function(e) {
    // Do not navigate if click target is checkbox.
    if (!$(e.target).is('input')) {
      window.location = $(this).find('.row-pay-url').text();
    }
  });
};

// Returns a comma-separated list of request codes for selected rows.
tadue.payments.getSelectedReqCodes = function() {
  var getValue = function() { return $(this).val(); };
  return $('.checkbox:checked').parents().siblings('.row-req-code').
    map(getValue).get().join(',');
};

// Stores most recent window.setTimeout() return value.
tadue.payments.timeoutID = null;

tadue.payments.applyActionToReqCodes = function(url, reqCodes, undo) {
  var data = {'reqCodes': reqCodes};
  if (undo) {
    data.undo = null;
  }
  var request = $.ajax({
    url: url,
    type: 'POST',
    data: data,
    dataType: 'html'
  });
  request.done(function(data) {
    if (tadue.payments.timeoutID !== null) {
      window.clearTimeout(tadue.payments.timeoutID);
    }
    $('#payments-data').html(data);
    var undoableReqCodes = $('#undoable-req-codes').val();
    if (!undo && undoableReqCodes !== '') {
      $('#undo').off('click');  // remove all existing click handlers
      $('#undo').on('click', function() {
        tadue.payments.applyActionToReqCodes(url, undoableReqCodes, true);
      });
      $('#undo').css('display', 'inline');
      var removeUndo = function() {
        $('#undo').css('display', 'none');
      };
      // Hide undo link after 30 seconds.
      tadue.payments.timeoutID = window.setTimeout(removeUndo, 30000);
    } else {
      $('#undo').css('display', 'none');
    }
    tadue.payments.updateVisibleState();
  });
  // TODO(sadovsky): Handle ajax failure.
  request.fail(function() {
  });
};

tadue.payments.applyAction = function(url) {
  tadue.payments.applyActionToReqCodes(
    url, tadue.payments.getSelectedReqCodes(), false);
};

// Called when user clicks "mark as paid" button.
tadue.payments.markAsPaid = function() {
  tadue.payments.applyAction('/payments/mark-as-paid');
};

// Called when user clicks "send reminder" button.
tadue.payments.sendReminder = function() {
  tadue.payments.applyAction('/payments/send-reminder');
};

// Called when user clicks "delete" button.
tadue.payments.doDelete = function() {
  tadue.payments.applyAction('/payments/delete');
};

// Note: We use a global click handler instead of targeting checkbox elements
// because after an action (e.g. delete) is taken, new checkboxes are created,
// and we don't want to bind new event handlers at that point.
tadue.payments.handleClick = function(e) {
  if (!$(e.target).is('input:checkbox')) { return; }
  if ($(e.target).is('#master-checkbox')) {
    $('.checkbox').prop('checked', $('#master-checkbox').is(':checked'));
  }
  tadue.payments.updateVisibleState();
};

tadue.payments.init = function() {
  $(document).click(tadue.payments.handleClick);

  $('#mark-as-paid').click(tadue.payments.markAsPaid);
  $('#send-reminder').click(tadue.payments.sendReminder);
  $('#delete').click(tadue.payments.doDelete);

  tadue.payments.updateVisibleState();
};
