// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var debug = require('debug')('widgets:editor');
var path = require('path');
var prr = require('prr');
var window = require('global/window');
var document = require('global/document');

module.exports = AceWidget;

// NOTE(sadovsky): We considered both Ace and CodeMirror, but CodeMirror has
// weird issues, e.g. its injected editor divs can obscure other DOM elements.

// Mercury widget that wraps a code editor around a file object.
function AceWidget(file) {
  if (!(this instanceof AceWidget)) {
    return new AceWidget(file);
  }

  var editor = this;

  // This tells Mercury to treat AceWidget as a mercury/virtual-dom widget not
  // to render it's internals.
  //
  // There will be trouble if the editor.type is ever changed so it is
  // protected here using prr.
  prr(editor, 'type', 'Widget');

  editor.filename = file.name;
  editor.extname = path.extname(file.name).replace('.', '');
  editor.text = file.body;

  debug('initialization %s', editor.filename);
}

// The first time an instance of widget is seen it will have widget.init()
// called by mercury. It is expected to return a dom element.
AceWidget.prototype.init = function() {
  var editor = this;

  debug('init() - %s', editor.filename);

  // Ace does all kinds of weird stuff with the global objects (window,
  // document) so moving the module's initialization here makes it possible to
  // run the tests headlessly in node without throwing errors.
  var ace = require('brace');
  require('brace/mode/javascript');
  require('brace/mode/golang');
  require('brace/theme/monokai');

  var element = document.createElement('div');

  editor.ace = ace.edit(element);
  editor.ace.setTheme('ace/theme/monokai');
  editor.ace.on('change', function(data) {
    var event = new window.CustomEvent('ace-change', {
      detail: {
        name: editor.filename,
        body: editor.ace.getValue()
      }
    });

    // Events are processed synchronously and there is no guarantee that the
    // element or it's parent will be in the DOM at the time this event
    // fires. The `process.nextTick` here ensures that the custom event above
    // is not emitted until after the virtual-dom create, update, insert cycle
    // has finished and the custom event above has a chance to bubble.
    process.nextTick(function dispatch(){
      element.dispatchEvent(event);
    });
  });


  var session = editor.ace.getSession();

  switch (editor.extname) {
    case 'go':
      session.setMode('ace/mode/golang');
      break;
    case 'vdl':
      session.setMode('ace/mode/golang');
      break;
    case 'json':
      session.setMode('ace/mode/javascript');
      break;
    case 'js':
      session.setMode('ace/mode/javascript');
      break;
    default:
      throw new Error('Language type not supported: ' + editor.extname);
  }

  // Tell Ace about the file contents.
  session.setValue(editor.text);

  // Disable syntax checking. The UI is annoying and only works for JS.
  session.setOption('useWorker', false);

  // Ensure that update is called on the first vdom cycle.
  editor.update(null, element);

  return element;
};

AceWidget.prototype.update = function(prev, element) {
  var current = this;

  current.ace = current.ace || prev.ace;
};
