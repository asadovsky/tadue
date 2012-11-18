// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

var drawOneLogo = function (id, font) {
  var canvas = $('#c' + id).get(0);

  canvas.width = $('#' + id).width();
  canvas.height = $('#' + id).height();

  var ctx = canvas.getContext('2d');

  ctx.fillStyle = '#448';
  ctx.fillRect(0, 0, 1000, 1000);
  ctx.fill();

  ctx.font = font;
  ctx.fillStyle = '#fff';
  ctx.textAlign = 'left';
  ctx.textBaseline = 'alphabetic';
  ctx.fillText('tadue', 0, canvas.height - 1);

  var imgSrc = canvas.toDataURL('image/png');
  $('#i' + id).attr('src', imgSrc);
};

var drawLogos = function () {
  drawOneLogo('logo', '60px Alice');
  drawOneLogo('logo_small', '40px Alice');
};

// Add a delay to allow font to load.
setTimeout(drawLogos, 200);
