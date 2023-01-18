use suborbital::runnable::*;
use suborbital::graphql::*;
use suborbital::log::*;
use suborbital::util;

struct RsGraqhql{}

impl Runnable for RsGraqhql {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let result = match query("https://api.github.com/graphql", "{ repository (owner: \"suborbital\", name: \"reactr\") { name, nameWithOwner }}") {
            Ok(response) => {
                info(util::to_string(response.clone()).as_str());
                response
            }
            Err(e) => {
                error(e.message.as_str());
                return Err(RunErr::new(1, e.message.as_str()))
            }
        };
    
        Ok(result)
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &RsGraqhql = &RsGraqhql{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
