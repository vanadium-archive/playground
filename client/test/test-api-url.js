// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var test = require('tape');
var API = require('../browser/api').constructor;
var format = require('format');

test('var api = API({ url: "https://host.tld/" })', function(t) {
  var options = { url: 'https://random-host.tld/' };
  var api = API(options);

  t.test('api.url()', function(t) {
    t.equal(api.url(), options.url);
    t.end();
  });

  t.test('api.url({ uuid: <uuid>, action: "read" })', function(t) {
    var options = {
      uuid: '0E88DE8C-88E3-43CC-BD80-87207CC124FA',
      action: 'read'
    };
    var expected = format('%sload?id=%s', api.url(), options.uuid);

    t.equal(api.url(options), expected);
    t.end();
  });

  t.test('api.url({ action: "create" })', function(t) {
    var options = {
      action: 'create'
    };
    var expected = format('%ssave', api.url());

    t.equal(api.url(options), expected);
    t.end();
  });

  t.test('api.url({ action: "run" })', function(t) {
    var options = {
      action: 'run'
    };
    var expected = format('%scompile', api.url());

    t.equal(api.url(options), expected);
    t.end();
  });
});

test('var api = API({ url: "https://host.tld/api" })', function(t) {
  var options = { url: 'https://random-host.tld/api' };
  var api = API(options);

  t.test('api.url()', function(t) {
    t.equal(api.url(), options.url + '/');
    t.end();
  });

  t.test('api.url({ uuid: <uuid>, action: "read" })', function(t) {
    var options = {
      uuid: '0E88DE8C-88E3-43CC-BD80-87207CC124FA',
      action: 'read'
    };
    var expected = format('%sload?id=%s', api.url(), options.uuid);

    t.equal(api.url(options), expected);
    t.end();
  });

  t.test('api.url({ action: "create" })', function(t) {
    var options = {
      action: 'create'
    };
    var expected = format('%ssave', api.url());

    t.equal(api.url(options), expected);
    t.end();
  });

  t.test('api.url({ action: "run" })', function(t) {
    var options = {
      action: 'run'
    };
    var expected = format('%scompile', api.url());

    t.equal(api.url(options), expected);
    t.end();
  });
});
