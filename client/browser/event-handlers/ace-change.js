// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var hg = require('mercury');
var extend = require('xtend');

// Tell the singleton Delegator to listen to the custom 'ace-change' event.
hg.Delegator().listenTo('ace-change');

module.exports = hg.BaseEvent(change);

// # change(channel, data)
//
// Custom change event for the Ace editor which broadcasts updated content of
// an editor on change.
function change(event, broadcast) {
  // The ProxyEvent object in dom-delegator doesn't set event.detail for
  // custom events until my PR is merged and a new version is attached to
  // mercury.
  //
  // TODO(jasoncampbell): Send a path upstream
  var detail = event.detail || event._rawEvent.detail;
  var data = extend(this.data, detail);

  broadcast(data);
}
