/*

Package ranger initializes and manages a trails app with sane defaults and custom configuration.

Ranger

The main entrypoint to package ranger is the Ranger type.
A Ranger ought to be constructed with NewRanger and any options required by your application.
Calling NewRanger() - without any options - is valid and fully operational.

(*Ranger).Guide begins a trails app's web server.
By default, (*Ranger).Guide listens on localhost:3000,
assuming either a reverse proxy proxies requests
or only a client application makes direct requests to the trails web server.

Upon calling (*Ranger).Guide, all routes configured up to that point are now active.
Stop that web server with (*Ranger).Shutdown, the signals that method listens for, or,
cancel the context.Context passed in with WithContext.

Configuration

A developer configures a trails app with default options and configurable options,
of which there are too many to enumerate here.
Refer to specific documentation for various functions and methods where necessary.

Some components are directly exposed by configuration functions whereas others are only indirect.
This is distinct from the exported and unexported fields on a *Ranger.

Direct exposure often entails a With* function and a Default* function.
Take WithLogger and DefaultLogger as examples.
DefaultLogger provides a sane logger intended to cover most use cases,
whereas WithLogger enables a developer to bring their own implementation of logger.Logger.
Notably, though, DefaultLogger can still be configured by providing any logger.LoggerOptFn.

Indirect exposure looks like how the template.Parser only has DefaultParser available.
If the template.Parser DefaultParser does not fit a use case,
a resp.WithParser() functional option may be passed
to resp.NewResponder or this package's DefaultParser.

Environment Variables

Constructing a new *Ranger with NewRanger reads from the OS's environment variables.
With that, trails leverages a file called .env to pickup configuration.
A file named .env ought to exist
in the same directory as the compiled trails' app's executable.
Finally, environment variables explicitly set
when running the trails app executable
have precedence over those read from the .env file.

Router

A Ranger uses a router.Router for serving resources for different HTTP requests.
Use WithRouter to first construct a router.Router
and then pass that into NewRanger.
Or, more conveniently, configure the Ranger's Router after NewRanger
and all the way up until calling (*Ranger).Guide.
As a last resort, overwriting the Router initially configured
is possible up until calling (*Ranger).Guide.

*/
package ranger
