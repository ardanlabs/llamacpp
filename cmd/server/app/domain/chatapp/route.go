package chatapp

import (
	"net/http"

	"github.com/ardanlabs/kronk/cmd/server/app/sdk/auth"
	"github.com/ardanlabs/kronk/cmd/server/app/sdk/mid"
	"github.com/ardanlabs/kronk/cmd/server/foundation/logger"
	"github.com/ardanlabs/kronk/cmd/server/foundation/web"
	"github.com/ardanlabs/kronk/sdk/kronk/cache"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Log   *logger.Logger
	Auth  *auth.Auth
	Cache *cache.Cache
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	bearer := mid.Bearer(cfg.Auth)

	api := newApp(cfg.Log, cfg.Cache)

	app.HandlerFunc(http.MethodPost, version, "/chat/completions", api.chatCompletions, bearer)
}
