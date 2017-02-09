'use strict';


async function applyPatch (request, cachedResponse) {
  const headers = new Headers(request.headers);
  if (cachedResponse.headers.has('etag')) {
    headers.set('if-none-match', cachedResponse.headers.get('etag'));
  }

  headers.set('a-im', 'bsdiff');
  const serverResponse = await fetch(request.url, { headers });

  if (serverResponse.status !== 226) {
    return serverResponse.status === 200 ? serverResponse : cachedResponse;
  }

  const [{ value: old }, { value: patch }] = await Promise.all([
    cachedResponse.body.getReader().read(),
    serverResponse.body.getReader().read()
  ]);
  const newFile = BSDiff.MultiPatch(old, patch);
  return new Response(newFile[0], {
    headers: new Headers(serverResponse.headers),
  });
}

async function findAndPatch (request) {
  if (!/foobar/.test(request.url)) { return fetch(request); }
  const cachedResponse = await caches.match(request);
  const response = await applyPatch(request, cachedResponse);
  const cache = await caches.open('v1');
  cache.put(request, response.clone());
  return response;
}

self.addEventListener('fetch', event => {
  event.respondWith(findAndPatch(event.request));
});
