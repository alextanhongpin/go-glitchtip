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
	"github.com/alextanhongpin/errors/causes"
	"github.com/alextanhongpin/errors/codes"
	"github.com/alextanhongpin/errors/stacktrace"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"go.jetpack.io/typeid"
)

func init() {
	stacktrace.SetMaxDepth(8)
}

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
	mux.Handle("/error", sentryHandler.HandleFunc(errorHandler))

	server.ListenAndServe(":8080", mux)
}

func messageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureMessage("This happened")
	}

	fmt.Fprint(w, "ok")
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := sentry.TransactionFromContext(ctx)
	defer span.Finish()
	span.SetData("span-data-key", "span-data-val")
	span.SetTag("span-tag-key", "span-tag-val")

	hub := sentry.GetHubFromContext(ctx)
	if hub != nil {
		scope := hub.Scope()
		tid := typeid.Must(typeid.New("user"))

		// https://develop.sentry.dev/sdk/event-payloads/breadcrumbs/#breadcrumb-types
		scope.AddBreadcrumb(&sentry.Breadcrumb{
			Type:     "debug",
			Category: "error.handle",
			Data: map[string]any{
				"hello": struct {
					ID   string
					Name string
				}{
					ID:   tid.String(),
					Name: "john",
				},
			},
			Level:     sentry.LevelDebug,
			Message:   "some message",
			Timestamp: time.Now(),
		}, 100)

		scope.SetExtra("key", "value")
		scope.SetExtra("foo", struct {
			Name string
		}{
			Name: "john",
		})
	}

	err := six(ctx)
	if err != nil {
		hub.CaptureException(err)
	}

	fmt.Fprint(w, http.StatusText(http.StatusInternalServerError))
}

type stackTrace struct {
	msg   string
	stack []uintptr
}

func (s *stackTrace) Error() string {
	return s.msg
}

func (s *stackTrace) StackTrace() []uintptr {
	return s.stack
}

func one() error {
	return stacktrace.New("one")
}

func two(ctx context.Context) error {
	err := stacktrace.Annotate(one(), "two")
	if err != nil {
		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			scope := hub.Scope()
			scope.SetTag("two-key", "two-val")

			scope.AddBreadcrumb(&sentry.Breadcrumb{
				Type:     "debug",
				Category: "two.handle",
				Data: map[string]any{
					"req": 1,
				},
				Level:     sentry.LevelDebug,
				Message:   "called two failed",
				Timestamp: time.Now(),
			}, 100)
		}

		return err
	}

	return nil
}

func three(ctx context.Context) error {
	return stacktrace.Annotate(two(ctx), "three")
}

func four(ctx context.Context) error {
	return stacktrace.Annotate(three(ctx), "four")
}

func five(ctx context.Context) error {
	return stacktrace.Annotate(four(ctx), "five")
}

func six(ctx context.Context) error {
	return stacktrace.Annotate(five(ctx), "six")
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
		return stacktrace.Annotate(fmt.Errorf("foo: %w", err), "foo")
	}

	return nil
}

func bar() error {
	return causes.New(codes.BadRequest, "bar_bad_request", "Don't user bar please")
}

func captureMessage(ctx context.Context, msg string) {
	if !sentry.HasHubOnContext(ctx) || msg == "" {
		return
	}

	sentry.GetHubFromContext(ctx).CaptureMessage(msg)
}

// Don't do this, cause the stacktrace will be added at this point, which is redundant.
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
