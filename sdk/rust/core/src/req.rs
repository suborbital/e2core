pub mod field_type;

use crate::ffi;
use crate::util;
use crate::STATE;
use field_type::FieldType;

extern "C" {
	fn request_get_field(field_type: i32, key_pointer: *const u8, key_size: i32, ident: i32) -> i32;
	fn request_set_field(
		field_type: i32,
		key_pointer: *const u8,
		key_size: i32,
		val_pointer: *const u8,
		val_size: i32,
		ident: i32,
	) -> i32;
}

pub fn method() -> String {
	get_field(FieldType::Meta.into(), "method").map_or("".into(), util::to_string)
}

pub fn set_method(val: &str) -> Result<(), super::runnable::HostErr> {
	set_field(FieldType::Meta.into(), "method", val)
}

pub fn url() -> String {
	get_field(FieldType::Meta.into(), "url").map_or("".into(), util::to_string)
}

pub fn set_url(val: &str) -> Result<(), super::runnable::HostErr> {
	set_field(FieldType::Meta.into(), "url", val)
}

pub fn id() -> String {
	get_field(FieldType::Meta.into(), "id").map_or("".into(), util::to_string)
}

pub fn body_raw() -> Vec<u8> {
	get_field(FieldType::Meta.into(), "body").unwrap_or_default()
}

pub fn set_body(val: &str) -> Result<(), super::runnable::HostErr> {
	set_field(FieldType::Body.into(), "body", val)
}

pub fn body_field(key: &str) -> String {
	get_field(FieldType::Body.into(), key).map_or("".into(), util::to_string)
}

pub fn set_body_field(key: &str, val: &str) -> Result<(), super::runnable::HostErr> {
	set_field(FieldType::Body.into(), key, val)
}

pub fn header(key: &str) -> String {
	get_field(FieldType::Header.into(), key).map_or("".into(), util::to_string)
}

pub fn set_header(key: &str, val: &str) -> Result<(), super::runnable::HostErr> {
	set_field(FieldType::Header.into(), key, val)
}

pub fn url_param(key: &str) -> String {
	get_field(FieldType::Params.into(), key).map_or("".into(), util::to_string)
}

pub fn set_url_param(key: &str, val: &str) -> Result<(), super::runnable::HostErr> {
	set_field(FieldType::Params.into(), key, val)
}

pub fn state(key: &str) -> Option<String> {
	get_field(FieldType::State.into(), key).map(util::to_string)
}

pub fn set_state(key: &str, val: &str) -> Result<(), super::runnable::HostErr> {
	set_field(FieldType::State.into(), key, val)
}

pub fn state_raw(key: &str) -> Option<Vec<u8>> {
	get_field(FieldType::State.into(), key)
}

pub fn query_param(key: &str) -> String {
	get_field(FieldType::Query.into(), key).map_or("".into(), util::to_string)
}

/// Executes the request via FFI
///
/// Then retreives the result from the host and returns it
fn get_field(field_type: i32, key: &str) -> Option<Vec<u8>> {
	let result_size = unsafe { request_get_field(field_type, key.as_ptr(), key.len() as i32, STATE.ident) };

	ffi::result(result_size).map_or(None, Option::from)
}

fn set_field(field_type: i32, key: &str, val: &str) -> Result<(), super::runnable::HostErr> {
	// make the request over FFI
	let result_size = unsafe {
		request_set_field(
			field_type,
			key.as_ptr(),
			key.len() as i32,
			val.as_ptr(),
			val.len() as i32,
			super::STATE.ident,
		)
	};

	// retreive the result from the host and return it
	match ffi::result(result_size) {
		Ok(_) => Ok(()),
		Err(e) => Err(e),
	}
}
