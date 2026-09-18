/* OmniPost web-push service worker (FCM).
 * Served from the SITE ROOT as /firebase-messaging-sw.js by the Go backend
 * (cmd/push.go), which injects the Firebase web config below from server
 * env at request time. Do not hardcode project keys here — the placeholders
 * keep the JS client, this worker, and the Go FCM sender on one project.
 * Web Push requires this file at the root scope; it cannot live under /admin/.
 */
importScripts('https://www.gstatic.com/firebasejs/10.12.0/firebase-app-compat.js');
importScripts('https://www.gstatic.com/firebasejs/10.12.0/firebase-messaging-compat.js');

(function init() {
  var config = {
    apiKey: '__FIREBASE_API_KEY__',
    authDomain: '__FIREBASE_AUTH_DOMAIN__',
    projectId: '__FIREBASE_PROJECT_ID__',
    messagingSenderId: '__FIREBASE_MESSAGING_SENDER_ID__',
    appId: '__FIREBASE_APP_ID__',
  };

  if (!config.apiKey || !config.projectId || !config.messagingSenderId
    || config.apiKey.indexOf('__FIREBASE') === 0) {
    // Push not configured on the server (see .env.sample FIREBASE_*).
    // Worker stays installed but idle so registration never breaks the app.
    return;
  }

  try {
    firebase.initializeApp(config);
    var messaging = firebase.messaging();

    // Background / closed-tab notifications.
    messaging.onBackgroundMessage(function onBackgroundMessage(payload) {
      var title = 'OmniPost';
      var body = '';
      if (payload && payload.notification) {
        if (payload.notification.title) {
          title = payload.notification.title;
        }
        if (payload.notification.body) {
          body = payload.notification.body;
        }
      }
      var data = (payload && payload.data) || {};
      self.registration.showNotification(title, {
        body: body,
        icon: '/public/static/favicon.png',
        data: data,
      });
    });
  } catch (e) {
    // Never let push setup break the worker registration.
  }
}());

// Tapping a notification opens the admin (or the payload's deep-link URL).
self.addEventListener('notificationclick', function onClick(event) {
  event.notification.close();
  var url = '/admin/';
  if (event.notification && event.notification.data && event.notification.data.url) {
    url = event.notification.data.url;
  }
  event.waitUntil(clients.openWindow(url));
});
