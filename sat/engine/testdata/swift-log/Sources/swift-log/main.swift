import Suborbital
import Foundation

class SwiftEcho: Suborbital.Runnable {
    func run(input: String) -> String {
        Suborbital.LogInfo(msg: "input: \(input.utf8)")

        let method = Suborbital.ReqMethod()
        let url = Suborbital.ReqURL()
        let helloTo = Suborbital.State(key: "hello")
        let baz = Suborbital.ReqBodyField(key: "baz") //testing it doesn't crash when something doesn't exist
        let username = Suborbital.ReqBodyField(key: "username")
        let body = Suborbital.ReqBodyRaw()
        
        Suborbital.LogInfo(msg: "url: \(url)")
        Suborbital.LogInfo(msg: "method: \(method)")
        Suborbital.LogInfo(msg: "helloTo: \(helloTo)")
        Suborbital.LogInfo(msg: "baz: \(baz)")
        Suborbital.LogInfo(msg: "username: \(username)")
        Suborbital.LogInfo(msg: "body: \(body)")

        return "hello " + helloTo
    }
}

Suborbital.Set(runnable: SwiftEcho())
