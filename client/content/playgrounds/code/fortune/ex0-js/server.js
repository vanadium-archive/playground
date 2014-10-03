var veyron = require('veyron');

/**
 * 1) Implement a simple fortune service
 */

var fortuneService = {
  // List of fortunes
  fortunes: [],

  numFortunesServed: 0,

  // Gets a random fortune
  getRandomFortune: function() {
    var numExistingfortunes = this.fortunes.length;
    if(numExistingfortunes === 0) {
      throw new Error('Sorry! No fortune available :(');
    }
    var randomIndex = Math.floor(Math.random() * numExistingfortunes);
    var fortune = this.fortunes[randomIndex];
    this.numFortunesServed++;
    console.info('Serving:', fortune);
    return fortune;
  },

  // Adds a new fortune
  addNewFortune: function(fortune) {
    if(!fortune || fortune.trim() === '') {
      throw new Error('Sorry! Can\'t add empty or null fortune!');
    }
    console.info('Adding:', fortune);
    this.fortunes.push(fortune);
  }
};

/**
 * 2) Publish the fortune service
 */

// Create a Veyron runtime using the configuration
veyron.init({}).then(function(rt){
  // Serve the fortune server under a name. Serve returns a Promise object
  rt.serve('bakery/cookie/fortune', fortuneService).then(function() {
    console.log('Fortune server serving under: bakery/cookie/fortune \n');
  }).catch(function(err) {
    console.log('Failed to serve the Fortune server because: \n', err);
  });
}).catch(function(err) {
  console.error('Failed to start the fortune server because:', err);
});

// Let's add a few fortunes to start with
fortuneService.addNewFortune('The fortune you seek is in another cookie.');
fortuneService.addNewFortune('Everything will now come your way.');
fortuneService.addNewFortune('Conquer your fears or they will conquer you.');
