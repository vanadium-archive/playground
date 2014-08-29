var _ = require('lodash');
var path = require('path');
var request = require('superagent');

var EmbeddedPlayground = require('./embedded');

_.forEach(document.querySelectorAll('.playground'), function(el) {
  var srcdir = el.getAttribute('data-srcdir');
  console.log('Creating playground', srcdir);

  fetchBundle(srcdir, function(err, bundle) {
    if (err) {
      el.innerHTML = '<div class="error"><p>Playground initialization error.' +
          ' Bundle not found: ' + srcdir + '</p></div>';
      return;
    }
    var pg = new EmbeddedPlayground(el, bundle.files);  // jshint ignore:line
  });
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
