use suborbital::runnable::*;

struct ReturnErr{}

impl Runnable for ReturnErr {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        Err(RunErr::new(400, "job failed"))
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &ReturnErr = &ReturnErr{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
