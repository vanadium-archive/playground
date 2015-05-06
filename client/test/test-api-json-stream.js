// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var test = require('tape');
var jsonStream = require('../browser/api/json-stream');
var Buffer = require('buffer').Buffer;

test('jsonStream() - single entity per write', function(t) {
  t.plan(4);

  var stream = jsonStream();

  stream
  .on('data', function(data) {
    t.same(data, { foo: 'bar' });
  });

  iterate(4, function() {
    stream.write(new Buffer('{ "foo": "bar" }\n'));
  });

  stream.end();
});

test('jsonStream() - empty strings', function(t) {
  t.plan(1);

  var stream = jsonStream();

  stream
  .on('data', function(data) {
    t.same(data, { foo: 'bar' });
  });

  stream.write(new Buffer('{ "foo": "bar" }\n'));
  stream.write(new Buffer(''));

  stream.end();
});

test('jsonStream() - single entry, multiple writes', function(t) {
  t.plan(1);

  var stream = jsonStream();

  stream
  .on('data', function(data) {
    t.same(data, { foo: 'bar', baz: 'qux' });
  });

  stream.write(new Buffer('{ "foo":'));
  stream.write(new Buffer('"ba'));
  stream.write(new Buffer('r", "ba'));
  stream.write(new Buffer('z":'));
  stream.write(new Buffer('"qux" }\n'));
  stream.end();
});

test('jsonStream() - chunk with several entries, then pieces', function(t) {
  t.plan(11);

  var stream = jsonStream();
  var chunk = '';

  stream
  .on('data', function(data) {
    t.same(data, { a: 'b' });
  });

  iterate(10, function() {
    chunk += '{ "a": "b" }\n';
  });

  chunk += '{ "a":';

  stream.write(new Buffer(chunk));
  stream.write(new Buffer(' "b" }\n'));
  stream.end();
});

test('jsonStream() - end with a partial entry', function(t) {
  t.plan(2);

  var stream = jsonStream();

  stream
  .on('end', t.end)
  .on('error', function(err) {
    t.fail('should not error');
  })
  .on('data', function(data) {
    t.same(data, { a: 'b' });
  });

  stream.write('{ "a": "b" }\n');
  stream.write('{ "a": "b" }\n');
  stream.write('{ "a": ');
  stream.end();
});

test('jsonStream() - blank line entries', function(t) {
  t.plan(1);

  var stream = jsonStream();

  stream
  .on('end', t.end)
  .on('error', function(err) {
    t.fail('should not error');
  })
  .on('data', function(data) {
    t.same(data, { a: 'b' });
  });

  stream.write('{ "a": "b" }\n');
  stream.write('');
  stream.end();
});

test('jsonStream() - bad json entry', function(t) {
  var stream = jsonStream();
  var chunk = '{ bad: "json"';

  stream.on('error', function(err) {
    t.ok(err instanceof Error, 'should error');
    t.ok(err instanceof SyntaxError, 'should be a SyntaxError');
    t.end();
  });

  stream.write(new Buffer(chunk + '\n'));
});

function iterate(times, iterator) {
  for (var i = 0; i < times; i++) {
    iterator(i);
  }
}
