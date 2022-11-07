import { cacheGet } from "@suborbital/suborbital"

export function run(_: ArrayBuffer): ArrayBuffer {
	let resp = cacheGet("name")
  
	return resp.Result
}