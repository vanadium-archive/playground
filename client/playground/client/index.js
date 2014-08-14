var _ = require('lodash');
var ace = require('brace');
require('brace/mode/javascript');
require('brace/mode/golang');
require('brace/theme/solarized_light');

// TODO(sadovsky): Neither CodeMirror (npm: codemirror or code-mirror) nor Ace
// (npm: brace) seem to work with Browserify. Ace comes close, but the cursor
// gets rendered incorrectly.
//
// NOTE(sadovsky): CodeMirror also appears to be incompatible with Bootstrap,
// and has other weird issues, e.g. its injected editor divs can obscure other
// DOM elements.

_.forEach(document.querySelectorAll('.playground'), function(el) {
  var srcdir = el.getAttribute('data-srcdir');
  var text = 'fmt.Println("Hello, playground ' + srcdir + '")';
  newEditor(el, 'go', text);
});

// Create a new editor and attach it to an element.
// * el = Element where editor will be attached. Can be a DOM node or id name.
// * type = Type of code being displayed. Currently only 'js' or 'go.
// * text = Initial text to embed in editor.
function newEditor(el, type, text) {
  var editor = ace.edit(el);
  editor.setTheme('ace/theme/solarized_light');
  editor.setFontSize(16);

  var session = editor.getSession();
  switch (type) {
    case 'go':
      session.setMode('ace/mode/golang');
      break;
    case 'js':
      session.setMode('ace/mode/javascript');
      break;
    default:
      throw new Error('Language type not supported: ' + type);
  }

  session.setValue(text);

  // Disable syntax checking. The UI is annoying and it only works in JS
  // anyways.
  session.setOption("useWorker", false);

  return editor;
}

