import { get_ffi_result } from "./env"

var current_ident: i32 = 0;

export function setIdent(ident: i32): void {
	current_ident = ident
}

export function getIdent(): i32 {
	return current_ident
}

export function ffi_result(size: i32): ArrayBuffer {
	if (size < 0) {
		return new ArrayBuffer(0)
	}

	let result_ptr = heap.alloc(size)

	let code = get_ffi_result(result_ptr, current_ident)
	if (code != 0) {
		return new ArrayBuffer(0)
	}

	return fromFFI(result_ptr, size)
}

export function fromFFI(ptr: usize, len: i32): ArrayBuffer {
	let mem = new Uint8Array(len)

	for (let i = 0; i < len; i++) {
		mem[i] = load<u8>(ptr + i);
	}

	return mem.buffer
}

export class ffiValue {
	ptr: usize
	size: i32

	constructor(ptr: usize, size: i32) {
		this.ptr = ptr
		this.size = size
	}
}

export function toFFI(val: ArrayBuffer): ffiValue {
	return new ffiValue(changetype<usize>(val), val.byteLength)
}