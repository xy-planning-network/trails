/*
Package auth plays with moving some auth patterns into trails.

JWT

Consumption of a JWT token happens in both wall_to_wall and second_child. This moves that code into a shared location
and updates the concept of an application token to live in a central location.

Google

We are receiving an oauth token from retool and fetching the user. This is a bit more secure than a bearer
authentication header which is where we started with API calls from retool. This is used in second_child, college_try,
and wall_to_wall.

Future

In the future, if we have the appetite, all of our logic for auth could live in trails, making it even easier to spin up
an app and have preconfigured handlers that manage our various auth approaches.
*/

package auth
