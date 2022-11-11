use suborbital::runnable::*;

struct PanicAtTheDisco{}

impl Runnable for PanicAtTheDisco {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        panic!("we had such high hopes")
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &PanicAtTheDisco = &PanicAtTheDisco{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
