use crate::ffi;
use crate::runnable::HostErr;
use crate::STATE;

extern "C" {
	fn cache_set(
		key_pointer: *const u8,
		key_size: i32,
		value_pointer: *const u8,
		value_size: i32,
		ttl: i32,
		ident: i32,
	) -> i32;
	fn cache_get(key_pointer: *const u8, key_size: i32, ident: i32) -> i32;
}

pub fn set(key: &str, val: Vec<u8>, ttl: i32) {
	let val_slice = val.as_slice();
	let val_ptr = val_slice.as_ptr();

	unsafe {
		cache_set(
			key.as_ptr(),
			key.len() as i32,
			val_ptr,
			val.len() as i32,
			ttl,
			super::STATE.ident,
		);
	}
}

/// Executes the request via FFI
///
/// Then retreives the result from the host and returns it
pub fn get(key: &str) -> Result<Vec<u8>, HostErr> {
	let result_size = unsafe { cache_get(key.as_ptr(), key.len() as i32, STATE.ident) };

	ffi::result(result_size)
}
