# go-glitchtip


Glitchtip is an open source error monitoring system, similar to Sentry.


## Context

### Problem

Testing the Sentry integration with the actual Sentry is hard.


### Solution

Setup a local Glitchtip to test the Sentry SDK locally.


## Steps

```bash
# Start up the containers.
$ make up

# Open the Web UI.
$ make open
```

1. Run the instructions above to setup the local infra.
2. Sign up a new account. Disregard the 500 internal server error, because Glitchtip will try to send an email for confirmation, but it is disabled.
3. Sign in using the email and password.
4. Create a new organization `Test Sentry`.
5. Create a new project with `Server` platform and language `Go`. Use the name `Test Sentry`.
6. Create a team `Test-Sentry`.
7. Once the project is created, copy the `DSN` and paste it into the `.env` file `SENTRY_DSN=<your_dsn>`
8. Run `make run` to run the server.
9. Run `make trigger` to execute the endpoints that sends the logs to Sentry.
10. Refresh the page to see the new logs.

### Thoughts


Span can panic if nil.

```go
	span := sentry.TransactionFromContext(context.Background())
	if span == nil {
		span = new(sentry.Span)
	}
	defer span.Finish()
```
