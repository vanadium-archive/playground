// Simple abstraction for mounting a code editor on a DOM node.

module.exports = {mount: mount};

var ace = require('brace');
require('brace/mode/javascript');
require('brace/mode/golang');
require('brace/theme/monokai');

// Creates a new editor and mounts it on a DOM node.
// * el: The DOM node to mount on.
// * type: Type of code. Currently either 'js' or 'go'.
// * text: Initial code.
function mount(el, type, text) {
  var editor = ace.edit(el);
  editor.setTheme('ace/theme/monokai');
  editor.setFontSize(15);

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
  session.setOption('useWorker', false);
}
