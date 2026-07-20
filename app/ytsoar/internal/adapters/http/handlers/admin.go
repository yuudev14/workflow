package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/yuudev14/ytsoar/internal/adapters/http/common/dto"
	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	rest "github.com/yuudev14/ytsoar/internal/adapters/http/rests"
	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// AdminHandler serves the settings module: users, roles, teams and the audit
// trail. They share one handler because they share one permission module and
// one service.
type AdminHandler struct {
	logger      logger.Logger
	authService *auth.Service
}

func NewAdminHandler(log logger.Logger, authService *auth.Service) *AdminHandler {
	return &AdminHandler{logger: log, authService: authService}
}

// actorAndTarget resolves who is acting and which entity they named. Every
// mutating route needs both, and the id must be validated here — gin's binder
// cannot bind a path param to uuid.UUID.
func (h *AdminHandler) actorAndTarget(c *gin.Context, param string) (uuid.UUID, uuid.UUID, bool) {
	response := rest.Response{C: c}

	actor, ok := middleware.CurrentUser(c)
	if !ok {
		response.ResponseError(http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, uuid.Nil, false
	}

	target, err := uuid.Parse(c.Param(param))
	if err != nil {
		response.ResponseError(http.StatusBadRequest, param+" must be a uuid")
		return uuid.Nil, uuid.Nil, false
	}
	return actor.ID, target, true
}

func (h *AdminHandler) actor(c *gin.Context) (uuid.UUID, bool) {
	actor, ok := middleware.CurrentUser(c)
	if !ok {
		response := rest.Response{C: c}
		response.ResponseError(http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, false
	}
	return actor.ID, true
}



func (h *AdminHandler) ListUsers(c *gin.Context) {
	response := rest.Response{C: c}

	var query dto.FilterQuery
	var filter auth.UserFilter

	if ok, code, err := rest.BindQueryAndValidate(c, &query); !ok {
		response.ResponseError(code, err)
		return
	}
	if ok, code, err := rest.BindQueryAndValidate(c, &filter); !ok {
		response.ResponseError(code, err)
		return
	}

	users, err := h.authService.ListUsers(c.Request.Context(), query.Offset, query.Limit, filter)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(users)
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	response := rest.Response{C: c}

	id, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.ResponseError(http.StatusBadRequest, "user_id must be a uuid")
		return
	}

	user, err := h.authService.GetUser(c.Request.Context(), id)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(user)
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, ok := h.actor(c)
	if !ok {
		return
	}

	var body auth.CreateUserInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.authService.CreateUser(c.Request.Context(), actorID, body)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.IndentedJSON(http.StatusCreated, user)
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, userID, ok := h.actorAndTarget(c, "user_id")
	if !ok {
		return
	}

	var body auth.UpdateUserInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.authService.UpdateUser(c.Request.Context(), actorID, userID, body)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(user)
}

func (h *AdminHandler) SetUserRoles(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, userID, ok := h.actorAndTarget(c, "user_id")
	if !ok {
		return
	}

	var body auth.SetUserRolesInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.authService.SetUserRoles(c.Request.Context(), actorID, userID, body.RoleIDs)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(user)
}

func (h *AdminHandler) SetUserPassword(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, userID, ok := h.actorAndTarget(c, "user_id")
	if !ok {
		return
	}

	var body auth.SetPasswordInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	if err := h.authService.SetUserPassword(c.Request.Context(), actorID, userID, body.Password); err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// DeactivateUser is what DELETE means here. Users are never hard-deleted:
// audit rows and uploaded connectors reference them.
func (h *AdminHandler) DeactivateUser(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, userID, ok := h.actorAndTarget(c, "user_id")
	if !ok {
		return
	}

	// Locking yourself out is easy to do by accident and awkward to undo.
	if actorID == userID {
		response.ResponseError(http.StatusBadRequest, "you cannot deactivate your own account")
		return
	}

	if err := h.authService.DeactivateUser(c.Request.Context(), actorID, userID); err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- roles ----

func (h *AdminHandler) ListRoles(c *gin.Context) {
	response := rest.Response{C: c}

	roles, err := h.authService.ListRoles(c.Request.Context())
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(roles)
}

func (h *AdminHandler) GetRole(c *gin.Context) {
	response := rest.Response{C: c}

	id, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		response.ResponseError(http.StatusBadRequest, "role_id must be a uuid")
		return
	}

	role, err := h.authService.GetRole(c.Request.Context(), id)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(role)
}

func (h *AdminHandler) CreateRole(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, ok := h.actor(c)
	if !ok {
		return
	}

	var body auth.RoleInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	role, err := h.authService.CreateRole(c.Request.Context(), actorID, body)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.IndentedJSON(http.StatusCreated, role)
}

func (h *AdminHandler) UpdateRole(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, roleID, ok := h.actorAndTarget(c, "role_id")
	if !ok {
		return
	}

	var body auth.UpdateRoleInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	role, err := h.authService.UpdateRole(c.Request.Context(), actorID, roleID, body)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(role)
}

func (h *AdminHandler) SetRolePermissions(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, roleID, ok := h.actorAndTarget(c, "role_id")
	if !ok {
		return
	}

	var body struct {
		Permissions map[string][]string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	role, err := h.authService.SetRolePermissions(c.Request.Context(), actorID, roleID, body.Permissions)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(role)
}

func (h *AdminHandler) DeleteRole(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, roleID, ok := h.actorAndTarget(c, "role_id")
	if !ok {
		return
	}

	if err := h.authService.DeleteRole(c.Request.Context(), actorID, roleID); err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- teams ----

func (h *AdminHandler) ListTeams(c *gin.Context) {
	response := rest.Response{C: c}

	var query dto.FilterQuery
	var filter auth.TeamFilter

	if ok, code, err := rest.BindQueryAndValidate(c, &query); !ok {
		response.ResponseError(code, err)
		return
	}
	if ok, code, err := rest.BindQueryAndValidate(c, &filter); !ok {
		response.ResponseError(code, err)
		return
	}

	teams, err := h.authService.ListTeams(c.Request.Context(), query.Offset, query.Limit, filter)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(teams)
}

func (h *AdminHandler) GetTeam(c *gin.Context) {
	response := rest.Response{C: c}

	id, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		response.ResponseError(http.StatusBadRequest, "team_id must be a uuid")
		return
	}

	team, err := h.authService.GetTeam(c.Request.Context(), id)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(team)
}

func (h *AdminHandler) CreateTeam(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, ok := h.actor(c)
	if !ok {
		return
	}

	var body auth.TeamInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	team, err := h.authService.CreateTeam(c.Request.Context(), actorID, body)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.IndentedJSON(http.StatusCreated, team)
}

func (h *AdminHandler) UpdateTeam(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, teamID, ok := h.actorAndTarget(c, "team_id")
	if !ok {
		return
	}

	var body auth.UpdateTeamInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	team, err := h.authService.UpdateTeam(c.Request.Context(), actorID, teamID, body)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(team)
}

func (h *AdminHandler) SetTeamMembers(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, teamID, ok := h.actorAndTarget(c, "team_id")
	if !ok {
		return
	}

	var body auth.SetTeamMembersInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	team, err := h.authService.SetTeamMembers(c.Request.Context(), actorID, teamID, body.MemberIDs)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(team)
}

func (h *AdminHandler) DeleteTeam(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, teamID, ok := h.actorAndTarget(c, "team_id")
	if !ok {
		return
	}

	if err := h.authService.DeleteTeam(c.Request.Context(), actorID, teamID); err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- audit ----

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	response := rest.Response{C: c}

	var query dto.FilterQuery
	var filter auth.AuditFilter

	if ok, code, err := rest.BindQueryAndValidate(c, &query); !ok {
		response.ResponseError(code, err)
		return
	}
	if ok, code, err := rest.BindQueryAndValidate(c, &filter); !ok {
		response.ResponseError(code, err)
		return
	}

	logs, err := h.authService.ListAuditLogs(c.Request.Context(), query.Offset, query.Limit, filter)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(logs)
}