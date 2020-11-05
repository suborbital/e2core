
#[no_mangle]
pub fn run(input: Vec<u8>) -> Option<Vec<u8>> {
    let in_string = String::from_utf8(input).unwrap();

    Some(String::from(format!("hello {}", in_string)).as_bytes().to_vec())
}