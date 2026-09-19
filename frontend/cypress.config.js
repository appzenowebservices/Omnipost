const { defineConfig } = require('cypress');

module.exports = defineConfig({
  env: {
    apiUrl: 'http://localhost:9000',
    serverInitCmd:
      'pkill -9 patra; cd ../ && PATRA_ADMIN_USER=admin PATRA_ADMIN_PASSWORD=patra ./patra --install --yes && setsid ./patra </dev/null >/dev/null 2>&1 &',
    serverInitBlankCmd:
      'pkill -9 patra; cd ../ && ./patra --install --yes && setsid ./patra </dev/null >/dev/null 2>&1 &',
    PATRA_ADMIN_USER: 'admin',
    PATRA_ADMIN_PASSWORD: 'patra',
  },
  viewportWidth: 1400,
  viewportHeight: 950,
  e2e: {
    experimentalRunAllSpecs: true,
    testIsolation: false,
    experimentalSessionAndOrigin: false,
    // We've imported your old cypress plugins here.
    // You may want to clean this up later by importing these.
    setupNodeEvents(on, config) {
      return require('./cypress/plugins/index.js')(on, config);
    },
    baseUrl: 'http://localhost:9000',
  },
});
