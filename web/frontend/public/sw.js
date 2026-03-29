const PROVISIONING_CACHE = 'moonhub-provisioning-v1';

// Static assets to cache (offline-cap capable)
const STATIC_ASSETS = [
  '/',
  '/provisioning/',
  '/provisioning/auth',
  '/provisioning/install',
];

// Patterns to skip (API calls must real-time)
const SKIP_PATTERNS = ['/api/', '/ws'];

// Install event - cache static assets
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(PROVISIONING_CACHE)
      .then((cache) => cache.addAll(STATIC_ASSETS))
      .then(() => self.skipWaiting())
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter(k => k !== PROVISIONING_CACHE).map(k => caches.delete(k)))
    )
  );
});

// Fetch event - serve from cache, fallback to network
self.addEventListener('fetch', (event) => {
  // Skip API calls - they must go to network
  if (SKIP_PATTERNS.some(p => event.request.url.includes(p))) {
    return;
  }

  event.respondWith(
    caches.match(event.request).then((cached) => {
      if (cached) {
        return cached;
      }
      return fetch(event.request);
    })
  );
});
