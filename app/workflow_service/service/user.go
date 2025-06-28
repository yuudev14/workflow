package service

import (
	"fmt"

	"github.com/yuudev14-workflow/workflow-service/dto"
	"github.com/yuudev14-workflow/workflow-service/models"
	"github.com/yuudev14-workflow/workflow-service/pkg/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	GetUserByEmailOrUsername(usernameOrEmail string) (*models.User, error)
	ValidateUserSignUp(username string, email string) error
	VerifyUser(form dto.LoginForm) (*models.User, error)
	CheckUserByUsername(username string) error
	CheckUserByEmail(email string) error
	CreateUser(form dto.SignupForm) (*models.User, error)
}

type UserServiceImpl struct {
	UserRepository repository.UserRepository
}

// Auth Service Constructor
func NewUserService(UserRepository repository.UserRepository) UserService {
	return &UserServiceImpl{
		UserRepository: UserRepository,
	}
}

// VerifyUser implements UserService.
func (a *UserServiceImpl) VerifyUser(form dto.LoginForm) (*models.User, error) {
	user, usernameError := a.UserRepository.GetUserByEmailOrUsername(form.Username)

	if usernameError != nil {
		return nil, usernameError
	}

	if user == nil {
		return nil, fmt.Errorf("user is not found")
	}

	isNotMatch := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(form.Password))

	if isNotMatch != nil {
		return nil, fmt.Errorf("password is not correct")
	}
	return user, nil
}

// ValidateUserSignUp implements UserService.
func (a *UserServiceImpl) ValidateUserSignUp(username string, email string) error {
	// check if username already exist
	usernameError := a.CheckUserByUsername(username)

	if usernameError != nil {
		return usernameError
	}

	// check if email already exist
	emailError := a.CheckUserByEmail(email)

	if emailError != nil {
		return emailError
	}
	return nil
}

// CheckUserByEmail implements UserService.
func (a *UserServiceImpl) CheckUserByEmail(email string) error {
	// check if email already exist
	user, emailError := a.UserRepository.GetUserByEmail(email)

	if emailError != nil {
		return emailError
	}

	if user != nil {
		return fmt.Errorf("email already exist")
	}
	return nil
}

// CheckUserByUsername implements UserService.
func (a *UserServiceImpl) CheckUserByUsername(username string) error {
	user, usernameError := a.UserRepository.GetUserByUsername(username)

	if usernameError != nil {
		return usernameError
	}

	if user != nil {
		return fmt.Errorf("username already exist")
	}
	return nil
}

// Get User by providing username or email
func (a *UserServiceImpl) GetUserByEmailOrUsername(usernameOrEmail string) (*models.User, error) {
	return a.UserRepository.GetUserByEmailOrUsername(usernameOrEmail)
}

// create user
func (a *UserServiceImpl) CreateUser(form dto.SignupForm) (*models.User, error) {
	// encode password
	newPassword, passwordErr := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)

	if passwordErr != nil {
		return nil, passwordErr
	}
	// save user
	excryptedPassword := string(newPassword)
	newUser := models.User{
		Username: form.Username,
		Password: excryptedPassword,
		Email:    form.Email,
	}

	return a.UserRepository.CreateUser(&newUser)
}
