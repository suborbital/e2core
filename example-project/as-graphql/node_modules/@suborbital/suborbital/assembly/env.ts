// returns a result to the host
export declare function return_result(ptr: usize, size: i32, ident: i32): void
// logs a message using the hosts' logger
export declare function log_msg(ptr: usize, size: i32, level: i32, ident: i32): void
// makes an http request
export declare function fetch_url(method: i32, url_ptr: usize, url_size: i32, body_ptr: usize, body_size: i32, ident: i32): i32
// makes a GraphQL request
export declare function graphql_query(endpoint_ptr: usize, endpoint_size: i32, query_ptr: usize, query_size: i32, ident: i32): i32
// sets a value in the cache
export declare function cache_set(key_ptr: usize, key_size: i32, value_ptr: usize, value_size: i32, ttl: i32, ident: i32): i32
// gets a value from the cache
export declare function cache_get(key_ptr: usize, key_size: i32, ident: i32): i32
//gets a field from the 'mounted' request
export declare function request_get_field(field_type: i32, key_pointer: usize, key_size: i32, ident: i32): i32
// gets the result of a guest->host FFI call
export declare function get_ffi_result(ptr: usize, ident: i32): i32
// handles the custom abort implementation
export declare function return_abort(msg_ptr: usize, msg_size: i32, file_ptr: usize, file_size: i32, line_num: u32, col_num: u32, ident: i32): void