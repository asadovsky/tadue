'use strict';

function renderPng(canvasSelector, imgSelector) {
  var imgSrc = $(canvasSelector).get(0).toDataURL('image/png');
  $(imgSelector).attr('src', imgSrc);
}

function initCanvas(canvasSelector, width, height) {
  var canvas = $(canvasSelector).get(0);
  canvas.width = width;
  canvas.height = height;
  return canvas.getContext('2d');
}

//////////////////////////////
// Colors

var BLACK = '#000';
var WHITE = '#fff';
var GRAY = '#777';
var PURPLE = '#66c';
var DARK_PURPLE = '#448';

//////////////////////////////
// Logo

function drawOneLogo(canvasSelector, spanSelector) {
  var span = $(spanSelector);
  var font = [
    span.css('font-weight'), span.css('font-size'), span.css('font-family')
  ].join(' ');
  var ctx = initCanvas(canvasSelector, span.width(), span.height());

  ctx.fillStyle = DARK_PURPLE;
  ctx.fillRect(0, 0, 1000, 1000);
  ctx.fill();

  ctx.font = font;
  ctx.fillStyle = WHITE;
  ctx.textAlign = 'left';
  ctx.textBaseline = 'alphabetic';
  ctx.fillText('tadue', 0, span.height() - 1);
}

function drawLogos() {
  drawOneLogo('#logo', '#css-logo');
  renderPng('#logo', '#ilogo');

  drawOneLogo('#logo-small', '#css-logo-small');
  renderPng('#logo-small', '#ilogo-small');
}

// Add a delay to allow font to load.
setTimeout(drawLogos, 200);

//////////////////////////////
// Icons

var RADIUS = 10;
var DIAMETER = 2 * RADIUS;
var LINE_LENGTH = 12;

function drawLine(ctx, startX, color, vertical) {
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
}

function drawMinus(ctx, startX, color) {
  drawLine(ctx, startX, color, false);
}

function drawPlus(ctx, startX, color) {
  drawLine(ctx, startX, color, false);
  drawLine(ctx, startX, color, true);
}

var X_WIDTH = 9;
var X_LINE_WIDTH = 2;

function drawX(ctx) {
  ctx.strokeStyle = GRAY;
  ctx.lineWidth = X_LINE_WIDTH;
  ctx.beginPath();
  ctx.moveTo(0, 0);
  ctx.lineTo(X_WIDTH, X_WIDTH);
  ctx.moveTo(0, X_WIDTH);
  ctx.lineTo(X_WIDTH, 0);
  ctx.stroke();
}

var FAVICON_WIDTH = 16;

function drawFavicon(ctx) {
  // Fill the background.
  ctx.fillStyle = DARK_PURPLE;
  ctx.fillRect(0, 0, FAVICON_WIDTH, FAVICON_WIDTH);
  // Draw the "t".
  // https://developer.mozilla.org/en-US/docs/Web/Guide/HTML/Canvas_tutorial/Applying_styles_and_colors#A_lineWidth_example
  ctx.strokeStyle = WHITE;
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(3.5, 6);
  ctx.lineTo(3.5, 13);
  ctx.moveTo(2, 8.5);
  ctx.lineTo(6, 8.5);
  ctx.moveTo(4, 13.5);
  ctx.lineTo(6, 13.5);
  ctx.moveTo(6, 12.5);
  ctx.lineTo(7, 12.5);
  ctx.stroke();
}

function makeImages() {
  var ctx = initCanvas('#plus', DIAMETER, DIAMETER);
  drawPlus(ctx, 0, PURPLE);
  renderPng('#plus', '#iplus');

  ctx = initCanvas('#plus-active', DIAMETER, DIAMETER);
  drawPlus(ctx, 0, BLACK);
  renderPng('#plus-active', '#iplus-active');

  ctx = initCanvas('#minus', DIAMETER, DIAMETER);
  drawMinus(ctx, 0, PURPLE);
  renderPng('#minus', '#iminus');

  ctx = initCanvas('#minus-active', DIAMETER, DIAMETER);
  drawMinus(ctx, 0, BLACK);
  renderPng('#minus-active', '#iminus-active');

  ctx = initCanvas('#close', X_WIDTH, X_WIDTH);
  drawX(ctx);
  renderPng('#close', '#iclose');

  ctx = initCanvas('#favicon', FAVICON_WIDTH, FAVICON_WIDTH);
  drawFavicon(ctx);
  renderPng('#favicon', '#ifavicon');  // png and ico have the same format
}

makeImages();
