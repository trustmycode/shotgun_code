package main

import (
	"context"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func hasRuntimeLogger(ctx context.Context) bool {
	return ctx != nil && ctx.Value("logger") != nil
}

func hasRuntimeEvents(ctx context.Context) bool {
	return ctx != nil && ctx.Value("events") != nil
}

func safeLogDebug(ctx context.Context, message string) {
	if hasRuntimeLogger(ctx) {
		wailsRuntime.LogDebug(ctx, message)
	}
}

func safeLogDebugf(ctx context.Context, format string, args ...interface{}) {
	if hasRuntimeLogger(ctx) {
		wailsRuntime.LogDebugf(ctx, format, args...)
	}
}

func safeLogInfo(ctx context.Context, message string) {
	if hasRuntimeLogger(ctx) {
		wailsRuntime.LogInfo(ctx, message)
	}
}

func safeLogInfof(ctx context.Context, format string, args ...interface{}) {
	if hasRuntimeLogger(ctx) {
		wailsRuntime.LogInfof(ctx, format, args...)
	}
}

func safeLogWarning(ctx context.Context, message string) {
	if hasRuntimeLogger(ctx) {
		wailsRuntime.LogWarning(ctx, message)
	}
}

func safeLogWarningf(ctx context.Context, format string, args ...interface{}) {
	if hasRuntimeLogger(ctx) {
		wailsRuntime.LogWarningf(ctx, format, args...)
	}
}

func safeLogError(ctx context.Context, message string) {
	if hasRuntimeLogger(ctx) {
		wailsRuntime.LogError(ctx, message)
	}
}

func safeLogErrorf(ctx context.Context, format string, args ...interface{}) {
	if hasRuntimeLogger(ctx) {
		wailsRuntime.LogErrorf(ctx, format, args...)
	}
}

func safeEventsEmit(ctx context.Context, eventName string, optionalData ...interface{}) {
	if hasRuntimeEvents(ctx) {
		wailsRuntime.EventsEmit(ctx, eventName, optionalData...)
	}
}
