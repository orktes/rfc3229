'use strict';

const bspatch = require('bspatch');

function typedArrayStreamReader (typedArray) {
  // FIXME: the built-in reader in Chrome doesn't read, apparently, so this just satisfies the interface as a workaround
  return {
    async read () {
      if (typedArray) {
        const value = typedArray;
        typedArray = null;
        return { done: false, value };
      } else {
        return { done: true, value: undefined };
      }
    },
  };
}

async function readAll (reader) {
  const buffers = [];

  while (true) {
    const { value: buffer, done } = await reader.read();
    if (done) { break; }
    buffers.push(buffer.slice());
  }

  const size = buffers.reduce((size, buffer) => size + buffer.byteLength, 0);
  const result = new Uint8Array(size);

  let cursor = 0;
  buffers.forEach(buffer => {
    result.subarray(cursor, cursor + buffer.length).set(buffer);
    cursor += buffer.length;
  });

  return result;
}

function getUint64LE (data, offset) {
  // FIXME: the upper half of the word won't work
  return data
    .subarray(offset, offset + 8)
    .reduce((result, byte, index) => (byte << (index * 8)) | result, 0);
}

function getPatchOffsets (patchContainer) {
  const patchCount = getUint64LE(patchContainer, 0);
  return [...new Array(patchCount)].map((_, index) => {
    return getUint64LE(patchContainer, 8 + 8 * index);
  });
}

function splitPatches (patchContainer) {
  const offsets = getPatchOffsets(patchContainer);
  const headerSize = offsets.length * 8 + 8;
  return offsets.map((offset, index, list) => {
    return index === list.length - 1 ? patchContainer.subarray(offset) : patchContainer.subarray(offset, list[index + 1]);
  });
}

function multiPatch (old, patchContainer) {
  const patches = splitPatches(patchContainer);
  return patches.reduce((old, patch) => {
    const patchBufferReader = typedArrayStreamReader(patch);
    return bspatch(old, patchBufferReader).getReader();
  }, old);
}

async function applyPatches (serverResponse, cachedResponse) {
  const patchContainer = new Uint8Array(await serverResponse.arrayBuffer());
  const oldReader = cachedResponse.body.getReader();

  const result = await readAll(multiPatch(oldReader, patchContainer));

  return new Response(result, {
    headers: new Headers(serverResponse.headers),
  });
}

async function applyPatch (request, cachedResponse) {
  const headers = new Headers(request.headers);
  if (cachedResponse && cachedResponse.headers.has('etag')) {
    headers.set('if-none-match', cachedResponse.headers.get('etag'));
    headers.set('a-im', 'bsdiff');
  }

  const serverResponse = await fetch(request.url, { headers });

  if (serverResponse.status !== 226) {
    return serverResponse.status === 200 ? serverResponse : cachedResponse;
  }

  try {
    return await applyPatches(serverResponse, cachedResponse);
  } catch (error) {
    console.error('failed to apply patch', error);
    return fetch(request);
  }
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
