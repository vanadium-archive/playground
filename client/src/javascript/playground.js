module.exports = Playground;

var _ = require('lodash');
var http = require('http');
var moment = require('moment');
var path = require('path');
var superagent = require('superagent');
var url = require('url');

var m = require('mercury');
var h = m.h;

var Editor = require('./widgets/editor');
var Spinner = require('./widgets/spinner');

// Timeout for save and load requests, in milliseconds.
var storageRequestTimeout = 1000;

// Shows each file in a tab.
// * el: The DOM element to mount on.
// * name: Name of this playground instance, used in debug messages.
// * bundle: The default bundle formatted as {files:[{name, body}]}, as written
//           by pgbundle, to use if id is unspecified or if load fails.
function Playground(el, name, bundle) {
  this.name_ = name;
  this.defaultBundle_ = bundle;
  this.files_ = [];
  this.editors_ = [];
  this.editorSpinner_ = new Spinner();
  // scrollState_ changes should not trigger render_, thus are not monitored
  // by mercury.
  this.scrollState_ = {bottom: true};
  // Mercury framework state. Changes to state trigger virtual DOM render.
  var state = m.state({
    notification: m.value({}),
    // False on page load (no bundle), empty if default bundle is loaded.
    bundleId: m.value(false),
    // Incrementing counter on each bundle reload to force render.
    bundleVersion: m.value(0),
    activeTab: m.value(0),
    idToLoad: m.value(''),
    // Save or load request in progress.
    requestActive: m.value(false),
    nextRunId: m.value(0),
    running: m.value(false),
    hasRun: m.value(false),
    consoleEvents: m.value([]),
    // Mercury framework channels. References to event handlers callable from
    // rendering code.
    channels: {
      switchTab: this.switchTab.bind(this),
      setIdToLoad: this.setIdToLoad.bind(this),
      // For testing. Load in production can be done purely using links.
      // Reload is not necessary before adding history.
      load: this.load.bind(this),
      save: this.save.bind(this),
      run: this.run.bind(this),
      stop: this.stop.bind(this),
      reset: this.reset.bind(this)
    }
  });
  m.app(el, state, this.render_.bind(this));
  // When page is first loaded, load bundle in url.
  this.loadUrl_(state, window.location.href);
  // When user goes forward/back, load bundle in url.
  var that = this;
  window.addEventListener('popstate', function(ev) {
    console.log('window.onpopstate', ev);
    that.loadUrl_(state, window.location.href);
  });
  // Enable ev-scroll listening.
  m.Delegator().listenTo('scroll');
}

// Attempts to load the bundle with id specified in the url, or the default
// bundle if id is not specified.
Playground.prototype.loadUrl_ = function(state, pgurl) {
  this.setNotification_(state);
  this.url_ = url.parse(pgurl, true);
  // Deleted to make url.format() use this.url_.query.
  delete this.url_.search;
  var bId = this.url_.query.id || '';
  // Filled into idToLoad to allow retrying using Load button.
  state.idToLoad.set(bId);
  if (bId) {
    console.log('Loading bundle', bId, 'from URL');
    process.nextTick(this.load.bind(this, state, {id: bId}));
  } else {
    console.log('Loading default bundle');
    process.nextTick(
      this.setBundle_.bind(this, state, this.defaultBundle_, ''));
  }
};

// Builds bundle object from editor contents.
Playground.prototype.getBundle_ = function() {
  var editors = this.editors_;
  return {
    files: _.map(this.files_, function(file, i) {
      var editor = editors[i];
      return {
        name: file.name,
        body: editor.getText()
      };
    })
  };
};

// Loads bundle object into editor contents, updates url.
Playground.prototype.setBundle_ = function(state, bundle, id) {
  this.files_ = _.map(bundle.files, function(file) {
    return _.assign({}, file, {
      basename: path.basename(file.name),
      type: path.extname(file.name).substr(1)
    });
  });
  this.editors_ = _.map(this.files_, function(file) {
    return new Editor(file.type, file.body);
  });
  state.activeTab.set(0);
  this.resetState_(state);
  state.bundleId.set(id);
  // Increment counter in state to force render.
  state.bundleVersion.set((state.bundleVersion() + 1) & 0x7fffffff);
  this.setUrlForId_(id);
};

// Updates url with new id if different from the current one.
Playground.prototype.setUrlForId_ = function(id) {
  if (!id && !this.url_.query.id || id === this.url_.query.id) {
    return;
  }
  if (!id) {
    delete this.url_.query.id;
  } else {
    this.url_.query.id = id;
  }
  window.history.pushState(null, '', url.format(this.url_));
};

// Determines base url for backend calls.
Playground.prototype.getBackendUrl_ = function() {
  var pgaddr = this.url_.query.pgaddr;
  if (pgaddr) {
    console.log('Using pgaddr', pgaddr);
  } else {
    // TODO(ivanpi): Change to detect staging/production. Restrict pgaddr.
    pgaddr = 'https://staging.v.io/playground';
  }
  return pgaddr;
};

// Shows notification, green if success is set, red otherwise.
// Call with undefined/blank msg to clear notification.
Playground.prototype.setNotification_ = function(state, msg, success) {
  if (!msg) {
    state.notification.set({});
  } else {
    state.notification.set({message: msg, ok: success});
    // TODO(ivanpi): Expire message.
  }
};

// Renders button with provided label and target.
// If disable is true or a request is active, the button is disabled.
Playground.prototype.button_ = function(state, label, target, disable) {
  if (disable || state.requestActive) {
    return h('button.btn', {
      disabled: true
    }, label);
  } else {
    return h('button.btn', {
      'ev-click': target
    }, label);
  }
};

Playground.prototype.renderLoadBar_ = function(state) {
  var idToLoad = h('input.bundleid', {
    type: 'text',
    name: 'idToLoad',
    size: 64,
    maxLength: 64,
    value: state.idToLoad,
    'ev-input': m.sendChange(state.channels.setIdToLoad)
  });
  var loadBtn = this.button_(state, 'Load',
    m.sendClick(state.channels.load, {id: state.idToLoad}),
    state.running);

  return h('div.widget-bar', [
    h('span', [idToLoad]),
    h('span.btns', [loadBtn])
  ]);
};

Playground.prototype.renderResetBar_ = function(state) {
  var idShow = h('span.bundleid',
    state.bundleId || (state.bundleId === '' ? '<default>' : '<none>'));
  var link = h('a', {
    href: window.location.href
  }, 'link');
  var notif = h('span.notif.' + (state.notification.ok ? 'success' : 'error'),
    state.notification.message || '');

  var resetBtn = this.button_(state, 'Reset',
    m.sendClick(state.channels.reset),
    state.bundleId === false);
  var reloadBtn = this.button_(state, 'Reload',
    m.sendClick(state.channels.load, {id: state.bundleId}),
    state.running || !state.bundleId);

  return h('div.widget-bar', [
    h('span', [idShow, ' ', link, ' ', notif]),
    h('span.btns', [resetBtn, reloadBtn])
  ]);
};

Playground.prototype.renderTabBar_ = function(state) {
  var tabs = _.map(this.files_, function(file, i) {
    var selector = 'div.tab';
    if (i === state.activeTab) {
      selector += '.active';
    }
    return h(selector, {
      'ev-click': m.sendClick(state.channels.switchTab, {index: i})
    }, file.basename);
  });

  var runStopBtn = state.running ?
    this.button_(state, 'Stop', m.sendClick(state.channels.stop)) :
    this.button_(state, 'Run', m.sendClick(state.channels.run),
      state.bundleId === false);
  var saveBtn = this.button_(state, 'Save',
    m.sendClick(state.channels.save),
    state.running || (state.bundleId === false));

  return h('div.widget-bar', [
    h('span', tabs),
    h('span.btns', [runStopBtn, saveBtn])
  ]);
};

Playground.prototype.renderEditors_ = function(state) {
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

  if (state.requestActive) {
    editors.push(this.editorSpinner_);
  }

  return h('div.editors', editors);
};

Playground.prototype.renderConsoleEvent_ = function(event) {
  var children = [];
  if (event.Timestamp) {
    // Convert UTC to local time.
    var t = moment(event.Timestamp / 1e6);
    children.push(h('span.timestamp', t.format('H:mm:ss.SSS') + ' '));
  }
  if (event.File) {
    children.push(h('span.filename', path.basename(event.File) + ': '));
  }
  // A single trailing newline is always ignored.
  // Ignoring the last character, check if there are any newlines in message.
  if (event.Message.slice(0, -1).indexOf('\n') !== -1) {
    // Multiline messages are marked with U+23CE and started in a new line.
    children.push('\u23ce'/* U+23CE RETURN SYMBOL */, h('br'));
  }
  children.push(h('span.message.' + (event.Stream || 'unknown'),
                  event.Message));
  return h('div', children);
};

// ScrollHandle provides a hook to keep the console scrolled to the bottom
// unless the user has scrolled up, and the update method to detect the
// user scrolling up.
function ScrollHandle(scrollState) {
  this.scrollState_ = scrollState;
}

ScrollHandle.prototype.hook = function(elem, propname) {
  var scrollState = this.scrollState_;
  process.nextTick(function() {
    if (scrollState.bottom) {
      elem.scrollTop = elem.scrollHeight - elem.clientHeight;
    }
  });
};

ScrollHandle.prototype.update = function(ev) {
  var elem = ev.currentTarget;
  this.scrollState_.bottom =
      elem.scrollTop === elem.scrollHeight - elem.clientHeight;
};

Playground.prototype.renderConsole_ = function(state) {
  if (state.hasRun) {
    var scrollHandle = new ScrollHandle(this.scrollState_);
    return h('div.console.open', {
      'ev-scroll': scrollHandle.update.bind(scrollHandle),
      'scrollhook': scrollHandle
    }, [
      h('div.text', _.map(state.consoleEvents, this.renderConsoleEvent_))
    ]);
  }
  return h('div.console');
};

Playground.prototype.render_ = function(state) {
  return h('div.pg', [
    this.renderLoadBar_(state),
    this.renderResetBar_(state),
    this.renderTabBar_(state),
    this.renderEditors_(state),
    this.renderConsole_(state)
  ]);
};

// Switches active tab to data.index.
Playground.prototype.switchTab = function(state, data) {
  this.setNotification_(state);
  state.activeTab.set(data.index);
};

// Reads the idToLoad text box into state.
Playground.prototype.setIdToLoad = function(state, formdata) {
  this.setNotification_(state);
  state.idToLoad.set(formdata.idToLoad);
};

Playground.prototype.showMessage_ = function(state, prefix, msg, ok) {
  var fullMsg = prefix + ': ' + msg;
  if (ok) {
    console.log(fullMsg);
  } else {
    console.error(fullMsg);
  }
  this.setNotification_(state, fullMsg, ok);
};

// Returns callback to be used for save and load requests. Callback loads the
// bundle returned from the server and updates bundleId and url.
Playground.prototype.saveLoadCallback_ = function(state, operation) {
  var that = this;
  return function(rerr, res) {
    state.requestActive.set(false);
    var processResponse = function() {
      if (rerr) {
        if (rerr.timeout) {
          return 'request timed out';
        }
        return 'error connecting to server: ' + rerr;
      }
      if (res.body && res.body.Error) {
        // TODO(ivanpi): Special handling of 404? Retry on other errors?
        return 'error ' + res.status + ': ' + res.body.Error;
      }
      if (res.error) {
        return 'error ' + res.status + ': unknown';
      }
      if (!res.body.Link || !res.body.Data) {
        return 'invalid response format';
      }
      var bundle;
      try {
        bundle = JSON.parse(res.body.Data);
      } catch (jerr) {
        return 'error parsing Data: ' + res.body.Data + '\n' + jerr.message;
      }
      // Opens bundle in editors, updates bundleId and url.
      that.setBundle_(state, bundle, res.body.Link);
      return null;
    };
    var errm = processResponse();
    if (!errm) {
      state.idToLoad.set('');
      that.showMessage_(state, operation, 'success', true);
    } else {
      // Load/save failed.
      if (state.bundleId() === false) {
        // If no bundle was loaded, load default.
        that.setBundle_(state, that.defaultBundle_, '');
      } else {
        // Otherwise, reset url to previously loaded bundle.
        that.setUrlForId_(state.bundleId());
      }
      that.showMessage_(state, operation, errm);
    }
  };
};

// Loads bundle for data.id.
Playground.prototype.load = function(state, data) {
  this.setNotification_(state);
  if (!data.id) {
    this.showMessage_(state, 'load', 'cannot load blank id');
    return;
  }
  superagent
    .get(this.getBackendUrl_() + '/load?id=' + encodeURIComponent(data.id))
    .accept('json')
    .timeout(storageRequestTimeout)
    .end(this.saveLoadCallback_(state, 'load ' + data.id));
  state.requestActive.set(true);
};

// Saves bundle and updates bundleId with the received id.
Playground.prototype.save = function(state) {
  this.setNotification_(state);
  superagent
    .post(this.getBackendUrl_() + '/save')
    .type('json')
    .accept('json')
    .timeout(storageRequestTimeout)
    .send(this.getBundle_())
    .end(this.saveLoadCallback_(state, 'save'));
  state.requestActive.set(true);
};

// Sends the files to the compile backend, streaming the response into the
// console.
Playground.prototype.run = function(state) {
  if (state.running()) {
    console.log('Already running', this.name_);
    return;
  }
  var runId = state.nextRunId();

  this.setNotification_(state);
  state.running.set(true);
  state.hasRun.set(true);
  state.consoleEvents.set([{Message: 'Running...'}]);
  this.scrollState_.bottom = true;

  var compileUrl = this.getBackendUrl_() + '/compile';
  if (this.url_.query.debug === '1') {
    compileUrl += '?debug=1';
  }

  var reqData = this.getBundle_();

  // TODO(sadovsky): To deal with cached responses, shift timestamps (based on
  // current time) and introduce a fake delay. Also, switch to streaming
  // messages, for usability.
  var that = this;

  // If the user stops the current run or resets the playground, functions
  // wrapped with ifRunActive become no-ops.
  var ifRunActive = function(cb) {
    return function() {
      if (runId === state.nextRunId()) {
        cb.apply(this, arguments);
      }
    };
  };

  var appendToConsole = function(events) {
    state.consoleEvents.set(state.consoleEvents().concat(events));
  };
  var makeEvent = function(stream, message) {
    return {Stream: stream, Message: message};
  };

  var urlp = url.parse(compileUrl);

  var options = {
    method: 'POST',
    protocol: urlp.protocol,
    hostname: urlp.hostname,
    port: urlp.port || (urlp.protocol === 'https:' ? '443' : '80'),
    path: urlp.path,
    // TODO(ivanpi): Change once deployed.
    withCredentials: false,
    headers: {
      'accept': 'application/json',
      'content-type': 'application/json'
    }
  };

  var req = http.request(options);

  var watchdog = null;
  // The heartbeat function clears the existing timeout (if any) and, if the run
  // is still active, starts a new timeout.
  var heartbeat = function() {
    if (watchdog !== null) {
      clearTimeout(watchdog);
    }
    watchdog = null;
    ifRunActive(function() {
      // TODO(ivanpi): Reduce timeout duration when server heartbeat is added.
      watchdog = setTimeout(function() {
        process.nextTick(ifRunActive(function() {
          req.destroy();
          appendToConsole(makeEvent('syserr', 'Server response timed out.'));
        }));
      }, 10500);
    })();
  };

  var endRunIfActive = ifRunActive(function() {
    that.stop(state);
    // Cleanup watchdog timer.
    heartbeat();
  });

  // error and close callbacks call endRunIfActive in the next tick to ensure
  // that if both events are triggered, both are executed before the run is
  // ended by either.
  req.on('error', ifRunActive(function(err) {
    console.error('Connection error: ' + err.message + '\n' + err.stack);
    appendToConsole(makeEvent('syserr', 'Error connecting to server.'));
    process.nextTick(endRunIfActive);
  }));

  // Holds partial prefix of next response line.
  var partialLine = '';

  req.on('response', ifRunActive(function(res) {
    heartbeat();
    if (res.statusCode !== 0 && res.statusCode !== 200) {
      appendToConsole(makeEvent('syserr', 'HTTP status ' + res.statusCode));
    }
    res.on('data', ifRunActive(function(chunk) {
      heartbeat();
      // Each complete line is one JSON Event.
      var eventsJson = (partialLine + chunk).split('\n');
      partialLine = eventsJson.pop();
      var events = [];
      _.forEach(eventsJson, function(line) {
        // Ignore empty lines.
        line = line.trim();
        if (line) {
          var ev;
          try {
            ev = JSON.parse(line);
          } catch (err) {
            console.error('Error parsing line: ' + line + '\n' + err.message);
            events.push(makeEvent('syserr', 'Error parsing server response.'));
            endRunIfActive();
            return false;
          }
          events.push(ev);
        }
      });
      appendToConsole(events);
    }));
  }));

  req.on('close', ifRunActive(function() {
    // Sanity check: partialLine should be empty when connection is closed.
    partialLine = partialLine.trim();
    if (partialLine) {
      console.error('Connection closed without newline after: ' + partialLine);
      appendToConsole(makeEvent('syserr', 'Error parsing server response.'));
    }
    process.nextTick(endRunIfActive);
  }));

  req.write(JSON.stringify(reqData));
  req.end();

  // Start watchdog.
  heartbeat();
};

// Clears the console and resets all editors to their original contents.
Playground.prototype.reset = function(state) {
  this.resetState_(state);
  _.forEach(this.editors_, function(editor) {
    editor.reset();
  });
  this.setUrlForId_(state.bundleId());
};

Playground.prototype.resetState_ = function(state) {
  state.consoleEvents.set([]);
  this.scrollState_.bottom = true;
  this.stop(state);
  state.hasRun.set(false);
};

// Stops bundle execution.
Playground.prototype.stop = function(state) {
  this.setNotification_(state);
  state.nextRunId.set((state.nextRunId() + 1) & 0x7fffffff);
  state.running.set(false);
};
