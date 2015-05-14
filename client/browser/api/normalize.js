// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

module.exports = normalize;

// TODO: Update once the API returns the correct data structures.
// map old data format to a new one and return a bundle state object.
function normalize(old) {
  var data = JSON.parse(old.data);

  return {
    uuid: old.slug || old.link,
    files: data.files
  };
}
