use suborbital::runnable;
use suborbital::request;
use suborbital::net;

struct FetchTest{}

impl runnable::Runnable for FetchTest {
    fn run(&self, input: Vec<u8>) -> Option<Vec<u8>> {
        let req = match request::from_json(input) {
            Some(r) => r,
            None => return Some(String::from("failed").as_bytes().to_vec())
        };

        let url = req.state["url"].as_str().unwrap();

        let data = net::fetch(url); 

        Some(data)
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &FetchTest = &FetchTest{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
