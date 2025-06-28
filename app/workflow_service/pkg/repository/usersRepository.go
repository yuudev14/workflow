package repository

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/models"
)

type UserRepository interface {
	GetUserByID(id uuid.UUID) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	CreateUser(user *models.User) (*models.User, error)
	GetUserByEmailOrUsername(usernameOrEmail string) (*models.User, error)
}

type UserRepositoryImpl struct {
	*sqlx.DB
}

// New User Persistence constructor
func NewUserRepository(db *sqlx.DB) UserRepository {
	return &UserRepositoryImpl{
		DB: db,
	}
}

// Get User by providing username or email
func (u *UserRepositoryImpl) GetUserByEmailOrUsername(usernameOrEmail string) (*models.User, error) {
	return DbSelectOne[models.User](
		u.DB,
		"SELECT * from users WHERE email=$1 OR username=$1",
		usernameOrEmail,
	)
}

// Get User by providing id
func (u *UserRepositoryImpl) GetUserByID(id uuid.UUID) (*models.User, error) {
	return DbSelectOne[models.User](
		u.DB,
		"SELECT id, username, email, first_name, last_name from users WHERE id=$1",
		id,
	)
}

// Get User by providing username
func (u *UserRepositoryImpl) GetUserByUsername(username string) (*models.User, error) {
	return DbSelectOne[models.User](
		u.DB,
		"SELECT id, username, email, first_name, last_name from users WHERE username=$1",
		username,
	)
}

// Create User
func (u *UserRepositoryImpl) CreateUser(user *models.User) (*models.User, error) {
	_, err := u.DB.Exec(`INSERT INTO users (email, username, password) VALUES ($1, $2, $3)`, user.Email, user.Username, user.Password)
	if err != nil {
		return nil, err
	}
	newUser, err := u.GetUserByUsername(user.Username)
	if err != nil {
		return nil, err
	}
	return newUser, nil
}

// Get User by providing Email
func (u *UserRepositoryImpl) GetUserByEmail(email string) (*models.User, error) {
	return DbSelectOne[models.User](
		u.DB,
		"SELECT * from users WHERE email=$1",
		email,
	)
}
