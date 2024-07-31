/*
Package req provides ergonomics for handling an HTTP request.

Package req provides a helper for parsing payloads in an HTTP request.
It supports JSON-encoded payloads and payloads encoded in query parameters.
In both cases, package req expects to parse payloads into a pointer to a struct.
That struct ought to leverage the apporpriate struct tags for performing two tasks.
First, matching keys in the payload to fields on the struct.
Second, for validating the payload's data meets requirements.

By leveraging req, handlers can get data out of an HTTP request into its application specific structs.
Notably, the parade of errors that may propogate from such a task
are translated to trails sentinel errors in order to provide a consistent interface
for issues that arise across encoding types.
*/
package req
