pub mod query;

use crate::ffi;
use crate::runnable;
use crate::STATE;

use query::{QueryArg, QueryType};

extern "C" {
	fn db_exec(query_type: i32, name_ptr: *const u8, name_size: i32, ident: i32) -> i32;
}

// insert executes the pre-loaded database query with the name <name>,
// and passes the arguments defined by <args>
//
// the return value is the inserted auto-increment ID from the query result, if any,
// formatted as JSON with the key `lastInsertID`
pub fn insert(name: &str, args: Vec<QueryArg>) -> Result<Vec<u8>, runnable::HostErr> {
	args.iter().for_each(|arg| ffi::add_var(&arg.name, &arg.value));

	let result_size = unsafe { db_exec(QueryType::INSERT.into(), name.as_ptr(), name.len() as i32, STATE.ident) };

	// retreive the result from the host and return it
	ffi::result(result_size)
}

// update executes the pre-loaded database query with the name <name>,
// and passes the arguments defined by <args>
//
// the return value is number of rows affected by the query,
// formatted as JSON with the key `rowsAffected`
pub fn update(name: &str, args: Vec<QueryArg>) -> Result<Vec<u8>, runnable::HostErr> {
	args.iter().for_each(|arg| ffi::add_var(&arg.name, &arg.value));

	let result_size = unsafe { db_exec(QueryType::UPDATE.into(), name.as_ptr(), name.len() as i32, STATE.ident) };

	// retreive the result from the host and return it
	ffi::result(result_size)
}

// update executes the pre-loaded database query with the name <name>,
// and passes the arguments defined by <args>
//
// the return value is number of rows affected by the query,
// formatted as JSON with the key `rowsAffected`
pub fn delete(name: &str, args: Vec<QueryArg>) -> Result<Vec<u8>, runnable::HostErr> {
	args.iter().for_each(|arg| ffi::add_var(&arg.name, &arg.value));

	let result_size = unsafe { db_exec(QueryType::DELETE.into(), name.as_ptr(), name.len() as i32, STATE.ident) };

	// retreive the result from the host and return it
	ffi::result(result_size)
}

// insert executes the pre-loaded database query with the name <name>,
// and passes the arguments defined by <args>
//
// the return value is the query result formatted as JSON, with each column name as a top-level key
pub fn select(name: &str, args: Vec<QueryArg>) -> Result<Vec<u8>, runnable::HostErr> {
	args.iter().for_each(|arg| ffi::add_var(&arg.name, &arg.value));

	let result_size = unsafe { db_exec(QueryType::SELECT.into(), name.as_ptr(), name.len() as i32, STATE.ident) };

	// retreive the result from the host and return it
	ffi::result(result_size)
}
