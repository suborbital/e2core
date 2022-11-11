import { cacheSet, logInfo } from "@suborbital/suborbital"

export function run(input: ArrayBuffer): ArrayBuffer {
	let val = String.UTF8.decode(input)

	logInfo("setting name:" + val)

	cacheSet("name", input, 0)
  
	return new ArrayBuffer(0)
}