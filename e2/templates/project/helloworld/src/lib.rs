use suborbital::runnable::*;

struct HelloWorld{}

impl Runnable for HelloWorld {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let in_string = String::from_utf8(input).unwrap();
    
        Ok(String::from(format!("hello {}", in_string)).as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &HelloWorld = &HelloWorld{};

#[no_mangle]
pub extern fn init() {
    use_runnable(RUNNABLE);
}
