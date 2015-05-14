// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// pg-index=1

var inherits = require('util').inherits;
var vanadium = require('vanadium');

var fortune = require('../fortune');

/**
 * 1) Implement a simple fortune service
 */

function FortuneService() {
  // List of fortunes
  this.fortunes = [];
}

inherits(FortuneService, fortune.Fortune);

// Gets a random fortune
FortuneService.prototype.getRandomFortune = function(ctx, serverCall) {
  var numExistingfortunes = this.fortunes.length;
  if(numExistingfortunes === 0) {
    throw new Error('Sorry! No fortune available :(');
  }
  var randomIndex = Math.floor(Math.random() * numExistingfortunes);
  var fortune = this.fortunes[randomIndex];
  console.info('Serving:', fortune);
  return Promise.resolve(fortune);
};

// Adds a new fortune
FortuneService.prototype.addNewFortune = function(ctx, serverCall, fortune) {
  if(!fortune || fortune.trim() === '') {
    throw new Error('Sorry! Can\'t add empty or null fortune!');
  }
  console.info('Adding:', fortune);
  this.fortunes.push(fortune);
  return Promise.resolve();
};

/**
 * 2) Publish the fortune service
 */

var fortuneService = new FortuneService();

// Create a Vanadium runtime using the configuration
vanadium.init().then(function(rt) {
  var server = rt.newServer();
  // Serve the fortune server under a name. Serve returns a Promise object
  server.serve('bakery/cookie/fortune', fortuneService).then(function() {
    console.log('Fortune server serving under: bakery/cookie/fortune');
  }).catch(function(err) {
    console.error('Failed to serve the fortune server because:\n', err);
    process.exit(1);
  });
}).catch(function(err) {
  console.error('Failed to start the fortune server because:\n', err);
  process.exit(1);
});

// Let's add a few fortunes to start with
[
  'The fortune you seek is in another cookie.',
  'Everything will now come your way.',
  'Conquer your fears or they will conquer you.'
].forEach(function(fortune) {
  fortuneService.addNewFortune(null, null, fortune);
});
