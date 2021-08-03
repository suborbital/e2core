# Background

Atmo was born out of a desire to simplify server-side development by making it easy to bootstrap new projects, automatically scale to handle large amounts of traffic, and ensure their security with new technology and best practices.

Atmo embodies the [SUFA design pattern](https://blog.suborbital.dev/how-to-familiarize-yourself-with-a-new-codebase) by making it easy to develop, deploy, and scale your application. To develop applications using Atmo, a declarative file called the Directive is used to describe how to handle requests and events using composable functions. This allows the developer to describe the entirety of their application's behaviour, while only needing to write code for the functions being called. Atmo then uses the Directive to run the desired application, automatically running a web server, job scheduler, and messaging mesh together to form a powerful platform upon which your application can thrive.

The core of Atmo is a job scheduler running WebAssembly modules, which allows running functions written in a variety of languages with near-native performance and massive improvements to security and ease of orchestration.

Atmo scales out using capability groups, which allows groups of Atmo instances to access sensitive resources, scale independently, and ensure your application is able to grow efficiently to handle increases in traffic with ease. Atmo's built in meshing capabilities means that building a flat network of instances is automated, secure, and efficient.

Atmo strives to make everything about developing and deploying your application simple so that you can focus on the hard problems of building your product. Atmo's mission statement is "help developers build applications that are powerful but never complicated".

