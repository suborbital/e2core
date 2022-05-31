use suborbital::runnable::*;
use suborbital::req;
use suborbital::http;
use suborbital::log;

struct FetchTest{}

impl Runnable for FetchTest {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {

        let msg = req::state("logme").unwrap_or_default();
        log::info(msg.as_str());

        let url = req::state("url").unwrap_or_default();

        match http::get(url.as_str(), None) {
            Ok(data) => Ok(data),
            Err(e) => Err(RunErr::new(500, e.message.as_str()))
        }
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &FetchTest = &FetchTest{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
