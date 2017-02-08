'use strict';

const bspatch = require('bspatch');

async function applyPatch (request, response) {
  if (!response || !response.headers.has('etag')) { return fetch(request); }
  const headers = new Headers(request.headers);
  headers.set('etag', response.headers.get('etag'));
  headers.set('a-im', 'bsdiff');
  const patchResponse = await fetch(request.url, { headers });
  if (!patchResponse.headers.has('im') || patchResponse.headers.get('im') !== 'bsdiff') { return patchResponse; }
  const newStream = bspatch(response.body.getReader(), patchResponse.body.getReader());
  return new Response(newStream, {
    headers: new Headers(patchResponse.headers),
  });
}

async function findAndPatch (request) {
  const cachedResponse = await caches.match(request);
  const response = await applyPatch(request, cachedResponse);
  const cache = await caches.open('v1');
  cache.put(request, response.clone());
  return response;
}

self.addEventListener('fetch', event => {
  event.respondWith(findAndPatch(event.request));
});
