use suborbital::runnable::*;
use suborbital::req;
use suborbital::http;
use suborbital::util;

struct Fetch{}

impl Runnable for Fetch {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let url = req::body_raw();
        
        let res = match http::get(util::to_string(url).as_str(), None) {
            Ok(res) => res,
            Err(e) => return Err(RunErr::new(1, e.message.as_str()))
        };

        return Ok(res)
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &Fetch = &Fetch{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
