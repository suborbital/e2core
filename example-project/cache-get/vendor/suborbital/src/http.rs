pub mod method;

use std::collections::BTreeMap;

use crate::ffi;
use crate::runnable::HostErr;
use crate::STATE;
use method::Method;

extern "C" {
	fn fetch_url(
		method: i32,
		url_pointer: *const u8,
		url_size: i32,
		body_pointer: *const u8,
		body_size: i32,
		ident: i32,
	) -> i32;
}

pub fn get(url: &str, headers: Option<BTreeMap<&str, &str>>) -> Result<Vec<u8>, HostErr> {
	do_request(Method::GET.into(), url, None, headers)
}

pub fn head(url: &str, headers: Option<BTreeMap<&str, &str>>) -> Result<Vec<u8>, HostErr> {
	do_request(Method::HEAD.into(), url, None, headers)
}

pub fn options(url: &str, headers: Option<BTreeMap<&str, &str>>) -> Result<Vec<u8>, HostErr> {
	do_request(Method::OPTIONS.into(), url, None, headers)
}

pub fn post(url: &str, body: Option<Vec<u8>>, headers: Option<BTreeMap<&str, &str>>) -> Result<Vec<u8>, HostErr> {
	do_request(Method::POST.into(), url, body, headers)
}

pub fn put(url: &str, body: Option<Vec<u8>>, headers: Option<BTreeMap<&str, &str>>) -> Result<Vec<u8>, HostErr> {
	do_request(Method::PUT.into(), url, body, headers)
}

pub fn patch(url: &str, body: Option<Vec<u8>>, headers: Option<BTreeMap<&str, &str>>) -> Result<Vec<u8>, HostErr> {
	do_request(Method::PATCH.into(), url, body, headers)
}

pub fn delete(url: &str, headers: Option<BTreeMap<&str, &str>>) -> Result<Vec<u8>, HostErr> {
	do_request(Method::DELETE.into(), url, None, headers)
}

/// Executes the request via FFI
///
/// Then retreives the result from the host and returns it
///
/// > Remark: The URL gets encoded with headers added on the end, seperated by ::
/// > eg. https://google.com/somepage::authorization:bearer qdouwrnvgoquwnrg::anotherheader:nicetomeetyou
fn do_request(
	method: i32,
	url: &str,
	body: Option<Vec<u8>>,
	headers: Option<BTreeMap<&str, &str>>,
) -> Result<Vec<u8>, HostErr> {
	let header_string = render_header_string(headers);

	let url_string = match header_string {
		Some(h) => format!("{}::{}", url, h),
		None => String::from(url),
	};

	let body_pointer: *const u8;
	let mut body_size: i32 = 0;

	match body {
		Some(b) => {
			let body_slice = b.as_slice();
			body_pointer = body_slice.as_ptr();
			body_size = b.len() as i32;
		}
		None => body_pointer = std::ptr::null::<u8>(),
	}

	let result_size = unsafe {
		fetch_url(
			method,
			url_string.as_str().as_ptr(),
			url_string.len() as i32,
			body_pointer,
			body_size,
			STATE.ident,
		)
	};

	ffi::result(result_size)
}

fn render_header_string(headers: Option<BTreeMap<&str, &str>>) -> Option<String> {
	let mut rendered: String = String::from("");

	let header_map = headers?;

	for key in header_map.keys() {
		rendered.push_str(key);
		rendered.push(':');

		let val: &str = match header_map.get(key) {
			Some(v) => v,
			None => "",
		};

		rendered.push_str(val);
		rendered.push_str("::")
	}

	Some(String::from(rendered.trim_end_matches("::")))
}
