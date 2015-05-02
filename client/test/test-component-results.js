// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var test = require('tape');
var results = require('../browser/components/results');
var raf = require('raf');
var event = require('synthetic-dom-events');
var document = require('global/document');
var hg = require('mercury');

test('results state', function(t) {
  var state = results();

  t.equal(state.open(), false);
  t.deepEqual(state.logs(), []);
  t.end();
});

// TODO(jasoncampbell): Refactor all the boilerplate here into some simple test
// helpers and assertion wrappers.
test('toggle open', function(t) {
  var div = document.createElement('div');
  document.body.appendChild(div);

  var state = results();
  var remove = hg.app(div, state, results.render);
  var toggle = document.getElementsByClassName('toggle-display')[0];

  t.equal(state.open(), false);

  toggle.dispatchEvent(event('click'));

  raf(function(){
    var results = document.getElementsByClassName('results')[0];

    t.equal(state.open(), true);
    t.ok(results.className.match('opened'), 'should have opened class');

    // NOTE: maybe reset state here?
    document.body.removeChild(div);
    remove();

    t.end();
  });
});
