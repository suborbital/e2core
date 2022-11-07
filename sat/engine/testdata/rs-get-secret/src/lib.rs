use suborbital::runnable::*;
use suborbital::secrets;

struct RsGetSecret{}

impl Runnable for RsGetSecret {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let key = String::from_utf8(input).unwrap();
    
        let val = match secrets::get_val(key.as_str()) {
            Ok(val) => val,
            Err(_) => String::from("")
        };

        Ok(val.as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &RsGetSecret = &RsGetSecret{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
