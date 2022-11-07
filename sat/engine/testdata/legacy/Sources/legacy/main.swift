import Suborbital

class HelloSwift: Suborbital.Runnable {
    func run(input: String) -> String {
        return "hello " + input
    }
}

@_cdecl("init")
func `init`() {
    Suborbital.Set(runnable: HelloSwift())
}
