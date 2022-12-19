pub enum FieldType {
	Meta,
	Body,
	Header,
	Params,
	State,
	Query,
}

impl From<FieldType> for i32 {
	fn from(field_type: FieldType) -> Self {
		match field_type {
			FieldType::Meta => 0,
			FieldType::Body => 1,
			FieldType::Header => 2,
			FieldType::Params => 3,
			FieldType::State => 4,
			FieldType::Query => 5,
		}
	}
}
