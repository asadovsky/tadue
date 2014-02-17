// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

var makeImage = function(canvasSelector, imgSelector) {
  var imgSrc = $(canvasSelector).get(0).toDataURL('image/png');
  $(imgSelector).attr('src', imgSrc);
};

//////////////////////////////
// Logo

var drawOneLogo = function(canvasSelector, spanSelector, font) {
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

var drawLogos = function() {
  drawOneLogo('#logo', '#css_logo', '60px Alice');
  makeImage('#logo', '#ilogo');

  drawOneLogo('#logo_small', '#css_logo_small', '40px Alice');
  makeImage('#logo_small', '#ilogo_small');
};

// Add a delay to allow font to load.
setTimeout(drawLogos, 200);

//////////////////////////////
// Icons

var RADIUS = 10;
var DIAMETER = 2 * RADIUS;
var LINE_LENGTH = 12;

var initCanvas = function(canvasSelector, width, height) {
  var canvas = $(canvasSelector).get(0);
  canvas.width = width;
  canvas.height = height;
  return canvas.getContext('2d');
};

var makeLine = function(ctx, startX, color, vertical) {
  ctx.strokeStyle = color;
  ctx.lineWidth = 2;
  ctx.beginPath();
  if (vertical) {
    ctx.moveTo(startX + RADIUS, RADIUS - LINE_LENGTH / 2);
    ctx.lineTo(startX + RADIUS, RADIUS + LINE_LENGTH / 2);
  } else {
    ctx.moveTo(startX + RADIUS - LINE_LENGTH / 2, RADIUS);
    ctx.lineTo(startX + RADIUS + LINE_LENGTH / 2, RADIUS);
  }
  ctx.stroke();
};

var makeMinus = function(ctx, startX, color) {
  makeLine(ctx, startX, color, false);
};

var makePlus = function(ctx, startX, color) {
  makeLine(ctx, startX, color, false);
  makeLine(ctx, startX, color, true);
};

var X_WIDTH = 9;
var X_LINE_WIDTH = 2;
var X_COLOR = '#777';

var makeX = function(ctx) {
  ctx.strokeStyle = X_COLOR;
  ctx.lineWidth = X_LINE_WIDTH;
  ctx.beginPath();
  ctx.moveTo(0, 0);
  ctx.lineTo(X_WIDTH, X_WIDTH);
  ctx.moveTo(0, X_WIDTH);
  ctx.lineTo(X_WIDTH, 0);
  ctx.stroke();
};

var PURPLE = '#66c';
var BLACK = '#000';

var drawIcons = function() {
  var ctx = initCanvas('#plus', DIAMETER, DIAMETER);
  makePlus(ctx, 0, PURPLE);
  makeImage('#plus', '#iplus');

  ctx = initCanvas('#plus-active', DIAMETER, DIAMETER);
  makePlus(ctx, 0, BLACK);
  makeImage('#plus-active', '#iplus-active');

  ctx = initCanvas('#minus', DIAMETER, DIAMETER);
  makeMinus(ctx, 0, PURPLE);
  makeImage('#minus', '#iminus');

  ctx = initCanvas('#minus-active', DIAMETER, DIAMETER);
  makeMinus(ctx, 0, BLACK);
  makeImage('#minus-active', '#iminus-active');

  ctx = initCanvas('#close', X_WIDTH, X_WIDTH);
  makeX(ctx);
  makeImage('#close', '#iclose');
};

drawIcons();
