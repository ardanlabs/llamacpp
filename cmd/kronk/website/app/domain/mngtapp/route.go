package mngtapp

import (
	"net/http"

	"github.com/ardanlabs/kronk/cmd/kronk/website/app/sdk/auth"
	"github.com/ardanlabs/kronk/cmd/kronk/website/app/sdk/mid"
	"github.com/ardanlabs/kronk/cmd/kronk/website/foundation/logger"
	"github.com/ardanlabs/kronk/cmd/kronk/website/foundation/web"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Log  *logger.Logger
	Auth *auth.Auth
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = ""

	bearer := mid.Bearer(cfg.Auth)

	api := newApp(cfg.Log)

	app.HandlerFunc(http.MethodGet, version, "/mngt/libs", api.libs, bearer)
	app.HandlerFunc(http.MethodGet, version, "/mngt/list", api.list, bearer)
	app.HandlerFunc(http.MethodGet, version, "/mngt/pull", api.pull, bearer)
	app.HandlerFunc(http.MethodGet, version, "/mngt/remove", api.remove, bearer)
	app.HandlerFunc(http.MethodGet, version, "/mngt/show", api.show, bearer)
}
