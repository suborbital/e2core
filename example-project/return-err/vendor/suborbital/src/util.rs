pub fn to_string(input: Vec<u8>) -> String {
	String::from_utf8(input).unwrap_or_default()
}

pub fn to_vec(input: String) -> Vec<u8> {
	input.as_bytes().to_vec()
}

pub fn str_to_vec(input: &str) -> Vec<u8> {
	String::from(input).as_bytes().to_vec()
}
