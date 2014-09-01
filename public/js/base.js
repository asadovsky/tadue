'use strict';

goog.provide('tadue.base');

tadue.base.init = function() {
  $('#close-message').click(function() {
    $(this).parent().addClass('display-none');
  });
};
