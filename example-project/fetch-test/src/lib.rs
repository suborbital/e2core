use suborbital::runnable;
use suborbital::request;
use suborbital::net;
use suborbital::log;

struct FetchTest{}

impl runnable::Runnable for FetchTest {
    fn run(&self, input: Vec<u8>) -> Option<Vec<u8>> {
        let req = match request::from_json(input) {
            Some(r) => r,
            None => return Some(String::from("failed").as_bytes().to_vec())
        };

        let msg = req.state["logme"].as_str().unwrap();
        log::info(msg);

        let url = req.state["url"].as_str().unwrap();

        let data = net::get(url); 

        Some(data)
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &FetchTest = &FetchTest{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
