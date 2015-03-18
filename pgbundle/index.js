var _ = require('lodash');
var fs = require('fs');
var glob = require('glob');
var path = require('path');

module.exports = {run: run};

function usage() {
  console.error('Usage: pgbundle [options] <glob_file> <root_path>');
  console.error('Arguments: <glob_file>: Path to file containing a list of' +
    ' glob patterns, one per line. The bundle includes only files with path' +
    ' suffixes matching one of the globs. Each glob must match at least one' +
    ' file, otherwise bundling fails with a non-zero exit code.');
  console.error('           <root_path>: Path to directory where files' +
    ' matching glob patterns are taken from.');
  console.error('Options: --verbose: Enable verbose output.');
  console.error('         --empty: Omit file contents in bundle, include only' +
    ' paths and metadata.');
  process.exit(1);
}

// If the first line is "// +build ignore", strip the line and return the
// remaining lines.
function stripBuildIgnore(lines) {
  if (lines.length > 0 && _.trim(lines[0]) === '// +build ignore') {
    return _.rest(lines);
  }
  return lines;
}

// Strip all blank lines at the beginning of the file.
function stripLeadingBlankLines(lines) {
  var nb = 0;
  for (; nb < lines.length && _.trim(lines[nb]) === ''; ++nb) /* no-op */;
  return _.slice(lines, nb);
}

// If the first line is an index comment, strip the line and return the index
// and remaining lines.
function getIndex(lines) {
  var index = null;
  var match = lines.length > 0 &&
              _.trim(lines[0]).match(/^\/\/\s*index=(\d+)/);
  if (match && match[1]) {
    index = match[1];
    lines = _.rest(lines);
  }
  return {
    index: index,
    lines: lines
  };
}

// Main function.
function run() {
  // Get the flags and positional arguments from process.argv.
  var argv = require('minimist')(process.argv.slice(2), {
    boolean: ['verbose', 'empty']
  });

  // Make sure the glob file and the root path path are specified.
  if (!argv._ || argv._.length !== 2) {
    return usage();
  }

  var globFile = argv._[0];
  var dir = argv._[1];
  // Read glob file, filtering out empty lines.
  var patterns = _.filter(
    fs.readFileSync(globFile, {encoding: 'utf8'}).split('\n'));
  // The root path must be a directory.
  if (!fs.lstatSync(dir).isDirectory()) {
    return usage();
  }

  var unmatched = [];

  // Apply each glob pattern to the directory.
  var relpaths = _.flatten(_.map(patterns, function(pattern) {
    var match = glob.sync('**/' + pattern, {
      cwd: dir,
      nodir: true
    });
    if (match.length === 0) {
      unmatched.push(pattern);
    }
    return match;
  }));

  // If any pattern matched zero files, halt with a non-zero exit code.
  // TODO(ivanpi): Allow optional patterns, e.g. prefixed by '?'?
  if (unmatched.length > 0) {
    console.warn('Error bundling "%s": unmatched patterns %j', dir, unmatched);
    process.exit(2);
  }

  var out = {files: []};

  // Loop over each file.
  _.each(relpaths, function(relpath) {
    var abspath = path.resolve(dir, relpath);
    var lines = fs.readFileSync(abspath, {encoding: 'utf8'}).split('\n');

    lines = stripBuildIgnore(lines);
    lines = stripLeadingBlankLines(lines);
    var indexAndLines = getIndex(lines);
    var index = indexAndLines.index;
    lines = indexAndLines.lines;

    out.files.push({
      name: relpath,
      body: argv.empty ? '' : lines.join('\n'),
      index: index
    });
  });

  out.files = _.sortBy(out.files, 'index');

  // Drop the index fields -- we don't need them anymore.
  out.files = _.map(out.files, function(f) {
    return _.omit(f, 'index');
  });

  // Write the bundle to stdout.
  process.stdout.write(JSON.stringify(out) + '\n');

  if (argv.verbose) {
    console.warn('Bundled "%s" using "%s"', dir, globFile);
  }
}
