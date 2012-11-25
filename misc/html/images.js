// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

var makeImage = function (canvasSelector, imgSelector) {
  var imgSrc = $(canvasSelector).get(0).toDataURL('image/png');
  $(imgSelector).attr('src', imgSrc);
};

//////////////////////////////
// Logo

var drawOneLogo = function (spanSelector, canvasSelector, font) {
  var canvas = $(canvasSelector).get(0);
  canvas.width = $(spanSelector).width();
  canvas.height = $(spanSelector).height();

  var ctx = canvas.getContext('2d');

  ctx.fillStyle = '#448';
  ctx.fillRect(0, 0, 1000, 1000);
  ctx.fill();

  ctx.font = font;
  ctx.fillStyle = '#fff';
  ctx.textAlign = 'left';
  ctx.textBaseline = 'alphabetic';
  ctx.fillText('tadue', 0, canvas.height - 1);
};

var drawLogos = function () {
  drawOneLogo('#logo', '#clogo', '60px Alice');
  makeImage('#clogo', '#ilogo');

  drawOneLogo('#logo_small', '#clogo_small', '40px Alice');
  makeImage('#clogo_small', '#ilogo_small');
};

// Add a delay to allow font to load.
setTimeout(drawLogos, 200);

//////////////////////////////
// Icons

var RADIUS = 8;
var BORDER_WIDTH = 1;
var LINE_LENGTH = 10;

var initCanvas = function (canvasSelector) {
  var canvas = $(canvasSelector).get(0);
  canvas.width = 2 * RADIUS;
  canvas.height = 2 * RADIUS;
  return canvas.getContext('2d');
};

var makeCircle = function (ctx, color, alpha) {
  ctx.fillStyle = color;
  ctx.beginPath();
  ctx.arc(RADIUS, RADIUS, RADIUS, 0, Math.PI * 2, true);
  ctx.fill();

  var oldAlpha = ctx.globalAlpha;
  ctx.globalAlpha = alpha;
  ctx.fillStyle = '#fff';
  ctx.beginPath();
  ctx.arc(RADIUS, RADIUS, RADIUS - BORDER_WIDTH, 0, Math.PI * 2, true);
  ctx.fill();
  ctx.globalAlpha = oldAlpha;
};

var makeLine = function (ctx, color, vertical) {
  ctx.strokeStyle = color;
  ctx.lineWidth = 2;
  ctx.beginPath();
  if (vertical) {
    ctx.moveTo(RADIUS, RADIUS - LINE_LENGTH / 2);
    ctx.lineTo(RADIUS, RADIUS + LINE_LENGTH / 2);
  } else {
    ctx.moveTo(RADIUS - LINE_LENGTH / 2, RADIUS);
    ctx.lineTo(RADIUS + LINE_LENGTH / 2, RADIUS);
  }
  ctx.stroke();
};

var makeMinus = function (ctx, color) {
  makeLine(ctx, color, false);
};

var makePlus = function (ctx, color) {
  makeLine(ctx, color, false);
  makeLine(ctx, color, true);
};

var FONT_COLOR = '#66c';
var GRAY = '#ddd';
var BLACK = '#000';

var drawIcons = function () {
  var ctx = initCanvas('#plus');
  makePlus(ctx, FONT_COLOR);
  makeImage('#plus', '#iplus');

  ctx = initCanvas('#plus-hover');
  makeCircle(ctx, GRAY, 0);
  makePlus(ctx, FONT_COLOR);
  makeImage('#plus-hover', '#iplus-hover');

  ctx = initCanvas('#plus-click');
  makeCircle(ctx, GRAY, 0);
  makePlus(ctx, BLACK);
  makeImage('#plus-click', '#iplus-click');

  ctx = initCanvas('#minus');
  makeMinus(ctx, FONT_COLOR);
  makeImage('#minus', '#iminus');

  ctx = initCanvas('#minus-hover');
  makeCircle(ctx, GRAY, 0);
  makeMinus(ctx, FONT_COLOR);
  makeImage('#minus-hover', '#iminus-hover');

  ctx = initCanvas('#minus-click');
  makeCircle(ctx, GRAY, 0);
  makeMinus(ctx, BLACK);
  makeImage('#minus-click', '#iminus-click');
};

drawIcons();
