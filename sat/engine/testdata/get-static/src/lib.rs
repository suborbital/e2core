use suborbital::runnable::*;
use suborbital::file;

struct GetStatic{}

impl Runnable for GetStatic {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let in_string = String::from_utf8(input).unwrap();
    
        let file = file::get_static(in_string.as_str())
            .unwrap_or("".as_bytes().to_vec());

        Ok(file)
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &GetStatic = &GetStatic{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
