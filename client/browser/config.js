// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var window = require('global/window');

module.exports = window.config = config;

var store = window.localStorage || new MemStorage();

// # config(key, value)
//
// Quick method to set configuration values via the developer console. For
// instance to point at a different API url:
//
//     config('api-url', 'http://120.0.0.1:9999')
//
function config(key, value) {
  if (typeof value === 'undefined') {
    return get(key);
  } else {
    return set(key, value);
  }
}

function get(key) {
  return store.getItem(key);
}

function set(key, value) {
  store.setItem(key, value);
}

// Stubbed out localStorage API for running tests.
function MemStorage() {
  this.store = {};
}

MemStorage.prototype.getItem = function(key) {
  if (this.store.hasOwnProperty(key)) {
    return this.store[key];
  }
};

MemStorage.prototype.setItem = function(key, value) {
  this.store[key] = value;
};
