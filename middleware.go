package fiber_sentinel_middleware

import (
	"net/http"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/gofiber/fiber/v2"
)

type (
	Option  func(*options)
	options struct {
		resourceExtract func(*fiber.Ctx) string
		blockFallback   func(*fiber.Ctx) error
	}
)

func evaluateOptions(opts []Option) *options {
	optCopy := &options{}
	for _, opt := range opts {
		opt(optCopy)
	}

	return optCopy
}

// WithResourceExtractor sets the resource extractor of the web requests.
func WithResourceExtractor(fn func(*fiber.Ctx) string) Option {
	return func(opts *options) {
		opts.resourceExtract = fn
	}
}

// WithBlockFallback sets the fallback handler when requests are blocked.
func WithBlockFallback(fn func(ctx *fiber.Ctx) error) Option {
	return func(opts *options) {
		opts.blockFallback = fn
	}
}

// SentinelMiddleware returns new gin.HandlerFunc
// Default resource name is {method}:{path}, such as "GET:/api/users/:id"
// Default block fallback is returning 429 code
// Define your own behavior by setting options
func SentinelMiddleware(opts ...Option) fiber.Handler {
	options := evaluateOptions(opts)
	return func(ctx *fiber.Ctx) error {
		resourceName := ctx.Route().Method + ":" + string(ctx.Context().Path())

		if options.resourceExtract != nil {
			resourceName = options.resourceExtract(ctx)
		}

		entry, entryErr := sentinel.Entry(
			resourceName,
			sentinel.WithResourceType(base.ResTypeWeb),
			sentinel.WithTrafficType(base.Inbound),
		)

		if entryErr != nil {
			if options.blockFallback != nil {
				return options.blockFallback(ctx)
			} else {
				return ctx.SendStatus(http.StatusTooManyRequests)
			}
		}

		defer entry.Exit()
		return ctx.Next()
	}
}
