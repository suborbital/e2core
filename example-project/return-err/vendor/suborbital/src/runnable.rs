use crate::STATE;
use std::mem;

extern "C" {
	fn return_result(result_pointer: *const u8, result_size: i32, ident: i32);
	fn return_error(code: i32, result_pointer: *const u8, result_size: i32, ident: i32);
}

pub struct RunErr {
	pub code: i32,
	pub message: String,
}

impl RunErr {
	pub fn new(code: i32, msg: &str) -> Self {
		RunErr {
			code,
			message: msg.into(),
		}
	}
}

pub struct HostErr {
	pub message: String,
}

impl HostErr {
	pub fn new(msg: &str) -> Self {
		HostErr {
			message: String::from(msg),
		}
	}
}

pub trait Runnable {
	fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr>;
}

pub fn use_runnable(runnable: &'static dyn Runnable) {
	unsafe {
		STATE.runnable = Some(runnable);
	}
}

/// # Safety
///
/// We hand over the the pointer to the allocated memory.
/// Caller has to ensure that the memory gets freed again.
#[no_mangle]
pub unsafe extern "C" fn allocate(size: i32) -> *const u8 {
	let mut buffer = Vec::with_capacity(size as usize);

	let pointer = buffer.as_mut_ptr();

	mem::forget(buffer);

	pointer as *const u8
}

/// # Safety
#[no_mangle]
pub unsafe extern fn deallocate(pointer: *mut u8, size: i32) {
	drop(Vec::from_raw_parts(pointer, size as usize, size as usize))
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn run_e(pointer: *mut u8, size: i32, ident: i32) {
	STATE.ident = ident;

	// rebuild the memory into something usable
	let in_bytes = Vec::from_raw_parts(pointer, size as usize, size as usize);

	match execute_runnable(STATE.runnable, in_bytes) {
		Ok(data) => {
			return_result(data.as_ptr(), data.len() as i32, ident);
		}
		Err(RunErr { code, message }) => {
			return_error(code, message.as_ptr(), message.len() as i32, ident);
		}
	}
}

fn execute_runnable(runnable: Option<&dyn Runnable>, data: Vec<u8>) -> Result<Vec<u8>, RunErr> {
	if let Some(runnable) = runnable {
		return runnable.run(data);
	}
	Err(RunErr::new(-1, "No runnable set"))
}
