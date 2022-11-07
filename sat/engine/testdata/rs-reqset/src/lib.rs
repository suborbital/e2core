use suborbital::runnable::*;
use suborbital::req;
use suborbital::log;

struct RsReqset{}

impl Runnable for RsReqset {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        match req::set_header("X-REACTR-TEST", "test successful!") {
            Err(e) => log::error(e.message.as_str()),
            Ok(_) => log::info("header set!")
        }
    
        Ok(Vec::new())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &RsReqset = &RsReqset{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
