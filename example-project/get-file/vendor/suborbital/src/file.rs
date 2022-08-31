use crate::ffi;
use crate::STATE;

extern "C" {
	fn get_static_file(name_ptr: *const u8, name_size: i32, ident: i32) -> i32;
}

/// Executes the request via FFI
///
/// Then retreives the result from the host and returns it
pub fn get_static(name: &str) -> Option<Vec<u8>> {
	let result_size = unsafe { get_static_file(name.as_ptr(), name.len() as i32, STATE.ident) };

	match ffi::result(result_size) {
		Ok(res) => Some(res),
		Err(_) => None,
	}
}
