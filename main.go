package main

import (
	"bytes"
	"context"
	"encoding/json"
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

	// Create an instance of sentryhttp
	sentryHandler := sentryhttp.New(sentryhttp.Options{})

	mux := http.NewServeMux()
	mux.Handle("/usecase", sentryHandler.HandleFunc(usecaseHandler))
	mux.Handle("/message", sentryHandler.HandleFunc(messageHandler))

	l := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	server.New(l, mux, 8080)
}

func messageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	captureMessage(ctx, "This happened")
	fmt.Fprint(w, "ok")
}

func usecaseHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sentryScope(ctx, func(scope *sentry.Scope) {
		scope.SetTag("page.locale", "de-at")
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

		for k, v := range r.Header {
			scope.SetExtra(k, v[0])
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		bb := new(bytes.Buffer)
		if err := json.Indent(bb, b, "", " "); err != nil {
			panic(err)
		}

		if body := bb.Bytes(); len(body) != 0 {
			scope.SetExtra("body-str", string(body))
			scope.SetExtra("body-byt", body)
		}

		scope.AddBreadcrumb(&sentry.Breadcrumb{
			Type:     "Controller",
			Category: "controller",
			Message:  "Enter controller",
			Data: map[string]interface{}{
				"controllerName": "SomeController",
			},
			Level:     sentry.LevelInfo,
			Timestamp: time.Now(),
		}, 10)
	})

	if err := usecase(ctx, usecaseDto{
		Name: "john",
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		captureException(ctx, err)
		return
	}

	fmt.Fprint(w, "hello world")
}

type usecaseDto struct {
	Name string
}

func usecase(ctx context.Context, dto usecaseDto) error {
	span := sentry.StartSpan(ctx, "operation")
	defer span.Finish()

	sentryScope(ctx, func(scope *sentry.Scope) {
		scope.AddBreadcrumb(&sentry.Breadcrumb{
			Type:     "Usecase",
			Category: "usecase",
			Message:  "Entering usecase",
			Level:    sentry.LevelInfo,
			Data: map[string]interface{}{
				"req": dto,
			},
			Timestamp: time.Now(),
		}, 10)
		scope.SetExtra("what", dto) // This is recommended
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
	return errcodes.New(errcodes.BadRequest, "bar_bad_request", "Don't user bar please")
}

func captureMessage(ctx context.Context, msg string) {
	if !sentry.HasHubOnContext(ctx) || msg == "" {
		return
	}

	sentry.GetHubFromContext(ctx).CaptureMessage(msg)
}

func captureException(ctx context.Context, err error) {
	if !sentry.HasHubOnContext(ctx) || err == nil {
		return
	}

	sentry.GetHubFromContext(ctx).CaptureException(err)
}

func sentryScope(ctx context.Context, fn func(scope *sentry.Scope)) {
	if !sentry.HasHubOnContext(ctx) {
		return
	}

	fn(sentry.GetHubFromContext(ctx).Scope())
}
