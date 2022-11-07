import Suborbital

class HelloSwift: Suborbital.Runnable {
    func run(input: String) -> String {
        return "hello " + input
    }
}

Suborbital.Set(runnable: HelloSwift())
