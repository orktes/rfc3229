'use strict';


async function applyPatch (request, cachedResponse) {
  const headers = new Headers(request.headers);
  if (cachedResponse && cachedResponse.headers.has('etag')) {
    headers.set('if-none-match', cachedResponse.headers.get('etag'));
  }

  headers.set('a-im', 'bsdiff');
  const serverResponse = await fetch(request.url, { headers });

  if (serverResponse.status !== 226) {
    return serverResponse.status === 200 ? serverResponse : cachedResponse;
  }

  console.log('Going to try to patch');

  const [{ value: old }, { value: patch }] = await Promise.all([
    cachedResponse.body.getReader().read(),
    serverResponse.body.getReader().read()
  ]);

  const newFile = BSDiff.MultiPatch(old, patch);
  const responseHeaders = new Headers(serverResponse.headers);
  responseHeaders.set('content-type', cachedResponse && cachedResponse.headers.get('content-type'));
  responseHeaders.set('content-length', newFile[0].length);
  return new Response(newFile[0], {
    headers: responseHeaders,
  });
}

async function findAndPatch (request) {
  if (/worker\.js/.test(request.url)) {
    return fetch(request);
  }

  const cachedResponse = await caches.match(request);
  const response = await applyPatch(request, cachedResponse);
  const cache = await caches.open('v1');
  cache.put(request, response.clone());
  return response;
}

self.addEventListener('fetch', event => {
  event.respondWith(findAndPatch(event.request));
});
