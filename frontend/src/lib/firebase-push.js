// Patra web-push client (FCM).
// The Firebase web config comes from the server at runtime
// (GET /api/public/firebase-config), so the JS client, the service worker,
// and the Go FCM sender always use the same Firebase project —
// a config mismatch is impossible by construction.
let messaging = null;
let firebaseConfig = null;

export async function getFirebaseConfig() {
  if (firebaseConfig) {
    return firebaseConfig;
  }
  const resp = await fetch('/api/public/firebase-config', {
    headers: { Accept: 'application/json' },
  });
  if (!resp.ok) {
    throw new Error(`firebase config request failed: ${resp.status}`);
  }
  const body = await resp.json();
  firebaseConfig = body.data || body;
  return firebaseConfig;
}

export function isPushSupported() {
  return typeof window !== 'undefined'
    && 'Notification' in window
    && 'serviceWorker' in navigator
    && 'PushManager' in window;
}

export function isSecureContextOk() {
  // Web Push needs a secure context. Plain http://localhost IS a secure
  // context (OK for testing); production must be HTTPS or getToken fails.
  return window.isSecureContext === true;
}

export function getPermissionState() {
  if (!isPushSupported()) {
    return 'unsupported';
  }
  return Notification.permission;
}

async function getMessaging() {
  if (messaging) {
    return messaging;
  }
  const cfg = await getFirebaseConfig();
  if (!cfg || !cfg.api_key || !cfg.project_id || !cfg.messaging_sender_id || !cfg.app_id) {
    throw new Error('push not configured: set FIREBASE_WEB_* env on the server (see .env.sample)');
  }
  const { initializeApp } = await import('firebase/app');
  const { getMessaging: getMsg } = await import('firebase/messaging');
  const app = initializeApp({
    apiKey: cfg.api_key,
    authDomain: cfg.auth_domain || undefined,
    projectId: cfg.project_id,
    messagingSenderId: cfg.messaging_sender_id,
    appId: cfg.app_id,
  });
  messaging = getMsg(app);
  return messaging;
}

// enablePush asks permission, registers /firebase-messaging-sw.js from the
// site root, and returns the FCM registration token (null when unavailable).
export async function enablePush() {
  if (!isPushSupported()) {
    throw new Error('web push is not supported in this browser');
  }
  if (!isSecureContextOk()) {
    throw new Error('web push needs a secure context: use https or http://localhost');
  }

  const permission = await Notification.requestPermission();
  if (permission !== 'granted') {
    throw new Error(`notification permission ${permission}`);
  }

  const cfg = await getFirebaseConfig();
  if (!cfg || !cfg.vapid_key) {
    throw new Error('push not configured: set FIREBASE_VAPID_KEY on the server (Firebase Console → Cloud Messaging → Web Push certificates)');
  }

  const msg = await getMessaging();
  const registration = await navigator.serviceWorker.register('/firebase-messaging-sw.js');
  const { getToken } = await import('firebase/messaging');
  const token = await getToken(msg, {
    vapidKey: cfg.vapid_key,
    serviceWorkerRegistration: registration,
  });
  if (!token) {
    throw new Error('FCM returned an empty token (VAPID key may belong to another project)');
  }
  return token;
}

// onForegroundPush invokes cb(payload) for messages received while the tab
// is open. Returns an unsubscribe function.
export async function onForegroundPush(cb) {
  const msg = await getMessaging();
  const { onMessage } = await import('firebase/messaging');
  return onMessage(msg, (payload) => cb(payload));
}
