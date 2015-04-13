// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

module.exports = normalize;

// Takes data from API messages and converts them to appropriate objects
// (lowecase, correct values, etc.).
function normalize(data) {
  // convert `data.Timestamp` nanosecond value to a float in milliseconds.
  var oneMillion = 1e6;
  var timestamp = data.Timestamp / oneMillion;

  return {
    message: data.Message,
    file: data.File,
    stream: data.Stream,
    timestamp: timestamp
  };
}
