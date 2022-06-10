use suborbital::runnable::*;
use suborbital::{log, req};

struct LogIt{}

impl Runnable for LogIt {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let in_string = String::from_utf8(input).unwrap();

        log::info(in_string.as_str());

        let method = req::method();
        if method == "SCHED" {
            log::info("running on a schedule");
        }
    
        Ok(String::from(format!("hello {}", in_string)).as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &LogIt = &LogIt{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
