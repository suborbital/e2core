
export function clamp_host(i, min, max) {
  if (!Number.isInteger(i)) throw new TypeError(`must be an integer`);
  if (i < min || i > max) throw new RangeError(`must be between ${min} and ${max}`);
  return i;
}

const UTF8_ENCODER = new TextEncoder('utf-8');

export function utf8_encode(s, realloc, memory) {
  if (typeof s !== 'string') throw new TypeError('expected a string');
  
  if (s.length === 0) {
    UTF8_ENCODED_LEN = 0;
    return 1;
  }
  
  let alloc_len = 0;
  let ptr = 0;
  let writtenTotal = 0;
  while (s.length > 0) {
    ptr = realloc(ptr, alloc_len, 1, alloc_len + s.length);
    alloc_len += s.length;
    const { read, written } = UTF8_ENCODER.encodeInto(
    s,
    new Uint8Array(memory.buffer, ptr + writtenTotal, alloc_len - writtenTotal),
    );
    writtenTotal += written;
    s = s.slice(read);
  }
  if (alloc_len > writtenTotal)
  ptr = realloc(ptr, alloc_len, 1, writtenTotal);
  UTF8_ENCODED_LEN = writtenTotal;
  return ptr;
}
export let UTF8_ENCODED_LEN = 0;
