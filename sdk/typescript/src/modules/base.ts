import FFI from "./ffi";

const decoder = new TextDecoder();

export default class Base {
  get env() {
    return FFI.env;
  }

  get ident() {
    return FFI.ident;
  }

  ffiResult(resultSize: number): Uint8Array {
    let isError = false;
    if (resultSize < 0) {
      isError = true;
      resultSize *= -1;
    }

    // Allocate memory to store the response
    // @ts-ignore
    const ptr = this.env._exports.canonical_abi_realloc(0, 0, 1, resultSize);

    // Write the response to memory
    this.env.getFfiResult(ptr, this.ident);

    // Create a view of the response
    const result = new Uint8Array(
      // @ts-ignore
      this.env._exports.memory.buffer,
      ptr,
      resultSize
    );

    if (isError) {
      const message = decoder.decode(result);
      throw new Error(message);
    }

    // TODO: We are leaking memory here since we're not calling `canonical_abi_free`
    // Copy the contents of the array to avoid exposing the whole wasm memory
    return new Uint8Array(result);
  }
}
