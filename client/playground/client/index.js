var _ = require('lodash');

// TODO(sadovsky): Neither CodeMirror (npm: codemirror or code-mirror) nor Ace
// (npm: brace) seem to work with Browserify. Ace comes close, but the cursor
// gets rendered incorrectly.
//
// NOTE(sadovsky): CodeMirror also appears to be incompatible with Bootstrap,
// and has other weird issues, e.g. its injected editor divs can obscure other
// DOM elements.

var useAce = true;

_.forEach(document.querySelectorAll('.playground'), function(el) {
  var srcdir = el.getAttribute('data-srcdir');
  var value = 'fmt.Println("Hello, playground ' + srcdir + '")';
  var editor = ace.edit(el);
  editor.setFontSize(16);
  var session = editor.getSession();
  //session.setMode('ace/mode/javascript');
  session.setMode('ace/mode/golang');
  session.setUseWorker(false);  // disable warning icons
  session.setValue(value);
});
