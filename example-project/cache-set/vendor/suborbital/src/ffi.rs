use std::slice;

use crate::runnable::HostErr;
use crate::util;
use crate::STATE;

extern "C" {
	fn get_ffi_result(pointer: *const u8, ident: i32) -> i32;
	fn add_ffi_var(name_ptr: *const u8, name_len: i32, val_ptr: *const u8, val_len: i32, ident: i32) -> i32;
}

pub(crate) fn result(size: i32) -> Result<Vec<u8>, HostErr> {
	let mut alloc_size = size;

	// FFI functions return negative values when an error occurs
	if size < 0 {
		if size == -1 {
			return Err(HostErr::new("unknown error returned from host"));
		}

		alloc_size = -size
	}

	// create some memory for the host to write into
	let mut result_mem = Vec::with_capacity(alloc_size as usize);
	let result_ptr = result_mem.as_mut_slice().as_mut_ptr() as *const u8;

	let code = unsafe { get_ffi_result(result_ptr, STATE.ident) };

	// check if it was successful, and then re-build the memory
	if code != 0 {
		return Err(HostErr::new("unknown error returned from host"));
	}

	let data: &[u8] = unsafe { slice::from_raw_parts(result_ptr, alloc_size as usize) };

	if size < 0 {
		let msg = Vec::from(data);
		return Err(HostErr::new(util::to_string(msg).as_str()));
	}

	Ok(Vec::from(data))
}

pub(crate) fn add_var(name: &str, value: &str) {
	unsafe {
		add_ffi_var(
			name.as_ptr(),
			name.len() as i32,
			value.as_ptr(),
			value.len() as i32,
			STATE.ident,
		);
	}
}
