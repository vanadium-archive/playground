// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var hg = require('mercury');
var extend = require('xtend');

hg.Delegator().listenTo('scroll');

module.exports = hg.BaseEvent(scroll);

// # scroll(channel, data)
//
// dom-delegator event for scrolling. Currently broadcasts:
//
//     { scrolledToBottom: true || false }
//
// Example:
//
//     h('.scroller', { 'ev-scroll': scroll(channel) })
//
function scroll(event, broadcast) {
  var element = event.target;
  var overflowTop = element.scrollTop;
  var visibleHeight = element.clientHeight;
  var scrolledToBottom = overflowTop + visibleHeight >= element.scrollHeight;

  var data = extend(this.data, {
    scrolledToBottom: scrolledToBottom
  });

  broadcast(data);
}
