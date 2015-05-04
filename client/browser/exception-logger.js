// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var window = require('global/window');
var UA = window.navigator ? window.navigator.userAgent : '';

module.exports = init;

function init() {
  window.addEventListener('error', onerror);
}

function onerror(event) {
  var ga = window.ga || noop;
  var category = 'JavaScript Error';
  var action = event.message;
  var label = UA + ' \n' + event.error.stack;

  // SEE: http://goo.gl/IIgLdL
  ga('send', 'event', category, action, label);
}

function noop() {

}
