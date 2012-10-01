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

// Returns a comma-separated list of request ids for selected rows.
var getSelectedReqIds = function () {
  var getValue = function () { return $(this).val(); };
  return $('.checkbox:checked').parents().siblings('.row-req-id').map(getValue).get().join(',');
};

// TODO(sadovsky): Handle ajax failure.
var applyActionToReqIds = function (url, reqIds, undo) {
  var data = {'reqIds': reqIds};
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
    var undoableReqIds = $('#undoable-req-ids').val();
    if (!undo && undoableReqIds !== '') {
      $('#undo').css('display', 'inline');
      $('#undo').off('click');  // remove all existing click handlers
      $('#undo').on('click', function () {
        applyActionToReqIds(url, undoableReqIds, true);
      });
    } else {
      $('#undo').css('display', 'none');
    }
    updateVisibleState();
  });
  request.fail(function (jqXHR, textStatus) {
  });
};

var applyAction = function (url) {
  applyActionToReqIds(url, getSelectedReqIds(), false);
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
