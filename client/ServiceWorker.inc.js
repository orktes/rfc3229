'use strict';


async function applyPatch (request, response) {
  if (!response || !response.headers.has('etag')) { return [true, await fetch(request)]; }
  const headers = new Headers(request.headers);
  headers.set('if-none-match', response.headers.get('etag'));
  headers.set('a-im', 'bsdiff');
  const patchResponse = await fetch(request.url, { headers });
  if (
    patchResponse.status !== 226 ||
    !patchResponse.headers.has('im') ||
    patchResponse.headers.get('im') !== 'bsdiff') {
      if (patchResponse.status === 304) {
        return [false, response];
      }
      return [false, await fetch(request)];
  }
  const [{ value: old }, { value: patch }] = await Promise.all([response.body.getReader().read(), patchResponse.body.getReader().read()]);
  const newFile = BSDiff.Patch(old, patch);
  return [true, new Response(newFile[0], {
    headers: new Headers(patchResponse.headers),
  })];
}

async function findAndPatch (request) {
  if (!/foobar/.test(request.url)) { return fetch(request); }
  const cachedResponse = await caches.match(request);
  const [shouldCacheResponse, response] = await applyPatch(request, cachedResponse);
  const cache = await caches.open('v1');
  shouldCacheResponse && cache.put(request, response.clone());
  return response;
}

self.addEventListener('fetch', event => {
  event.respondWith(findAndPatch(event.request));
});
