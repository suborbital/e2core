use crate::STATE;

extern "C" {
	fn log_msg(pointer: *const u8, result_size: i32, level: i32, ident: i32);
}

pub fn debug(msg: &str) {
	log_at_level(msg, 4)
}

pub fn info(msg: &str) {
	log_at_level(msg, 3)
}

pub fn warn(msg: &str) {
	log_at_level(msg, 2)
}

pub fn error(msg: &str) {
	log_at_level(msg, 1)
}

fn log_at_level(msg: &str, level: i32) {
	unsafe { log_msg(msg.as_ptr(), msg.len() as i32, level, STATE.ident) };
}
