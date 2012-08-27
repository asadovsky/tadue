// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

var drawLogo = function () {
  var canvas = document.getElementById('clogo');

  canvas.width = $('#logo').width();
  canvas.height = $('#logo').height();

  var ctx = canvas.getContext('2d');

  ctx.fillStyle = '#448';
  ctx.fillRect(0, 0, 1000, 1000);
  ctx.fill();

  ctx.font = '60px Alice';
  ctx.fillStyle = '#fff';
  ctx.textAlign = 'left';
  ctx.textBaseline = 'alphabetic';
  ctx.fillText('tadue', 0, canvas.height - 1);

  var imgSrc = canvas.toDataURL('image/png');
  console.log(imgSrc);
  $('#ilogo').attr('src', imgSrc);
};

// Add a delay to allow font to load.
setTimeout(drawLogo, 200);
