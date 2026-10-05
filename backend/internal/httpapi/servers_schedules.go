package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"app/api/v1alpha1"
	"app/internal/schedule"
)

// ScheduleView is a schedule with its next and last run.
type ScheduleView struct {
	v1alpha1.Schedule
	NextRunAt  *time.Time `json:"nextRunAt,omitempty"`
	LastRunAt  *time.Time `json:"lastRunAt,omitempty"`
	LastResult string     `json:"lastResult,omitempty"`
	Running    bool       `json:"running"`
}

// ScheduleList lists the schedules of a server.
type ScheduleList struct {
	// TimeZone the cron expressions are evaluated in.
	TimeZone string         `json:"timeZone" example:"Europe/Berlin"`
	Items    []ScheduleView `json:"items"`
}

// UpdateSchedulesRequest replaces all schedules of a server.
type UpdateSchedulesRequest struct {
	Items []v1alpha1.Schedule `json:"items"`
}

func (a *API) scheduleList(gs *v1alpha1.GameServer) ScheduleList {
	now := time.Now().In(a.Schedules.Location)
	out := ScheduleList{TimeZone: a.Schedules.Location.String(), Items: []ScheduleView{}}
	for _, s := range gs.Spec.Schedules {
		v := ScheduleView{Schedule: s, Running: a.Schedules.Running(gs, s.Name)}
		if next := schedule.Next(s.Cron, now); s.Enabled && !next.IsZero() {
			v.NextRunAt = &next
		}
		for _, st := range gs.Status.Schedules {
			if st.Name == s.Name && st.LastRunAt != nil {
				t := st.LastRunAt.Time
				v.LastRunAt, v.LastResult = &t, st.LastResult
			}
		}
		out.Items = append(out.Items, v)
	}
	return out
}

// listSchedules godoc
//
//	@Summary	Schedules of a server
//	@Tags		Schedules
//	@Produce	json
//	@Param		server	path		string	true	"Server name"
//	@Success	200		{object}	ScheduleList
//	@Security	BearerAuth
//	@Router		/servers/{server}/schedules [get]
func (a *API) listSchedules(c *gin.Context) {
	if gs, ok := a.loadServer(c); ok {
		c.JSON(http.StatusOK, a.scheduleList(gs))
	}
}

// updateSchedules godoc
//
//	@Summary		Replace the schedules of a server
//	@Description	Cron expressions (5 fields or @daily, @hourly, …) use the panel time zone. Tasks run one after
//	@Description	another; the delay waits before a task.
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			server	path		string					true	"Server name"
//	@Param			body	body		UpdateSchedulesRequest	true	"Schedules"
//	@Success		200		{object}	ScheduleList
//	@Failure		422		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/servers/{server}/schedules [put]
func (a *API) updateSchedules(c *gin.Context) {
	var req UpdateSchedulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		a.fail(c, badRequest(err))
		return
	}
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	list, err := schedule.Validate(req.Items)
	if err != nil {
		a.fail(c, err)
		return
	}
	patch := client.MergeFrom(gs.DeepCopy())
	gs.Spec.Schedules = list
	if err := a.Client.Patch(c, gs, patch); err != nil {
		a.fail(c, err)
		return
	}
	a.audit(c, "schedules updated", "server", gs.Name, "count", len(list))
	c.JSON(http.StatusOK, a.scheduleList(gs))
}

// runSchedule godoc
//
//	@Summary		Run a schedule now
//	@Description	Starts the tasks in the background (also for disabled schedules).
//	@Tags			Schedules
//	@Param			server		path	string	true	"Server name"
//	@Param			schedule	path	string	true	"Schedule name"
//	@Success		202
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse	"already running"
//	@Security		BearerAuth
//	@Router			/servers/{server}/schedules/{schedule}/run [post]
func (a *API) runSchedule(c *gin.Context) {
	gs, ok := a.loadServer(c)
	if !ok {
		return
	}
	if err := a.Schedules.RunNow(c, gs, c.Param("schedule")); err != nil {
		a.fail(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}
