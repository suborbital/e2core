import { cache_set, cache_get } from "./env"
import { ffi_result, getIdent, toFFI } from "./ffi"

export function cacheGet(key: string): ArrayBuffer {
	let keyBuf = String.UTF8.encode(key)
	let keyFFI = toFFI(keyBuf)

	let result_size = cache_get(keyFFI.ptr, keyFFI.size, getIdent())

	let result = ffi_result(result_size)

	return result
}

export function cacheSet(key: string, value: ArrayBuffer, ttl: i32): void {
	let keyBuf = String.UTF8.encode(key)
	let keyFFI = toFFI(keyBuf)

	let valFFI = toFFI(value)

	cache_set(keyFFI.ptr, keyFFI.size, valFFI.ptr, valFFI.size, ttl, getIdent())
}