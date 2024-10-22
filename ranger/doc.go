/*
Package ranger initializes and manages a trails app with sane defaults.

# Ranger

The main entrypoint to package ranger is the [Ranger] type.
A [Ranger] ought to be constructed with [New] using a [Config].
A [Config] must be set with the concrete type for the user of your application.
That user must implement [RangerUser]

[*Ranger.Guide] begins a trails app's web server.
By default, [*Ranger.Guide] listens on [DefaultHost]:[DefaultPort] (localhost:3000),
assuming either a reverse proxy proxies requests
or only a client application makes direct requests to the trails web server.

Upon calling [*Ranger.Guide], all routes configured up to that point are now active.
Stop that web server with [*Ranger.Shutdown],
call the context.CancelFunc returned by [*Ranger.Cancel],
or send a signal [*Ranger.Guide] listens for.

# Configuration

A developer configures a trails app through environment variables
and by setting fields on [Config].
For environment variables, required values can be discovered by inspecting the errors [New] returns.

Environment variables ought to be set in a file called ".env"
found at the same directory the application is executed from.

Here are the available environment variables.
  - APP_DESCRIPTION: a short description of the application
  - APP_TITLE: a short title for the application
  - ASSETS_URL: the base URL the application serves client-side assets over
  - BASE_URL: the base URL the application runs on; replaces HOST & PORT
  - CONTACT_US: the email address end users can contact XYPN at; default: hello@xyplanningnetwork.com
  - DATABASE_HOST: the host the database is running on; default: localhost
  - DATABASE_NAME: the name of the database
  - DATABASE_PORT: the port the database is listening on; default: 5432
  - DATABASE_URL: the fully-qualified connection string for connecting to the database; replaces all other DATABASE_* env vars
  - DATABASE_USER: the user for authenticating a connection to the database
  - DATABSE_PASSWORD: the password for authenticating a connection to the database
  - ENVIRONMENT: the environment the application is running in; cf. [trails.Environment]
  - HOST: the host the application is running on; default: localhost
  - LOG_LEVEL: the level at which to begin logging; default: INFO; cf. [logger.LogLevel]
  - PORT: the port the application should listen on; default: :3000
  - SERVER_IDLE_TIMEOUT: the timeout - as understood by [time.ParseDuration] - for idiling between requests when using keep-alives; default: 120s
  - SERVER_READ_TIMEOUT: the timeout - as understood by [time.ParseDuration] - for reading HTTP requests; default: 5s
  - SERVER_WRITE_TIMEOUT: the timeout - as understood by [time.ParseDuration] - for writing HTTP responses; default: 5s
  - SESSION_AUTH_KEY: a hex-encoded key for authenticating cookies; cf. [encoding/hex]
  - SESSION_ENCRYPTION_KEY: a hex-encoded key for encrypting cookies; cf. [encoding/hex]
  - SESSION_DOMAIN: the host the application is served over for setting as the cookie's domain; default: the hostname of BASE_URL
*/
package ranger
