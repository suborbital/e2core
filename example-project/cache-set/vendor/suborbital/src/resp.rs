use crate::STATE;

extern "C" {
	fn resp_set_header(key_pointer: *const u8, key_size: i32, val_pointer: *const u8, val_size: i32, ident: i32);
}

pub fn set_header(key: &str, val: &str) {
	unsafe {
		resp_set_header(
			key.as_ptr(),
			key.len() as i32,
			val.as_ptr(),
			val.len() as i32,
			STATE.ident,
		)
	};
}

pub fn content_type(ctype: &str) {
	set_header("Content-Type", ctype);
}
