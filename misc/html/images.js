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

var initCanvas = function (canvasSelector, width, height) {
  var canvas = $(canvasSelector).get(0);
  canvas.width = width;
  canvas.height = height;
  return canvas.getContext('2d');
};

var initCircleCanvas = function (canvasSelector) {
  return initCanvas(canvasSelector, 2 * RADIUS, 2 * RADIUS);
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

var X_SIZE = 9;
var X_LINE_WIDTH = 2;
var X_COLOR = '#555';

var makeX = function (ctx) {
  ctx.strokeStyle = X_COLOR;
  ctx.lineWidth = X_LINE_WIDTH;
  ctx.beginPath();
  ctx.moveTo(0, 0);
  ctx.lineTo(X_SIZE, X_SIZE);
  ctx.moveTo(0, X_SIZE);
  ctx.lineTo(X_SIZE, 0);
  ctx.stroke();
};

var FONT_COLOR = '#66c';
var LIGHT_GRAY = '#ddd';
var BLACK = '#000';

var drawIcons = function () {
  var ctx = initCircleCanvas('#plus');
  makePlus(ctx, FONT_COLOR);
  makeImage('#plus', '#iplus');

  ctx = initCircleCanvas('#plus-hover');
  makeCircle(ctx, LIGHT_GRAY, 0);
  makePlus(ctx, FONT_COLOR);
  makeImage('#plus-hover', '#iplus-hover');

  ctx = initCircleCanvas('#plus-click');
  makeCircle(ctx, LIGHT_GRAY, 0);
  makePlus(ctx, BLACK);
  makeImage('#plus-click', '#iplus-click');

  ctx = initCircleCanvas('#minus');
  makeMinus(ctx, FONT_COLOR);
  makeImage('#minus', '#iminus');

  ctx = initCircleCanvas('#minus-hover');
  makeCircle(ctx, LIGHT_GRAY, 0);
  makeMinus(ctx, FONT_COLOR);
  makeImage('#minus-hover', '#iminus-hover');

  ctx = initCircleCanvas('#minus-click');
  makeCircle(ctx, LIGHT_GRAY, 0);
  makeMinus(ctx, BLACK);
  makeImage('#minus-click', '#iminus-click');

  ctx = initCanvas('#close', X_SIZE, X_SIZE);
  makeX(ctx);
  makeImage('#close', '#iclose');
};

drawIcons();
