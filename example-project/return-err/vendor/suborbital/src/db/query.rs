
#[derive(Clone)]
pub struct QueryArg {
	pub name: String,
	pub value: String,
}

impl QueryArg {
	pub fn new(name: &str, value: &str) -> Self {
		QueryArg {
			name: String::from(name),
			value: String::from(value),
		}
	}
}

pub enum QueryType {
	SELECT,
	INSERT,
	UPDATE,
	DELETE
}

impl From<QueryType> for i32 {
	fn from(query_type: QueryType) -> Self {
		match query_type {
			QueryType::INSERT => 0,
			QueryType::SELECT => 1,
			QueryType::UPDATE => 2,
			QueryType::DELETE => 3,
		}
	}
}
