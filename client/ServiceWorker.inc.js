'use strict';


async function applyPatch (request, cachedResponse) {
  const headers = new Headers(request.headers);
  if (cachedResponse && cachedResponse.headers.has('etag')) {
    headers.set('if-none-match', cachedResponse.headers.get('etag'));
  }

  headers.set('a-im', 'bsdiff');
  const serverResponse = await fetch(request.url, { headers });

  if (serverResponse.status !== 226) {
    cachedResponse.headers.set('patch-length', 0);
    return serverResponse.status === 200 ? serverResponse : cachedResponse;
  }

  const [old, patch] = await Promise.all([
    cachedResponse.arrayBuffer().then((buf)=> new Uint8Array(buf)),
    serverResponse.arrayBuffer().then((buf)=> new Uint8Array(buf))
  ]);

  const newFile = BSDiff.MultiPatch(old, patch);
  const responseHeaders = new Headers(serverResponse.headers);
  responseHeaders.set('content-type', cachedResponse && cachedResponse.headers.get('content-type'));
  responseHeaders.set('content-length', newFile[0].length);
  responseHeaders.set('patch-length', patch.length);
  const patchedResponse = new Response(newFile[0], {
    headers: responseHeaders,
  });
  return patchedResponse;
}

async function findAndPatch (request) {
  if (/worker\.js/.test(request.url)) {
    return fetch(request);
  }

  const cachedResponse = await caches.match(request);

  let response;
  if (!cachedResponse) {
    response = await fetch(request);
  } else {
    response = await applyPatch(request, cachedResponse);
  }

  const cache = await caches.open('v1');
  cache.put(request, response.clone());

  return response;
}

self.addEventListener('fetch', event => {
  event.respondWith(
    findAndPatch(event.request).then((response)=> {
      self.clients.matchAll({type: 'window'}).then((clients)=> {
        for (var i = 0; i < clients.length; i++) {
          if (clients[i].id === event.clientId) {
            return clients[i];
          }
        }
      }).then(client => client && client.postMessage({
        size: response.headers.get('patch-length') || response.headers.get('content-length')
      }));
      return response;
    })
  )
});
