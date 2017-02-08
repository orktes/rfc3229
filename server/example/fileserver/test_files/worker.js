/******/ (function(modules) { // webpackBootstrap
/******/ 	// The module cache
/******/ 	var installedModules = {};

/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {

/******/ 		// Check if module is in cache
/******/ 		if(installedModules[moduleId])
/******/ 			return installedModules[moduleId].exports;

/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = installedModules[moduleId] = {
/******/ 			i: moduleId,
/******/ 			l: false,
/******/ 			exports: {}
/******/ 		};

/******/ 		// Execute the module function
/******/ 		modules[moduleId].call(module.exports, module, module.exports, __webpack_require__);

/******/ 		// Flag the module as loaded
/******/ 		module.l = true;

/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}


/******/ 	// expose the modules object (__webpack_modules__)
/******/ 	__webpack_require__.m = modules;

/******/ 	// expose the module cache
/******/ 	__webpack_require__.c = installedModules;

/******/ 	// identity function for calling harmony imports with the correct context
/******/ 	__webpack_require__.i = function(value) { return value; };

/******/ 	// define getter function for harmony exports
/******/ 	__webpack_require__.d = function(exports, name, getter) {
/******/ 		if(!__webpack_require__.o(exports, name)) {
/******/ 			Object.defineProperty(exports, name, {
/******/ 				configurable: false,
/******/ 				enumerable: true,
/******/ 				get: getter
/******/ 			});
/******/ 		}
/******/ 	};

/******/ 	// getDefaultExport function for compatibility with non-harmony modules
/******/ 	__webpack_require__.n = function(module) {
/******/ 		var getter = module && module.__esModule ?
/******/ 			function getDefault() { return module['default']; } :
/******/ 			function getModuleExports() { return module; };
/******/ 		__webpack_require__.d(getter, 'a', getter);
/******/ 		return getter;
/******/ 	};

/******/ 	// Object.prototype.hasOwnProperty.call
/******/ 	__webpack_require__.o = function(object, property) { return Object.prototype.hasOwnProperty.call(object, property); };

/******/ 	// __webpack_public_path__
/******/ 	__webpack_require__.p = "";

/******/ 	// Load entry module and return exports
/******/ 	return __webpack_require__(__webpack_require__.s = 2);
/******/ })
/************************************************************************/
/******/ ([
/* 0 */
/***/ (function(module, exports, __webpack_require__) {

"use strict";


const { decompress, header } = __webpack_require__(1);

const MAGIC = "BSDIFF40";
const BUFFER_SIZE = 256;

function asyncIteratorToStream (iterator) {
  let prevSlice = null;
  let cursor = 0;
  let promiseResult = null;
  return new ReadableStream({
    start () {},
    close () {},
    async pull (controller) {
      if (!prevSlice || cursor >= prevSlice.length) {
        prevSlice = null;

        while (true) {
          const { done, value } = iterator.next(promiseResult);

          if (done) {
            controller.close();
            controller.byobRequest && controller.byobRequest.respond(0);
            return;
          }

          if (!value.then) {
            prevSlice = value;
            cursor = 0;
            break;
          }

          promiseResult = await value;
        }
      }

      if (controller.byobRequest) {
        const view = controller.byobRequest.view;
        let written = 0;
        const writeAmount = Math.min(prevSlice.length - cursor, view.length);
        view.set(prevSlice.subarray(cursor, cursor + writeAmount));
        cursor += writeAmount;
        controller.byobRequest.respond(writeAmount);
      } else {
        const result = prevSlice.subarray(cursor);
        prevSlice = null;
        cursor = 0;
        controller.enqueue(result);
      }
    },
  });
}

function bitReaderFromStreamReader (streamReader) {
  const BITMASK = new Uint8Array([0, 0x01, 0x03, 0x07, 0x0F, 0x1F, 0x3F, 0x7F, 0xFF]);
  const buffer = new Uint8Array(1);
  let cursor = -1;
  return async function readBits (n) {
  	let result = 0;
  	while (n > 0){
  		if (cursor === -1) {
        const { value : byteContainer, done } = await streamReader.read(buffer);
        if (done || byteContainer.byteLength === 0) { return -1; }
        cursor = 0;
  		}

  		const left = 8 - cursor;
  		const currentByte = buffer[0];

  		if (n >= left) {
  			result <<= left;
  			result |= (BITMASK[left] & currentByte);
  			cursor = -1;
  			n -= left;
  		} else {
  			result <<= n;
  			result |= ((currentByte & (BITMASK[n] << (8 - n - cursor))) >> (8 - n - cursor));
  			cursor += n;
  			n = 0;
  		}
  	}
  	return result;
  };
}

function * bzip2 (streamReader) {
  const bitReader = bitReaderFromStreamReader(streamReader);

  const size = yield header(bitReader);

  while (true) {
    const chunk = yield decompress(bitReader, size);
    if (chunk === -1) { break; }
    const block = new Uint8Array(chunk.length);
    for (let i = 0; i < chunk.length; i++) {
      block[i] = chunk.charCodeAt(i);
    }
    yield block;
  }

  // seek over the CRC
  yield bitReader(8 * 4);
}

function bzip2Stream (streamReader) {
  return asyncIteratorToStream(bzip2(streamReader));
}

function getInt64 (bytes, offset = 0) {
  const result =
    (bytes[offset + 0] & 0xff) << (0 * 8) |
    (bytes[offset + 1] & 0xff) << (1 * 8) |
    (bytes[offset + 2] & 0xff) << (2 * 8) |
    (bytes[offset + 3] & 0xff) << (3 * 8) |
    (bytes[offset + 4] & 0xff) << (4 * 8) |
    (bytes[offset + 5] & 0xff) << (5 * 8) |
    (bytes[offset + 6] & 0xff) << (6 * 8) |
    (bytes[offset + 7] & 0x7f) << (7 * 8);
  const signed = bytes[offset + 7] & 0x80 != 0;
  return signed ? -result : result;
}

function add (left, right, offset, end) {
  for (let i = offset; i < end; i++) {
    left[i] += right[i];
  }

  return left;
}

function getAsciiString (bytes, offset, end) {
  let string = '';

  for (let i = offset; i < end; i++) {
    string += String.fromCharCode(bytes[i]);
  }

  return string;
}

function emulateBYOB (streamReader) {
  if (streamReader.read.length === 1) { return streamReader; }
  let buffer = null;
  let offset = 0;

  async function read (destination) {
    if (!buffer || buffer.length === offset) {
      let done;
      ({ value : buffer, done } = await streamReader.read());
      if (done) { return { done, value: destination.subarray(0, 0) }; }
      offset = 0;
    }

    const copyAmount = Math.min(destination.length, buffer.length - offset);
    destination.set(buffer.subarray(offset, offset + copyAmount));
    offset += copyAmount;
    const value = destination.subarray(0, copyAmount);
    return { done: false, value };
  }

  return { read };
}

async function read (streamReader, buffer, offset, end) {
  if (offset >= end) { return; }
  const destination = buffer.subarray(offset, end);
  const { value: newView, done } = await streamReader.read(destination);
  return read(streamReader, buffer, offset + newView.byteLength, end);
}

async function readControlHeader (streamReader) {
  const data = new Uint8Array(24);
  await read(streamReader, data, 0, data.length);
  const addSize = getInt64(data, 0 * 8);
  const copySize = getInt64(data, 1 * 8);
  const seekSize = getInt64(data, 2 * 8);
  return { addSize, copySize, seekSize };
}

async function readHeader (streamReader) {
  const data = new Uint8Array(32);
  await read(streamReader, data, 0, data.length);
  const magic = getAsciiString(data, 0, 8);
  const controlSize = getInt64(data, 1 * 8);
  const diffSize = getInt64(data, 2 * 8);
  const newSize = getInt64(data, 3 * 8);
  return { magic, controlSize, diffSize, newSize };
}

function typedArrayStreamReader (typedArray) {
  return emulateBYOB(asyncIteratorToStream([typedArray][Symbol.iterator]()).getReader());
}

function * bspatch (oldReader, patchReader) {
  oldReader = emulateBYOB(oldReader);
  patchReader = emulateBYOB(patchReader);

  const { magic, controlSize, diffSize, newSize } = yield readHeader(patchReader);

  if (magic != MAGIC || controlSize < 0 || diffSize < 0 || newSize < 0) {
    throw new Error('Corrupt patch');
  }

  const control = new Uint8Array(controlSize);
  const controlReader = emulateBYOB(bzip2Stream(typedArrayStreamReader(control)).getReader());
  yield read(patchReader, control, 0, controlSize);

  const diff = new Uint8Array(diffSize);
  const diffReader = emulateBYOB(bzip2Stream(typedArrayStreamReader(diff)).getReader());
  yield read(patchReader, diff, 0, diffSize);

  const extraReader = emulateBYOB(bzip2Stream(patchReader).getReader());

  const temp1 = new Uint8Array(BUFFER_SIZE);
  const temp2 = new Uint8Array(BUFFER_SIZE);
  let newPos = 0;
  let oldPos = 0;

  while (newPos < newSize) {
    const { addSize, copySize, seekSize } = yield readControlHeader(controlReader);

    if (newPos + addSize > newSize || addSize < 0) {
      throw new Error('Corrupt patch');
    }

    for (let i = 0; i < addSize; i += BUFFER_SIZE) {
      const readAmount = Math.min(BUFFER_SIZE, addSize - i);
      yield Promise.all([read(oldReader, temp1, 0, readAmount), read(diffReader, temp2, 0, readAmount)]);
      add(temp1, temp2, 0, readAmount);
      newPos += readAmount;
      oldPos += readAmount;
      yield temp1.slice(0, readAmount);
    }

    if (newPos + copySize > newSize || copySize < 0) {
      throw new Error('Corrupt patch');
    }

    for (let i = 0; i < copySize; i += BUFFER_SIZE) {
      const readAmount = Math.min(BUFFER_SIZE, copySize - i);
      yield read(extraReader, temp1, 0, readAmount);
      newPos += readAmount;
      yield temp1.slice(0, readAmount);
    }

    if (seekSize < 0) {
      throw new Error('Corrupt patch');
    }

    for (let i = 0; i < seekSize; i += BUFFER_SIZE) {
      const readAmount = Math.min(BUFFER_SIZE, seekSize - i);
      yield read(oldReader, temp1, 0, readAmount);
      oldPos += readAmount;
    }
  }
}

function bspatchStream (oldReader, patchReader) {
  return asyncIteratorToStream(bspatch(oldReader, patchReader));
}

module.exports = bspatchStream;


/***/ }),
/* 1 */
/***/ (function(module, exports, __webpack_require__) {

"use strict";


var bzip2 = {};

bzip2.array = function(bytes){
  var bit = 0, byte = 0;
  var BITMASK = [0, 0x01, 0x03, 0x07, 0x0F, 0x1F, 0x3F, 0x7F, 0xFF ];
  return function(n){
    var result = 0;
    while(n > 0){
      var left = 8 - bit;
      if(n >= left){
        result <<= left;
        result |= (BITMASK[left] & bytes[byte++]);
        bit = 0;
        n -= left;
      }else{
        result <<= n;
        result |= ((bytes[byte] & (BITMASK[n] << (8 - n - bit))) >> (8 - n - bit));
        bit += n;
        n = 0;
      }
    }
    return result
  }
}

bzip2.simple = async function(bits){
  var size = await bzip2.header(bits);
  var all = '', chunk = '';
  do{
    all += chunk;
    chunk = await bzip2.decompress(bits, size);
  }while(chunk != -1);
  return all;
}

bzip2.header = async function(bits){
  if((await bits(8*3)) != 4348520) throw "No magic number found";
  var i = (await bits(8)) - 48;
  if(i < 1 || i > 9) throw "Not a BZIP archive";
  return i;
};


//takes a function for reading the block data (starting with 0x314159265359)
//a block size (0-9) (optional, defaults to 9)
//a length at which to stop decompressing and return the output
bzip2.decompress = async function(bits, size, len){
  var MAX_HUFCODE_BITS = 20;
  var MAX_SYMBOLS = 258;
  var SYMBOL_RUNA = 0;
  var SYMBOL_RUNB = 1;
  var GROUP_SIZE = 50;

  var bufsize = 100000 * size;
  for(var h = '', i = 0; i < 6; i++) h += (await bits(8)).toString(16);
  if(h == "177245385090") return -1; //last block
  if(h != "314159265359") throw "eek not valid bzip data";
  await bits(32); //ignore CRC codes
  if(await bits(1)) throw "unsupported obsolete version";
  var origPtr = await bits(24);
  if(origPtr > bufsize) throw "Initial position larger than buffer size";
  var t = await bits(16);
  var symToByte = new Uint8Array(256),
      symTotal = 0;
  for (i = 0; i < 16; i++) {
    if(t & (1 << (15 - i))) {
      var k = await bits(16);
      for(j = 0; j < 16; j++){
        if(k & (1 << (15 - j))){
          symToByte[symTotal++] = (16 * i) + j;
        }
      }
    }
  }

  var groupCount = await bits(3);
  if(groupCount < 2 || groupCount > 6) throw "another error";
  var nSelectors = await bits(15);
  if(nSelectors == 0) throw "meh";
  var mtfSymbol = []; //TODO: possibly replace JS array with typed arrays
  for(var i = 0; i < groupCount; i++) mtfSymbol[i] = i;
  var selectors = new Uint8Array(32768);

  for(var i = 0; i < nSelectors; i++){
    for(var j = 0; await bits(1); j++) if(j >= groupCount) throw "whoops another error";
    var uc = mtfSymbol[j];
    mtfSymbol.splice(j, 1); //this is a probably inefficient MTF transform
    mtfSymbol.splice(0, 0, uc);
    selectors[i] = uc;
  }

  var symCount = symTotal + 2;
  var groups = [];
  for(var j = 0; j < groupCount; j++){
    var length = new Uint8Array(MAX_SYMBOLS),
        temp = new Uint8Array(MAX_HUFCODE_BITS+1);
    t = await bits(5); //lengths
    for(var i = 0; i < symCount; i++){
      while(true){
        if (t < 1 || t > MAX_HUFCODE_BITS) throw "I gave up a while ago on writing error messages";
        if(!(await bits(1))) break;
        if(!(await bits(1))) t++;
        else t--;
      }
      length[i] = t;
    }
    var  minLen,  maxLen;
    minLen = maxLen = length[0];
    for(var i = 1; i < symCount; i++){
      if(length[i] > maxLen) maxLen = length[i];
      else if(length[i] < minLen) minLen = length[i];
    }
    var hufGroup;
    hufGroup = groups[j] = {};
    hufGroup.permute = new Uint32Array(MAX_SYMBOLS);
    hufGroup.limit = new Uint32Array(MAX_HUFCODE_BITS + 1);
    hufGroup.base = new Uint32Array(MAX_HUFCODE_BITS + 1);
    hufGroup.minLen = minLen;
    hufGroup.maxLen = maxLen;
    var base = hufGroup.base.subarray(1);
    var limit = hufGroup.limit.subarray(1);
    var pp = 0;
    for(var i = minLen; i <= maxLen; i++)
      for(var t = 0; t < symCount; t++)
      if(length[t] == i) hufGroup.permute[pp++] = t;
      for(i = minLen; i <= maxLen; i++) temp[i] = limit[i] = 0;
      for(i = 0; i < symCount; i++) temp[length[i]]++;
      pp = t = 0;
      for(i = minLen; i < maxLen; i++) {
        pp += temp[i];
        limit[i] = pp - 1;
        pp <<= 1;
        base[i+1] = pp - (t += temp[i]);
      }
      limit[maxLen]=pp+temp[maxLen]-1;
      base[minLen]=0;
  }
  var byteCount = new Uint32Array(256);
  for(var i = 0; i < 256; i++) mtfSymbol[i] = i;
  var runPos, count, symCount, selector;
  runPos = count = symCount = selector = 0;
  var buf = new Uint32Array(bufsize);
  while(true){
    if(!(symCount--)){
      symCount = GROUP_SIZE - 1;
      if(selector >= nSelectors) throw "meow i'm a kitty, that's an error";
      hufGroup = groups[selectors[selector++]];
      base = hufGroup.base.subarray(1);
      limit = hufGroup.limit.subarray(1);
    }
    i = hufGroup.minLen;
    j = await bits(i);
    while(true){
      if(i > hufGroup.maxLen) throw "rawr i'm a dinosaur";
      if(j <= limit[i]) break;
      i++;
      j = (j << 1) | await bits(1);
    }
    j -= base[i];
    if(j < 0 || j >= MAX_SYMBOLS) throw "moo i'm a cow";
    var nextSym = hufGroup.permute[j];
    if (nextSym == SYMBOL_RUNA || nextSym == SYMBOL_RUNB) {
      if(!runPos){
        runPos = 1;
        t = 0;
      }
      if(nextSym == SYMBOL_RUNA) t += runPos;
      else t += 2 * runPos;
      runPos <<= 1;
      continue;
    }
    if(runPos){
      runPos = 0;
      if(count + t >= bufsize) throw "Boom.";
      uc = symToByte[mtfSymbol[0]];
      byteCount[uc] += t;
      while(t--) buf[count++] = uc;
    }
    if(nextSym > symTotal) break;
    if(count >= bufsize) throw "I can't think of anything. Error";
    i = nextSym -1;
    uc = mtfSymbol[i];
    mtfSymbol.splice(i, 1);
    mtfSymbol.splice(0, 0, uc);
    uc = symToByte[uc];
    byteCount[uc]++;
    buf[count++] = uc;
  }
  if(origPtr < 0 || origPtr >= count) throw "I'm a monkey and I'm throwing something at someone, namely you";
  var j = 0;
  for(var i = 0; i < 256; i++){
    k = j + byteCount[i];
    byteCount[i] = j;
    j = k;
  }
  for(var i = 0; i < count; i++){
    uc = buf[i] & 0xff;
    buf[byteCount[uc]] |= (i << 8);
    byteCount[uc]++;
  }
  var pos = 0, current = 0, run = 0;
  if(count) {
    pos = buf[origPtr];
    current = (pos & 0xff);
    pos >>= 8;
    run = -1;
  }
  count = count;
  var output = '';
  var copies, previous, outbyte;
  if(!len) len = Infinity;
  while(count){
    count--;
    previous = current;
    pos = buf[pos];
    current = pos & 0xff;
    pos >>= 8;
    if(run++ == 3){
      copies = current;
      outbyte = previous;
      current = -1;
    }else{
      copies = 1;
      outbyte = current;
    }
    while(copies--){
      output += (String.fromCharCode(outbyte));
      if(!--len) return output;
    }
    if(current != previous) run = 0;
  }
  return output;
}

module.exports = bzip2;


/***/ }),
/* 2 */
/***/ (function(module, exports, __webpack_require__) {

"use strict";


const bspatch = __webpack_require__(0);

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


/***/ })
/******/ ]);
