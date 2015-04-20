// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

module.exports = FollowConsoleHook;

// # FollowConsoleHook(state.follow)
//
// Used to hook into the vdom cycle so that DOM attributes can be accessed
// and manipulated causing the DOM node to scroll to the bottom.
function FollowConsoleHook(shouldFollow){
  if (!(this instanceof FollowConsoleHook)) {
    return new FollowConsoleHook(shouldFollow);
  }

  this.track = shouldFollow;
}

FollowConsoleHook.prototype.hook = function(node, property) {
  if (! this.track) {
    return;
  }

  // Do not mutate the DOM node until it has been inserted.
  process.nextTick(update);

  function update() {
    var visibleHeight = node.clientHeight;
    var overflow = node.scrollHeight - visibleHeight;

    // Scroll the overflowing content into view.
    node.scrollTop = overflow;
  }
};
