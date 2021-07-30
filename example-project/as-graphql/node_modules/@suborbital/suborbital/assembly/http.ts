import { fetch_url } from "./env"
import { ffi_result, getIdent, toFFI } from "./ffi"

export function httpGet(url: string, headers: Map<string, string> | null): ArrayBuffer {
	return do_request(method_get, url, new ArrayBuffer(0), headers)
}

export function httpPost(url: string, body: ArrayBuffer, headers: Map<string, string> | null): ArrayBuffer {
	return do_request(method_post, url, body, headers)
}

export function httpPatch(url: string, body: ArrayBuffer, headers: Map<string, string> | null): ArrayBuffer {
	return do_request(method_patch, url, body, headers)
}

export function httpDelete(url: string, headers: Map<string, string> | null): ArrayBuffer {
	return do_request(method_delete, url, new ArrayBuffer(0), headers)
}

const method_get = 1
const method_post = 2
const method_patch = 3
const method_delete = 4

function do_request(method: i32, url: string, body: ArrayBuffer, headers: Map<string, string> | null): ArrayBuffer {
	var headerString = ""
	if (headers != null) {
		headerString = renderHeaderString(headers)
	}

	let urlBuf = String.UTF8.encode(url + headerString)
	let urlFFI = toFFI(urlBuf)

	let bodyFFI = toFFI(body)

	let result_size = fetch_url(method, urlFFI.ptr, urlFFI.size, bodyFFI.ptr, bodyFFI.size, getIdent())

	let result = ffi_result(result_size)

	return result
}

function renderHeaderString(headers: Map<string,string>): string {
	var rendered: string = ""
	let keys = headers.keys()
	
	for (let i = 0; i < keys.length; ++i) {
		let key = keys[i]
		let val = headers.get(key)

		rendered += "::" + key + ":" + val
	}

	return rendered
}