/*
Package postgres manages our database connection. As part of the connection process, we also ensure that all migrations
have been run on the proper database. The situation where the database is simply a target for some testing has been
considered as well. In this scenario, we are dropping the public schema.

There is a very basic set of getter methods that have been implemented as well. An interface has been provided such that
it can be mocked out for testing that does not need an actual database running in the environment.
*/
package postgres
