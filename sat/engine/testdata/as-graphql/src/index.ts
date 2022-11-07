// DO NOT EDIT; generated file

import { return_result, return_abort, toFFI, fromFFI, getIdent, setIdent } from "@suborbital/suborbital";
import { run } from "./lib"

export function run_e(ptr: usize, size: i32, ident: i32): void {
  // set the current ident for other API methods to use
	setIdent(ident)

  // read the memory that was passed as input
	var inBuffer = fromFFI(ptr, size)

  // execute the Runnable
	let result = run(inBuffer)

  // return the result to the host
  return_result(changetype<usize>(result), result.byteLength, getIdent())
}

export function allocate(size: i32): usize {
  return heap.alloc(size)
}

export function deallocate(ptr: i32, _: i32): void {
  heap.free(ptr)
}

function abort(message: string | null, fileName: string | null, lineNumber: u32, columnNumber: u32): void {
  let msgFFI = toFFI(String.UTF8.encode(message ? message : ""))
  let fileFFI = toFFI(String.UTF8.encode(fileName ? fileName : ""))

  return_abort(msgFFI.ptr, msgFFI.size, fileFFI.ptr, fileFFI.size, lineNumber, columnNumber, getIdent())
}