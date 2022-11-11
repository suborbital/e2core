use suborbital::runnable::*;
use suborbital::resp;

struct ContentType{}

impl Runnable for ContentType {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let in_string = String::from_utf8(input).unwrap();
    
        resp::content_type("application/json");

        Ok(String::from(format!("hello {}", in_string)).as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &ContentType = &ContentType{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
