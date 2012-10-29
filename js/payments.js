// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

// Called when user clicks a checkbox or performs an action (e.g. delete), and
// at initialization time.
var updateVisibleState = function () {
  var toggleHighlight = function () {
    $(this).closest('tr').toggleClass('highlight', $(this).is(':checked'));
  };
  $('.checkbox').each(toggleHighlight);
  $('.action-button').prop('disabled', $('.checkbox:checked').size() === 0);
  if ($('.checkbox').size() === 0) {
    $('#master-checkbox').prop('disabled', true);
  } else {
    $('#master-checkbox').prop('checked',
                               $('.checkbox:checked').size() === $('.checkbox').size());
  }
};

// Returns a comma-separated list of request codes for selected rows.
var getSelectedReqCodes = function () {
  var getValue = function () { return $(this).val(); };
  return $('.checkbox:checked').parents().siblings('.row-req-code').map(getValue).get().join(',');
};

var applyActionToReqCodes = function (url, reqCodes, undo) {
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
  request.done(function (data) {
    $('#payments-data').html(data);
    var undoableReqCodes = $('#undoable-req-codes').val();
    if (!undo && undoableReqCodes !== '') {
      $('#undo').css('display', 'inline');
      $('#undo').off('click');  // remove all existing click handlers
      $('#undo').on('click', function () {
        applyActionToReqCodes(url, undoableReqCodes, true);
      });
    } else {
      $('#undo').css('display', 'none');
    }
    updateVisibleState();
  });
  // TODO(sadovsky): Handle ajax failure.
  request.fail(function (jqXHR, textStatus) {
  });
};

var applyAction = function (url) {
  applyActionToReqCodes(url, getSelectedReqCodes(), false);
};

// Called when user clicks "mark as paid" button.
var markAsPaid = function () {
  applyAction('/payments/mark-as-paid');
};

// Called when user clicks "send reminder" button.
var sendReminder = function () {
  applyAction('/payments/send-reminder');
};

// Called when user clicks "delete" button.
var doDelete = function () {
  applyAction('/payments/delete');
};

// Note: We use a global click handler instead of targeting checkbox elements
// because after an action (e.g. delete) is taken, new checkboxes are created,
// and we don't want to bind new event handlers at that point.
var handleClick = function (e) {
  if (!$(e.target).is('input:checkbox')) { return; }
  if ($(e.target).is('#master-checkbox')) {
    $('.checkbox').prop('checked', $('#master-checkbox').is(':checked'));
  }
  updateVisibleState();
};
$(document).click(handleClick);

$('#mark-as-paid').click(markAsPaid);
$('#send-reminder').click(sendReminder);
$('#delete').click(doDelete);

updateVisibleState();
