var _ = require('lodash');

var editor = require('./editor');
var EmbeddedPlayground = require('./embedded');

// NOTE(sadovsky): We considered both Ace and CodeMirror, but CodeMirror appears
// to be incompatible with Bootstrap and has other weird issues, e.g. its
// injected editor divs can obscure other DOM elements.

_.forEach(document.querySelectorAll('.playground'), function(el) {
  var srcdir = el.getAttribute('data-srcdir');
  console.log('Creating playground', srcdir);
  var files;
  if (srcdir === 'foo') {
    files = [
      {name: 'hello.go', text: 'fmt.Println("Hello ' + srcdir + '")'},
      {name: 'goodbye.go', text: 'fmt.Println("Goodbye ' + srcdir + '")'}
    ];
  } else {
    files = [
      {name: 'hello.js', text: 'console.log(\'Hello ' + srcdir + '\')'},
      {name: 'goodbye.js', text: 'console.log(\'Goodbye ' + srcdir + '\')'}
    ];
  }
  var pg = new EmbeddedPlayground(el, files);  // jshint ignore:line
  return;
});

// Temporary, for testing.
_.forEach(document.querySelectorAll('.vanilla-editor'), function(el) {
  editor.mount(el, 'go', 'fmt.Println("Hello normal editor")');
});
