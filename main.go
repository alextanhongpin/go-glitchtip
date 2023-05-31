package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alextanhongpin/core/http/server"
	"github.com/alextanhongpin/errcodes"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"golang.org/x/exp/slog"
)

type BadRequestError struct {
}

func (e *BadRequestError) Error() string {
	return "bad request"
}

var ErrBadRequest = errors.New("bad request")

func main() {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              os.Getenv("SENTRY_DSN"),
		Environment:      "production",
		AttachStacktrace: true,
		EnableTracing:    true,
		// Enable printing of SDK debug messages.
		// Useful when getting started or trying to figure something out.
		Debug: true,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	defer sentry.Recover()
	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentry.Flush(2 * time.Second)

	sentry.CaptureException(usecase(context.Background()))
	sentry.CaptureMessage("It works!")

	// Create an instance of sentryhttp
	sentryHandler := sentryhttp.New(sentryhttp.Options{})

	mux := http.NewServeMux()
	mux.Handle("/", sentryHandler.HandleFunc(handler))
	l := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	server.New(l, mux, 8080)
}

func handler(w http.ResponseWriter, r *http.Request) {
	if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			scope.SetExtra("unwantedQuery", "someQueryDataMaybe")
			b, err := io.ReadAll(r.Body)
			if err != nil {
				hub.CaptureException(err)
			} else {
				scope.SetExtra("body", string(b))
				hub.CaptureMessage("User provided unwanted query string, but we recovered just fine")
			}
		})
	}
	fmt.Fprint(w, "hello world")
}

func usecase(ctx context.Context) error {
	span := sentry.StartSpan(ctx, "operation")
	defer span.Finish()

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID:        "usser-1234",
			Email:     "john.doe@mail.com",
			IPAddress: "0.0.0.0",
			Username:  "john doe",
			Name:      "John",
			Segment:   "user-segment-a",
			Data: map[string]string{
				"Age":      "12",
				"Verified": "true",
			},
		})
		scope.SetTag("page.locale", "de-at")
		scope.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "foo",
			Message:  "foo",
			Level:    sentry.LevelInfo,
		}, 10)
		scope.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "bar",
			Message:  "bar",
			Level:    sentry.LevelInfo,
		}, 10)
		scope.SetExtra("what", "is extra")
		scope.SetExtra("json", `{"name": "john"}`)
	})

	return foo()
}

func foo() error {
	err := bar()
	if err != nil {
		return fmt.Errorf("foo: %w", err)
	}
	return nil
}

func bar() error {
	//return ErrBadRequest
	//return &BadRequestError{}
	return errcodes.New(errcodes.BadRequest, "bar_bad_request", "Don't user bar please")
}
