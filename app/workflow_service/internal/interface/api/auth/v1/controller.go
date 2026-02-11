package auth_api_v1

import (
	"fmt"
	"net/http"
	"time"

	rest "github.com/yuudev14-workflow/workflow-service/internal/rests"
	"github.com/yuudev14-workflow/workflow-service/internal/token"

	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/yuudev14-workflow/workflow-service/dto"
	"github.com/yuudev14-workflow/workflow-service/internal/logging"
	"github.com/yuudev14-workflow/workflow-service/service"
)

type AuthController struct {
	UserService service.UserService
}

func NewAuthController(UserService service.UserService) *AuthController {
	return &AuthController{
		UserService: UserService,
	}
}

// Signup godoc
// @Summary signup
// @Description api for signing up user
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} dto.AuthResponse "response object"
// @Param request body dto.SignupForm true "Request Body"
// @Router /api/auth/sign-up [post]
func (a *AuthController) SignUp(c *gin.Context) {
	response := rest.Response{C: c}

	var form dto.SignupForm
	logging.Sugar.Debug("validating form...")
	check, code, validErr := rest.BindFormAndValidate(c, &form)

	if !check {
		logging.Sugar.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}

	valid := validation.Validation{}

	valid.Email(form.Email, "email")
	check, code, err := rest.ValidateData(valid, form)
	if !check || err != nil {
		logging.Sugar.Errorf(fmt.Sprintf("%v", err))
		response.ResponseError(code, err)
		return
	}

	validateErr := a.UserService.ValidateUserSignUp(form.Username, form.Email)

	if validateErr != nil {
		logging.Sugar.Errorf(validateErr.Error())
		response.ResponseError(http.StatusBadRequest, validateErr.Error())
		return
	}

	addedUser, addUserErr := a.UserService.CreateUser(form)
	logging.Sugar.Debug("added user...")

	if addUserErr != nil {
		response.ResponseError(http.StatusBadRequest, addUserErr.Error())
		return
	}

	// generate token
	logging.Sugar.Debug("generating token...")
	accessToken, refreshToken, tokenErr := token.GeneratePairToken(jwt.MapClaims{
		"sub": addedUser.ID.String(),
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}, time.Now().Add(time.Hour*24*30).Unix())

	if tokenErr != nil {
		response.ResponseError(http.StatusInternalServerError, tokenErr.Error())
		return
	}

	response.ResponseSuccess(dto.AuthResponse{
		AccessToken:  *accessToken,
		RefreshToken: *refreshToken,
		User: dto.User{
			ID:        addedUser.ID,
			Username:  addedUser.Username,
			Email:     addedUser.Email,
			FirstName: addedUser.FirstName,
			LastName:  addedUser.LastName,
			CreatedAt: addedUser.CreatedAt,
		},
	})
}

// Login godoc
// @Summary login
// @Description api for logging in
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} dto.AuthResponse "response object"
// @Param request body dto.LoginForm true "Request Body"
// @Router /api/auth/login [post]
func (a *AuthController) Login(c *gin.Context) {
	response := rest.Response{C: c}

	var form dto.LoginForm
	logging.Sugar.Debug("validating form...")

	check, code, validErr := rest.BindFormAndValidate(c, &form)
	if !check {
		logging.Sugar.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}
	logging.Sugar.Debugf("form validated... %v", form)

	// check if username already exist
	user, usernameError := a.UserService.VerifyUser(form)
	if usernameError != nil {
		logging.Sugar.Errorf(usernameError.Error())
		response.ResponseError(http.StatusBadRequest, usernameError.Error())
		return
	}
	// generate token
	logging.Sugar.Debug("generating token...")
	accessToken, refreshToken, tokenErr := token.GeneratePairToken(jwt.MapClaims{
		"sub": user.ID.String(),
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}, time.Now().Add(time.Hour*24*30).Unix())

	if tokenErr != nil {
		logging.Sugar.Errorf(tokenErr.Error())
		response.ResponseError(http.StatusInternalServerError, tokenErr.Error())
		return
	}

	response.ResponseSuccess(dto.AuthResponse{
		AccessToken:  *accessToken,
		RefreshToken: *refreshToken,
		User: dto.User{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			CreatedAt: user.CreatedAt,
		},
	})
}
