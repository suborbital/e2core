use suborbital::runnable::*;

struct HelloEcho{}

impl Runnable for HelloEcho {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let in_string = String::from_utf8(input).unwrap();

    
        Ok(format!("hello {}", in_string).as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &HelloEcho = &HelloEcho{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
