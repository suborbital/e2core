use suborbital::runnable::*;
use suborbital::req;
use suborbital::http;
use suborbital::log;

struct FetchTest{}

impl Runnable for FetchTest {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {

        let msg = req::state("logme");
        log::info(msg.as_str());

        let url = req::state("url");

        let data = http::get(url.as_str(), None); 

        Ok(data)
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &FetchTest = &FetchTest{};

#[no_mangle]
pub extern fn init() {
    use_runnable(RUNNABLE);
}
