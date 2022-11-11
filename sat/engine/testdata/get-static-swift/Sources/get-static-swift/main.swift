import Suborbital

class GetStaticSwift: Suborbital.Runnable {
    func run(input: String) -> String {
        return Suborbital.GetStaticFile(name: "important.md")
    }
}

Suborbital.Set(runnable: GetStaticSwift())
