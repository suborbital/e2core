pub enum Method {
	GET,
	HEAD,
	OPTIONS,
	POST,
	PUT,
	PATCH,
	DELETE,
}

impl From<Method> for i32 {
	fn from(field_type: Method) -> Self {
		match field_type {
			Method::GET => 0,
			Method::HEAD => 1,
			Method::OPTIONS => 2,
			Method::POST => 3,
			Method::PUT => 4,
			Method::PATCH => 5,
			Method::DELETE => 6,
		}
	}
}
