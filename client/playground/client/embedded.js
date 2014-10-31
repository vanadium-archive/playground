module.exports = EmbeddedPlayground;

var _ = require('lodash');
var mercury = require('mercury');
var moment = require('moment');
var path = require('path');
var request = require('superagent');
var url = require('url');

var Editor = require('./editor');

var m = mercury;
var h = mercury.h;

// Shows each file in a tab.
// * el: The DOM element to mount on.
// * id: Identifier for this playground instance, used in debug messages.
// * files: List of {name, body}, as written by bundler.
function EmbeddedPlayground(el, id, files) {
  this.id_ = id;
  this.files_ = _.map(files, function(file) {
    return _.assign({}, file, {
      basename: path.basename(file.name),
      type: path.extname(file.name).substr(1)
    });
  });
  this.editors_ = _.map(this.files_, function(file) {
    return new Editor(file.type, file.body);
  });
  this.state_ = m.struct({
    activeTab: m.value(0),
    nextRunId: m.value(0),
    running: m.value(false),
    hasRun: m.value(false),
    consoleEvents: m.value([])
  });
  mercury.app(el, this.state_, this.render_.bind(this));
}

EmbeddedPlayground.prototype.renderTopBar_ = function(state) {
  var that = this;

  var tabs = _.map(this.files_, function(file, i) {
    var selector = 'div.tab';
    if (i === state.activeTab) {
      selector += '.active';
    }
    return h(selector, {
      'ev-click': function() {
        that.state_.activeTab.set(i);
      }
    }, file.basename);
  });

  var runBtn = h('button.btn', {
    'ev-click': that.run.bind(that)
  }, 'Run');
  var resetBtn = h('button.btn', {
    'ev-click': that.reset.bind(that)
  }, 'Reset');

  return h('div.top-bar', [h('div', tabs), h('div.btns', [runBtn, resetBtn])]);
};

EmbeddedPlayground.prototype.renderEditors_ = function(state) {
  var editors = _.map(this.editors_, function(editor, i) {
    var properties = {};
    if (i !== state.activeTab) {
      // Use "visibility: hidden" rather than "display: none" because the latter
      // causes the editor to initialize lazily and thus flicker when it's first
      // opened.
      properties['style'] = {visibility: 'hidden'};
    }
    return h('div.editor', properties, editor);
  });

  return h('div.editors', editors);
};

function renderConsoleEvent(event) {
  var children = [];
  if (event.Timestamp) {
    // Convert UTC to local time.
    var t = moment(event.Timestamp / 1e6);
    children.push(h('span.timestamp', t.format('H:mm:ss.SSS') + ' '));
  }
  if (event.File) {
    children.push(h('span.filename', path.basename(event.File) + ': '));
  }
  children.push(h('span.' + (event.Stream || 'unknown'), event.Message));
  return h('div', children);
}

EmbeddedPlayground.prototype.renderConsole_ = function(state) {
  if (state.hasRun) {
    return h('div.console.open', [
      h('div.text', _.map(state.consoleEvents, renderConsoleEvent))
    ]);
  }
  return h('div.console');
};

EmbeddedPlayground.prototype.render_ = function(state) {
  return h('div.pg', [
    this.renderTopBar_(state),
    this.renderEditors_(state),
    this.renderConsole_(state)
  ]);
};

// Sends the files to the backend, then injects the response in the console.
EmbeddedPlayground.prototype.run = function() {
  if (this.state_.running()) {
    console.log('Already running', this.id_);
    return;
  }
  var runId = this.state_.nextRunId();

  // TODO(sadovsky): Visually disable the "Run" button or change it to a "Stop"
  // button.
  this.state_.running.set(true);
  this.state_.hasRun.set(true);
  this.state_.consoleEvents.set([{Message: 'Running...'}]);

  var myUrl = url.parse(window.location.href, true);
  var pgaddr = myUrl.query.pgaddr;
  if (pgaddr) {
    console.log('Using pgaddr', pgaddr);
  } else {
    pgaddr = '//staging.v.io/playground';
  }
  var compileUrl = pgaddr + '/compile';
  if (myUrl.query.debug === '1') {
    compileUrl += '?debug=1';
  }

  var editors = this.editors_;
  var req = {
    files: _.map(this.files_, function(file, i) {
      var editor = editors[i];
      return {
        Name: file.name,
        Body: editor.getText()
      };
    }),
    Identities: []
  };

  // TODO(sadovsky): To deal with cached responses, shift timestamps (based on
  // current time) and introduce a fake delay. Also, switch to streaming
  // messages, for usability.
  var that = this, state = this.state_;
  request
      .post(compileUrl)
      .type('json')
      .accept('json')
      .send(req)
      .end(function(err, res) {
        // If the user has stopped this run or reset the playground, do nothing.
        if (runId !== state.nextRunId()) {
          return;
        }
        that.endRun_();
        // TODO(sadovsky): Show system errors to the user somehow.
        if (err) {
          return console.error(err);
        }
        if (res.error) {
          return console.error(res.error);
        }
        if (res.body.Errors) {
          return state.consoleEvents.set([{Message: res.body.Errors}]);
        }
        if (res.body.Events) {
          return state.consoleEvents.set(res.body.Events);
        }
      });
};

// Clears the console and resets all editors to their original contents.
EmbeddedPlayground.prototype.reset = function() {
  this.state_.consoleEvents.set([]);
  _.forEach(this.editors_, function(editor) {
    editor.reset();
  });
  this.endRun_();
  this.state_.hasRun.set(false);
};

EmbeddedPlayground.prototype.endRun_ = function() {
  this.state_.nextRunId.set(this.state_.nextRunId() + 1);
  this.state_.running.set(false);
};
