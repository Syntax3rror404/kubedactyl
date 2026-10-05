package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"app/internal/files"
)

// FilesSession describes the file container of a server. It only runs while the file
// manager is used and is removed after IdleTimeoutSeconds without file operations.
type FilesSession struct {
	Ready bool `json:"ready"`
	// State is Stopped, Starting, Ready or Stopping.
	State string `json:"state" enums:"Stopped,Starting,Ready,Stopping" example:"Starting"`
	// Message explains a Starting state (e.g. "ContainerCreating").
	Message            string `json:"message,omitempty"`
	IdleTimeoutSeconds int    `json:"idleTimeoutSeconds" example:"60"`
	// StopsInSeconds is how long a ready container stays without further file operations
	// (reading the session does not count as one).
	StopsInSeconds int    `json:"stopsInSeconds,omitempty" example:"42"`
	Error          string `json:"error,omitempty"`
}

// How long a file operation waits for the file container before it answers 503.
const filesStartWait = 45 * time.Second

// getFilesSession godoc
//
//	@Summary		State of the file container
//	@Description	Does not start the container and does not count as activity.
//	@Tags			Files
//	@Produce		json
//	@Param			server	path		string	true	"Server name"
//	@Success		200		{object}	FilesSession
//	@Security		BearerAuth
//	@Router			/servers/{server}/files/session [get]
func (a *API) getFilesSession(c *gin.Context) {
	if gs, ok := a.loadServer(c); ok {
		c.JSON(http.StatusOK, sessionOf(a.Files.PodState(c, files.RefOf(gs))))
	}
}

// openFilesSession godoc
//
//	@Summary		Start the file container
//	@Description	Starts the file container of the server if needed and returns its state. Poll GET until ready.
//	@Tags			Files
//	@Produce		json
//	@Param			server	path		string	true	"Server name"
//	@Success		200		{object}	FilesSession
//	@Security		BearerAuth
//	@Router			/servers/{server}/files/session [post]
func (a *API) openFilesSession(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	a.Files.Activity.Touch(files.RefOf(gs))
	st := a.Files.PodState(c, files.RefOf(gs))
	if !st.Ready {
		a.Trigger(gs.Namespace, gs.Name)
	}
	c.JSON(http.StatusOK, sessionOf(st))
}

// requireFilesPod runs before every file operation: it counts as activity and waits
// (up to filesStartWait) until the file container is ready.
func (a *API) requireFilesPod(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c, filesStartWait)
	defer cancel()
	err := a.Files.EnsurePod(ctx, files.RefOf(gs))
	if errors.Is(err, files.ErrForeignPod) {
		a.fail(c, err)
		return
	}
	if err != nil {
		st := sessionOf(a.Files.PodState(c, files.RefOf(gs)))
		st.Error = err.Error()
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, st)
		return
	}
	c.Set(serverKey, gs)
	c.Next()
}

func sessionOf(st files.PodState) FilesSession {
	return FilesSession{
		Ready: st.Ready, State: st.State, Message: st.Message, IdleTimeoutSeconds: int(files.IdleTimeout.Seconds()),
		StopsInSeconds: int(st.StopsIn.Round(time.Second).Seconds()),
	}
}
