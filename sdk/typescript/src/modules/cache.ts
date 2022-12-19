import Base from "./base";

const encoder = new TextEncoder();
const decoder = new TextDecoder();

export default class Cache extends Base {
  get(key: string): string {
    return decoder.decode(this.getBytes(key));
  }

  getBytes(key: string): Uint8Array {
    const resultSize = this.env.cacheGet(key, this.ident);
    return this.ffiResult(resultSize);
  }

  set(key: string, value: string | Uint8Array, ttl: number) {
    let bytes;
    if (typeof value === "string") {
      bytes = encoder.encode(value);
    } else {
      bytes = value;
    }

    this.env.cacheSet(key, bytes, ttl, this.ident);
  }
}
