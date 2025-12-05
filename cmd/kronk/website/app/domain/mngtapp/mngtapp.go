// Package mngtapp provides endpoints to handle server managment.
package mngtapp

import (
	"context"
	"net/http"

	"github.com/ardanlabs/kronk/cmd/kronk/website/foundation/logger"
	"github.com/ardanlabs/kronk/cmd/kronk/website/foundation/web"
)

type app struct {
	build string
	log   *logger.Logger
}

func newApp(log *logger.Logger) *app {
	return &app{
		log: log,
	}
}

func (a *app) libs(ctx context.Context, r *http.Request) web.Encoder {
	return nil
}

func (a *app) list(ctx context.Context, r *http.Request) web.Encoder {
	return nil
}

func (a *app) pull(ctx context.Context, r *http.Request) web.Encoder {
	return nil
}

func (a *app) remove(ctx context.Context, r *http.Request) web.Encoder {
	return nil
}

func (a *app) show(ctx context.Context, r *http.Request) web.Encoder {
	return nil
}
