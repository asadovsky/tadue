// Copyright 2012 Adam Sadovsky. All rights reserved.

'use strict';

var _gaq = _gaq || [];
_gaq.push(['_setAccount', 'UA-32357997-1']);
_gaq.push(['_trackPageview']);

(function () {
  var ga = document.createElement('script');
  ga.type = 'text/javascript';
  ga.async = true;
  ga.src = ('https:' === document.location.protocol ? 'https://ssl' : 'http://www') + '.google-analytics.com/ga.js';
  var s = document.getElementsByTagName('script')[0];
  s.parentNode.insertBefore(ga, s);
}());
