use suborbital::runnable::*;
use serde::{Serialize, Deserialize};

#[derive(Serialize, Deserialize, Debug)]
struct RequestMessage {
    text: String
}

struct HelloWorld{}

impl Runnable for HelloWorld {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let in_string = String::from_utf8(input).unwrap();
    
        let request_message: RequestMessage = serde_json::from_str(&in_string).unwrap();
        
        Ok(format!("👋 Hello World {} 🌍", request_message.text).as_bytes().to_vec())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &HelloWorld = &HelloWorld{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
