import { request_get_field } from "./env";
import { ffi_result, getIdent, toFFI } from "./ffi"

const FIELD_TYPE_META: i32 = 0
const FIELD_TYPE_BODY: i32 = 1
const FIELD_TYPE_HEADER: i32 = 2
const FIELD_TYPE_PARAMS: i32 = 3
const FIELD_TYPE_STATE: i32 = 4

export function reqMethod(): string {
	let result = get_field(FIELD_TYPE_META, "method")
	return String.UTF8.decode(result)
}

export function reqURL(): string {
	let result = get_field(FIELD_TYPE_META, "url")
	return String.UTF8.decode(result)
}

export function reqID(): string {
	let result = get_field(FIELD_TYPE_META, "id")
	return String.UTF8.decode(result)
}

export function reqBody(): ArrayBuffer {
	return get_field(FIELD_TYPE_META, "body")
}

export function reqBodyField(key: string): string {
	let result = get_field(FIELD_TYPE_BODY, key)
	return String.UTF8.decode(result)
}

export function reqHeader(key: string): string {
	let result = get_field(FIELD_TYPE_HEADER, key)
	return String.UTF8.decode(result)
}


export function reqURLParam(key: string): string {
	let result = get_field(FIELD_TYPE_PARAMS, key)
	return String.UTF8.decode(result)
}

export function reqState(key: string): string {
	let result = get_field(FIELD_TYPE_STATE, key)
	return String.UTF8.decode(result)
}

export function reqStateRaw(key: string): ArrayBuffer {
	return get_field(FIELD_TYPE_STATE, key)
}

function get_field(field_type: i32, key: string): ArrayBuffer {
	let keyBuf = String.UTF8.encode(key)
	let keyFFI = toFFI(keyBuf)

	let result_size = request_get_field(field_type, keyFFI.ptr, keyFFI.size, getIdent())

	let result = ffi_result(result_size)

	return result
}