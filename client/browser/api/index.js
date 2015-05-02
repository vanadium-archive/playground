// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var debug = require('debug')('api');
var request = require('superagent');
var hyperquest = require('hyperquest');
var parallel = require('run-parallel');
var format = require('format');
var normalize = require('./normalize');
var url = require('url');
var extend = require('xtend');
var prr = require('prr');
var config = require('../config');
var JSONStream = require('./json-stream');
var split = require('split');
var defaults = {
  // Timeout for HTTP requests, 5 secs in milliseconds.
  timeout: 5 * 60 * 1000,
  // Temporarily default to the staging load balancer until bundle lists are
  // available from the API.
  url: 'https://playground-api.staging.v.io',
  debug: false
};

var options = {};

// If options have been defined by a developer via the console use them to
// override the defaults.
if (config('api-url')) {
  options.url = config('api-url');
}

if (config('api-debug')) {
  options.debug = config('api-debug');
}

// Create a singleton instance of the API object which can be shared across
// modules and components.
module.exports = API(options);

function API(options) {
  if (!(this instanceof API)) {
    return new API(options);
  }

  options = options || {};

  var api = this;

  api.options = extend(defaults, options);

  debug('initializing with options: %o', api.options);

  prr(api, '_url', url.parse(api.options.url));
  prr(api, '_pending', []);
}

API.prototype.url = function(options) {
  options = options || {};

  // By default return a formatted url string.
  options = extend({ format: true }, options);

  var api = this;
  var clone = extend(api._url);

  // Append trailing slash if it's missing
  if (! clone.pathname.match(/\/$/)) {
    clone.pathname = clone.pathname + '/';
  }

  // NOTE: paying attention to options.action is temporary until the API gets
  // cleaned up and becomes RESTful making the url templating simpler.
  switch (options.action) {
    case 'create':
      clone.pathname = url.resolve(clone.pathname, 'save');
      break;
    case 'read':
      clone.pathname = url.resolve(clone.pathname, 'load');
      break;
    case 'run':
      clone.pathname = url.resolve(clone.pathname, 'compile');
      break;
  }

  if (options.uuid) {
    clone.query = { id: encodeURIComponent(options.uuid) };
  }

  if (api.options.debug || options.debug) {
    clone.query = clone.query || {};
    clone.query.debug = 1;
  }

  if (options.format) {
    return url.format(clone);
  } else {
    return clone;
  }
};

// Temporary way to list bundles until there is an API endpoint to hit.
API.prototype.bundles = function(callback) {
  var api = this;
  // TODO(jasoncampbell): remove this list once a list API endpoint is
  // available.
  var ids = [
    '_cadcfa075a6ac6d1939d12a64ac6e57bc7256c0422fb5d0690b3d8618779565',
    '_be43fb9b2d03087dfd7c84437fd37dac7f6977d8cac330b9fce6aad94414558',
    '_5385edd72b550c57bee83b100731338c70349ac7354dc4353665a1998fa7c8c',
    '_46f8b66f0e80be00adc6222ac0235b1f8e70183daa64ec5924b14267dc6f0fd'
  ];

  var workers = ids.map(createWorker);

  // Request all ids in parallel.
  parallel(workers, callback);

  function createWorker(id) {
    return worker;

    function worker(cb) {
      api.get(id, cb);
    }
  }
};

API.prototype.get = function(uuid, callback) {
  var api = this;
  var uri = api.url({ uuid: uuid, action: 'read' });

  request
  .get(uri)
  .accept('json')
  .timeout(api.options.timeout)
  .end(onget);

  function onget(err, res, body) {
    if (err) {
      return callback(err);
    }

    if (! res.ok) {
      var message = format('GET %s - %s NOT OK', uri, res.statusCode);
      err = new Error(message);
      return callback(err);
    }

    var data = normalize(res.body);

    callback(null, data);
  }
};

API.prototype.create = function(data, callback) {
  var api = this;
  var uri = api.url({ action: 'create' });

  request
  .post(uri)
  .type('json')
  .accept('json')
  .timeout(api.options.timeout)
  .send(data)
  .end(oncreate);

  function oncreate(err, res) {
    if (err) {
      return callback(err);
    }

    if (! res.ok) {
      var message = format('POST %s - %s NOT OK', uri, res.status);
      err = new Error(message);
      return callback(err);
    }

    var data = normalize(res.body);

    callback(null, data);
  }
};

API.prototype.isPending = function(uuid) {
  return this._pending.indexOf(uuid) >= 0;
};

API.prototype.pending = function(uuid) {
  return this._pending.push(uuid);
};

API.prototype.done = function(uuid) {
  var api = this;
  var start = api._pending.indexOf(uuid);
  var deleteCount = 1;

  return api._pending.splice(start, deleteCount);
};

// Initializes an http request and returns a stream.
//
// TODO(jasoncampbell): Drop the callback API and return the stream
// immediately.
// TODO(jasoncampbell): stop pending xhr
// SEE: https://github.com/vanadium/issues/issues/39
API.prototype.run = function(data, callback) {
  var api = this;
  var uuid = data.uuid;
  var uri = api.url({
        action: 'run',
        debug: true
      });

  if (api.isPending(uuid)) {
    var message = format('%s is already running');
    var err = new Error(message);
    return callback(err);
  }

  api.pending(uuid);

  var options = {
    withCredentials: false,
    headers: {
      'accept': 'application/json',
      'content-type': 'application/json'
    }
  };

  // TODO(jasoncampbell): Consolidate http libraries.
  // TODO(jasoncampbell): Verify XHR timeout logic and handle appropriately.
  var req = hyperquest.post(uri, options);

  req.once('error', callback);

  var stream = JSONStream(); // jshint ignore:line

  stream.on('end', function() {
    api.done(uuid);
  });

  req
  .pipe(split())
  .pipe(stream);

  callback(null, stream);

  var string = JSON.stringify(data);

  req.write(string);
  req.end();
};
