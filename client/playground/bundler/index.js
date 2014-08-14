var _ = require('lodash');
var fs = require('fs');
var glob = require('glob');
var path = require('path');

// Filename to write the data to.
var BUNDLE_NAME = 'bundle.json';

module.exports = { run: run };

// TODO(nlacasse): improve this.
function usage(){
  console.error('Usage: pgbundle <path> [<path> <path> ...]');
  process.exit(1);
}

// Main function.
function run(){
  // Get the paths from process.argv
  var argv = require('minimist')(process.argv.slice(2));
  var dirs = argv._;

  // Make sure there is at least one path.
  if (!dirs || dirs.length === 0) {
    return usage();
  }

  // Loop over each path.
  _.each(dirs, function(dir){
    var subFiles = glob.sync('*', {
      cwd: dir,
      mark: true // Add a '/ character to directory matches
    });

    if (subFiles.length === 0) {
      return usage();
    }

    var out = { files: [] };

    // Loop over each subfile in the path.
    _.each(subFiles, function(fileName) {
      // Ignore directories.
      if (_.last(fileName) === '/') {
        return;
      }

      // Ignore a previously bundled file.
      if (fileName === BUNDLE_NAME) {
        return;
      }

      var fullFilePath = path.resolve(dir, fileName);
      out.files.push({
        name: fileName,
        body: fs.readFileSync(fullFilePath, { encoding: 'utf8' })
      });
    });

    // Write the bundle.json.
    var outFile = path.resolve(dir, BUNDLE_NAME);
    fs.writeFileSync(outFile, JSON.stringify(out));
    console.log('Wrote ' + outFile);
  });
}
