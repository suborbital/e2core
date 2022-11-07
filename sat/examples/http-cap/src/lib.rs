use suborbital::runnable::*;
use suborbital::http;
use suborbital::util;

struct Fetch{}

impl Runnable for Fetch {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let url = util::to_string(input);
        
        let _ = match http::get(url.as_str(), None) {
            Ok(res) => return Ok(res),
            Err(e) => return Err(RunErr::new(1, e.message.as_str()))
        };
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &Fetch = &Fetch{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
