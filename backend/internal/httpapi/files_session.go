package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"app/base/api"
	"app/internal/files"
)

// FilesSession describes the file container of a server. It only runs while the file
// manager is used and is removed after IdleTimeoutSeconds without file operations.
type FilesSession struct {
	Ready bool `json:"ready"`
	// State is Stopped, Starting, Ready or Stopping.
	State string `json:"state" enums:"Stopped,Starting,Ready,Stopping" example:"Starting"`
	// Message explains a Starting state (e.g. "ContainerCreating").
	Message string `json:"message,omitempty"`
	// IdleTimeoutSeconds is how long the container stays after the last file operation (seconds).
	IdleTimeoutSeconds int `json:"idleTimeoutSeconds" example:"60"`
	// StopsInSeconds is how long a ready container stays without further file operations
	// (reading the session does not count as one).
	StopsInSeconds int `json:"stopsInSeconds,omitempty" example:"42"`
	// StartingSeconds is how long a starting container exists (seconds).
	StartingSeconds int    `json:"startingSeconds,omitempty" example:"12"`
	Error           string `json:"error,omitempty"`
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
func (a *API) getFilesSession(w http.ResponseWriter, r *http.Request) {
	if gs, ok := a.loadServer(w, r); ok {
		api.JSON(w, http.StatusOK, sessionOf(a.Files.PodState(r.Context(), files.RefOf(gs))))
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
func (a *API) openFilesSession(w http.ResponseWriter, r *http.Request) {
	gs, ok := a.loadServer(w, r)
	if !ok {
		return
	}
	a.Files.Activity.Touch(files.RefOf(gs))
	st := a.Files.PodState(r.Context(), files.RefOf(gs))
	if !st.Ready {
		a.Trigger(gs.Namespace, gs.Name)
	}
	api.JSON(w, http.StatusOK, sessionOf(st))
}

// requireFilesPod runs before every file operation: it counts as activity and waits
// (up to filesStartWait) until the file container is ready.
func (a *API) requireFilesPod(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gs, ok := a.loadServer(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), filesStartWait)
		defer cancel()
		err := a.Files.EnsurePod(ctx, files.RefOf(gs))
		if errors.Is(err, files.ErrForeignPod) {
			a.Fail(w, r, err)
			return
		}
		if err != nil {
			st := sessionOf(a.Files.PodState(r.Context(), files.RefOf(gs)))
			st.Error = err.Error()
			api.JSON(w, http.StatusServiceUnavailable, st)
			return
		}
		next.ServeHTTP(w, withServer(r, gs))
	})
}

func sessionOf(st files.PodState) FilesSession {
	return FilesSession{
		Ready: st.Ready, State: st.State, Message: st.Message, IdleTimeoutSeconds: int(files.IdleTimeout.Seconds()),
		StopsInSeconds:  int(st.StopsIn.Round(time.Second).Seconds()),
		StartingSeconds: int(st.StartingFor.Round(time.Second).Seconds()),
	}
}
