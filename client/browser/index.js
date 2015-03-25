// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var _ = require('lodash');
var path = require('path');
var superagent = require('superagent');

var Playground = require('./playground');

_.forEach(document.querySelectorAll('.playground'), function(el) {
  var src = el.getAttribute('data-src');
  console.log('Creating playground', src);

  fetchBundle(src, function(err, bundle) {
    if (err) {
      el.innerHTML = '<div class="error"><p>Playground error.' +
        '<br>Bundle not found: <strong>' + src + '</strong></p></div>';
      return;
    }
    new Playground(el, src, bundle);  // jshint ignore:line
  });
});

function fetchBundle(loc, cb) {
  var basePath = '/bundles';
  console.log('Fetching bundle', loc);
  superagent
    .get(path.join(basePath, loc))
    .accept('json')
    .end(function(err, res) {
      if (err) {
        return cb(err);
      }
      if (res.error) {
        return cb(res.error);
      }
      cb(null, res.body);
    });
}
