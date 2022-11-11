use suborbital::runnable::*;
use suborbital::util;
use suborbital::log;

struct HelloEcho{}

impl Runnable for HelloEcho {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let message = util::to_string(input);
        let output = format!("hello {}", message);

        log::info(output.as_str());

        Ok(output.as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &HelloEcho = &HelloEcho{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}