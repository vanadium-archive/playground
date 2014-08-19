var _ = require('lodash');
var path = require('path');
var request = require('superagent');

var editor = require('./editor');
var EmbeddedPlayground = require('./embedded');

// NOTE(sadovsky): We considered both Ace and CodeMirror, but CodeMirror appears
// to be incompatible with Bootstrap and has other weird issues, e.g. its
// injected editor divs can obscure other DOM elements.

_.forEach(document.querySelectorAll('.playground'), function(el) {
  var srcdir = el.getAttribute('data-srcdir');
  console.log('Creating playground', srcdir);

  fetchBundle(srcdir, function(err, bundle) {
    if (err) {
      el.innerText = 'ERROR! Bundle not found: ' + srcdir;
      return;
    }

    var pg = new EmbeddedPlayground(el, bundle.files);  // jshint ignore:line
  });
});

// Temporary, for testing.
_.forEach(document.querySelectorAll('.vanilla-editor'), function(el) {
  editor.mount(el, 'go', 'fmt.Println("Hello normal editor")');
});

function fetchBundle(loc, cb) {
  var basePath = '/guides/code/';
  console.log('Fetching bundle', loc);
  request
    .get(path.join(basePath, loc, 'bundle.json'))
    .accept('json')
    .end(function(err, res) {
      if (err) {
        return cb(err);
      }
      if (res.error) {
        return cb(res.error);
      }
      cb(null, res.body);
    });
}
