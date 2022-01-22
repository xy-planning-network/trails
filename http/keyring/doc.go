/*

Package keyring defines how keys in a *http.Request.Context should behave
and a way for storing and retrieving those keys for wider use in the application.

The main method for managing keys is through a Keyring, or a custom implementation of Keyringable.

Following https://medium.com/@matryer/context-keys-in-go-5312346a868d,
context keys ought to be unexported by a package.
This package cannot, in other words, provide default keys to be used in a context.Context.

*/
package keyring
