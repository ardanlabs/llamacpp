package toolapp

import (
	"net/http"

	"github.com/ardanlabs/kronk/cmd/server/app/sdk/auth"
	"github.com/ardanlabs/kronk/cmd/server/app/sdk/mid"
	"github.com/ardanlabs/kronk/cmd/server/foundation/logger"
	"github.com/ardanlabs/kronk/cmd/server/foundation/web"
	"github.com/ardanlabs/kronk/sdk/cache"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Log   *logger.Logger
	Auth  *auth.Auth
	Cache *cache.Cache
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = ""

	bearer := mid.Bearer(cfg.Auth)

	api := newApp(cfg.Log, cfg.Cache)

	app.HandlerFunc(http.MethodPost, version, "/v1/libs/pull", api.pullLibs, bearer)
	app.HandlerFunc(http.MethodGet, version, "/v1/models", api.listModels, bearer)
	app.HandlerFunc(http.MethodGet, version, "/v1/models/{model}", api.showModel, bearer)
	app.HandlerFunc(http.MethodGet, version, "/v1/models/status", api.modelStatus, bearer)
	app.HandlerFunc(http.MethodPost, version, "/v1/models/pull", api.pullModels, bearer)
	app.HandlerFunc(http.MethodDelete, version, "/v1/models/{model}", api.removeModel, bearer)
}
