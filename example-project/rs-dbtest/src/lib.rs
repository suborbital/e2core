use suborbital::runnable::*;
use suborbital::db;
use suborbital::db::query;
use suborbital::log;
use uuid::Uuid;

struct RsDbtest{}

impl Runnable for RsDbtest {
    fn run(&self, _: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let mut args: Vec<query::QueryArg> = Vec::new();
        args.push(query::QueryArg::new("uuid", Uuid::new_v4().to_string().as_str()));
        args.push(query::QueryArg::new("email", "connor@suborbital.dev"));

        match db::insert("PGInsertUser", args) {
            Ok(_) => log::info("insert successful"),
            Err(e) => {
                return Err(RunErr::new(500, e.message.as_str()))
            }
        };
        
        let mut args2: Vec<query::QueryArg> = Vec::new();
        args2.push(query::QueryArg::new("email", "connor@suborbital.dev"));

        match db::select("PGSelectUserWithEmail", args2) {
            Ok(result) => Ok(result),
            Err(e) => {
                Err(RunErr::new(500, e.message.as_str()))
            }
        }
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &RsDbtest = &RsDbtest{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
