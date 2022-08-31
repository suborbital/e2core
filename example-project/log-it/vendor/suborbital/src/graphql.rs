use crate::ffi;
use crate::runnable::HostErr;
use crate::STATE;

extern "C" {
	fn graphql_query(
		endpoint_pointer: *const u8,
		endpoint_size: i32,
		query_pointer: *const u8,
		query_size: i32,
		ident: i32,
	) -> i32;
}

/// Retreives the result from the host and returns it
pub fn query(endpoint: &str, query: &str) -> Result<Vec<u8>, HostErr> {
	let endpoint_size = endpoint.len() as i32;
	let query_size = query.len() as i32;

	let result_size = unsafe {
		graphql_query(
			endpoint.as_ptr(),
			endpoint_size,
			query.as_ptr(),
			query_size,
			STATE.ident,
		)
	};

	ffi::result(result_size)
}
