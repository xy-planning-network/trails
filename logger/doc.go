/*

Package logger provides logging functionality to a trails app by defining the required behavior in [Logger]
and providing an implementation of it with [TrailsLogger].

# Overview

The Logger interface outputs messages at certain levels of importance.
LogLevel is the type to use to represent those levels.
An implementation of Logger may be initialized at a certain [LogLevel]
and only emit messages at or above that level of importance.
For example, [TrailsLogger] accepts a [LogLevel],
and if initialized with [LogLevelWarn],
only [*TrailsLogger.Warn], [*TrailsLogger.Error], and [*TrailsLogger.Fatal] produce messages.

# TrailsLogger

The [TrailsLogger] provides all the logging functionality needed for a trails app.
It is the implementation of [Logger] returned by the [New] function.

Log messages emitted by [TrailsLogger] are composed of a few parts:
	- timestamp
	- log level
	- call site
	- message
	- log context

Here's an example:
	2022/04/28 15:55:21 [DEBUG] web/dashboard_handler.go:43 'such fun!' log_context: "{"user":"{"id": 1, "email": "trails@example.com"}}"

The file, line number, and parent directory of where a [TrailsLogger] comprise the call site.
The message is the actual string passed into the [TrailsLogger] method, in this example, [*TrailsLogger.Debug].
Lastly, the log context is a JSON-encoded [*LogContext].
The last component allows for including additional data inessential to the message proper,
but provides a fuller picture of the application state at the time of logging.

# SkipLogger

Sometimes, especially with internal packages, the file and line number in a log needs to be configurable.
[SkipLogger] provides additional configuration functionality by setting the number of frames to skip
back in order to reach the desired caller.
*/
package logger
