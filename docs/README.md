# Welcome

### Welcome to the Atmo guide

Building web services should be simple. Atmo makes it easy to create a powerful server application without needing to worry about scalability, infrastructure, or complex networking.

Atmo enables you to write small self-contained functions called **Runnables** using a variety of languages, and define your business logic by **declaratively composing** them. Atmo then automatically scales out a **flat network** of instances to handle traffic using its meshed message bus and embedded job scheduler. Atmo can handle request-based traffic, and soon will be able to handle events sourced from various systems like Kafka or EventBridge.

Atmo is currently focused on building web services, particularly APIs. The project will evolve over time to include things like static files, templating, and more.

The Atmo **Directive** is a YAML file wherein you declare your application's behaviour. Because the Directive can describe everything you need to make your application work \(including routes, logic, and more\), there is no need to write boilerplate ever again.

Atmo is not a library, but rather a self-hosted platform that uses a bundle containing your Runnables and Directive to automatically run your application.

With Atmo, you only need to do three things: 

1. Write self-contained, composable functions 

2. Declare how you want Atmo to handle requests by creating a "Directive" 

3. Build and deploy your Runnable bundle.

## Coming soon

Further functionality is incoming along with improved docs, more examples, and an improved Directive format. Visit [the Suborbital website](https://suborbital.dev) to sign up for email updates related to new versions of Atmo.

## Status

Atmo is currently in **beta**, and is intended to be the flagship project in the Suborbital Development Platform.

Atmo is built atop [Vektor](https://github.com/suborbital/vektor), [Hive](https://github.com/suborbital/hive) and [Grav](https://github.com/suborbital/grav).

Copyright Suborbital contributors 2020.

