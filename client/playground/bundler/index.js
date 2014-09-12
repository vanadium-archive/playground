var _ = require('lodash');
var fs = require('fs');
var glob = require('glob');
var path = require('path');

// Filename to write the data to.
var BUNDLE_NAME = 'bundle.json';

module.exports = { run: run };

// TODO(nlacasse): improve this.
function usage() {
  console.log('Usage: pgbundle [options] <path> [<path> <path> ...]');
  console.log('Options: --verbose   defaults to false');
  process.exit(1);
}

// Main function.
function run() {
  // Get the paths from process.argv.
  var argv = require('minimist')(process.argv.slice(2));
  var dirs = argv._;

  // Make sure there is at least one path.
  if (!dirs || dirs.length === 0) {
    return usage();
  }

  // Loop over each path.
  _.each(dirs, function(dir) {
    var subFiles = glob.sync('**', {
      cwd: dir,
      mark: true // Add a '/ character to directory matches
    });

    if (subFiles.length === 0) {
      return usage();
    }

    var out = { files: [] };

    // Loop over each subfile in the path.
    _.each(subFiles, function(fileName) {
      if (shouldIgnore(fileName)) {
        return;
      }

      var fullFilePath = path.resolve(dir, fileName);
      var text = fs.readFileSync(fullFilePath, { encoding: 'utf8' });

      var indexAndText = getIndex(text);
      var index = indexAndText.index;
      text = indexAndText.text;

      out.files.push({
        name: path.basename(fileName),
        text: text,
        index: index
      });
    });

    out.files = _.sortBy(out.files, 'index');

    // Write the bundle.json.
    var outFile = path.resolve(dir, BUNDLE_NAME);
    fs.writeFileSync(outFile, JSON.stringify(out));

    if (argv.verbose) {
      console.log('Wrote ' + outFile);
    }
  });
}

// If the first line is an index comment, strip the line from text and return
// the index and stripped text.
function getIndex(text) {
  var index = null;
  var lines = text.split('\n');
  var match = lines[0].match(/^\/\/\s*index=(\d+)/);
  if (match && match[1]) {
    index = match[1];
    // Remove the first line from text.
    text = _.rest(lines).join('\n');
  }
  return {
    index: index,
    text: text
  };
}

function shouldIgnore(fileName) {
  // Ignore directories.
  if (_.last(fileName) === '/') {
    return true;
  }

  // Ignore bundle files.
  if (fileName === BUNDLE_NAME) {
    return true;
  }

  // Ignore generated .vdl.go files.
  if ((/\.vdl\.go$/i).test(fileName)) {
    return true;
  }

  return false;
}
