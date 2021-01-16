use suborbital::runnable;
use suborbital::req;
use suborbital::http;
use suborbital::log;

struct FetchTest{}

impl runnable::Runnable for FetchTest {
    fn run(&self, _: Vec<u8>) -> Option<Vec<u8>> {

        let msg = req::state("logme");
        log::info(msg.as_str());

        let url = req::state("url");

        let data = http::get(url.as_str()); 

        Some(data)
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &FetchTest = &FetchTest{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
