use suborbital::{log, req, runnable};

struct LogIt{}

impl runnable::Runnable for LogIt {
    fn run(&self, input: Vec<u8>) -> Option<Vec<u8>> {
        let in_string = String::from_utf8(input).unwrap();

        log::info(in_string.as_str());

        let method = req::method();
        if method == "SCHED" {
            log::info("running on a schedule");
        }
    
        Some(String::from(format!("hello {}", in_string)).as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &LogIt = &LogIt{};

#[no_mangle]
pub extern fn init() {
    runnable::set(RUNNABLE);
}
