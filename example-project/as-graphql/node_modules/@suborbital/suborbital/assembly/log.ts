import { log_msg } from "./env"
import { getIdent, toFFI } from "./ffi"

export function logDebug(msg: string): void {
	log_raw(msg, 4)
}

export function logInfo(msg: string): void {
	log_raw(msg, 3)
}

export function logWarn(msg: string): void {
	log_raw(msg, 2)
}

export function logErr(msg: string): void {
	log_raw(msg, 4)
}

function log_raw(msg: string, level: i32): void {
	let msgBuf = String.UTF8.encode(msg)
	let msgFFI = toFFI(msgBuf)

	log_msg(msgFFI.ptr, msgFFI.size, level, getIdent())
}