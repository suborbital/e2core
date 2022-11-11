import { httpGet } from "@suborbital/suborbital"

export function run(input: ArrayBuffer): ArrayBuffer {
	let url = String.UTF8.decode(input)

	let resp = httpGet(url, null)
  
	return resp.Result
}