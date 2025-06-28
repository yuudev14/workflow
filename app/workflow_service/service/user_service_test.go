package service_test

import (
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/dto"
	"github.com/yuudev14-workflow/workflow-service/environment"
	"github.com/yuudev14-workflow/workflow-service/models"
	"github.com/yuudev14-workflow/workflow-service/pkg/logging"
	"github.com/yuudev14-workflow/workflow-service/pkg/repository"
	"github.com/yuudev14-workflow/workflow-service/service"
	"golang.org/x/crypto/bcrypt"
)

var (
	password     = "password"
	sqlxDB       *sqlx.DB
	mock         sqlmock.Sqlmock
	userService  service.UserService
	repo         repository.UserRepository
	expectedUser *models.User
)

func TestMain(m *testing.M) {
	environment.Setup()
	logging.Setup("DEBUG")

	// Create a new mock database connection
	mockDB, sqlmock, mockErr := sqlmock.New()
	mock = sqlmock
	if mockErr != nil {
		logging.Sugar.Fatalf("an error '%s' was not expected when opening a stub database connection", mockErr)
	}
	defer mockDB.Close()

	// Wrap the mock database with sqlx
	sqlxDB = sqlx.NewDb(mockDB, "sqlmock")

	// Create an instance of UserRepositoryImpl with the mock database
	repo = repository.NewUserRepository(sqlxDB)
	userService = service.NewUserService(repo)

	// Set up the expectation
	generateExpectedUser()
	// Run tests
	code := m.Run()

	// Exit
	os.Exit(code)
}

func generateExpectedUser() {

	id, _ := uuid.NewUUID()
	encryptedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	expectedUser = &models.User{
		ID:       id,
		Username: "testuser",
		Email:    "test@example.com",
		Password: string(encryptedPassword),
	}

}

func setMockRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "username", "email", "password"}).
		AddRow(expectedUser.ID, expectedUser.Username, expectedUser.Email, expectedUser.Password)
}

func checkExpectedOutput(t *testing.T, err error, expected interface{}, output interface{}) {
	assert.NoError(t, err)
	assert.Equal(t, expected, output)
}

func TestGetUserByEmailOrUsername(t *testing.T) {

	tests := []struct {
		name     string
		username string
		expected *models.User
	}{
		{
			name:     "user exist",
			username: "testuser",
			expected: expectedUser,
		},
		{
			name:     "user does not exist",
			username: "testuser1",
			expected: nil,
		},
	}

	// Define the expected query and result
	expectedQuery := "SELECT \\* from users WHERE email=\\$1 OR username=\\$1"
	rows := setMockRows()

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			mock.ExpectQuery(expectedQuery).
				WithArgs(test.username).
				WillReturnRows(rows)
			// Call the method being tested
			user, err := userService.GetUserByEmailOrUsername(test.username)
			// Assert the results
			checkExpectedOutput(t, err, test.expected, user)
		})

	}

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByUsername(t *testing.T) {
	// create the tests
	tests := []struct {
		name     string
		username string
		expected *models.User
	}{
		{
			name:     "user exist",
			username: expectedUser.Username,
			expected: expectedUser,
		},
		{
			name:     "user does not exist",
			username: "testuser1",
			expected: nil,
		},
	}

	expectedQuery := "SELECT id, username, email, first_name, last_name from users WHERE username=\\$1"

	rows := setMockRows()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mock.ExpectQuery(expectedQuery).
				WithArgs(test.username).
				WillReturnRows(rows)
			user, err := repo.GetUserByUsername(test.username)
			checkExpectedOutput(t, err, test.expected, user)
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByEmail(t *testing.T) {
	// create the tests
	tests := []struct {
		name     string
		email    string
		expected *models.User
	}{
		{
			name:     "user exist",
			email:    expectedUser.Email,
			expected: expectedUser,
		},
		{
			name:     "user does not exist",
			email:    "testuser1",
			expected: nil,
		},
	}

	expectedQuery := "SELECT \\* from users WHERE email=\\$1"

	rows := setMockRows()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mock.ExpectQuery(expectedQuery).
				WithArgs(test.email).
				WillReturnRows(rows)
			user, err := repo.GetUserByEmail(test.email)
			checkExpectedOutput(t, err, test.expected, user)
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVerifyUser(t *testing.T) {
	expectedQuery := "SELECT \\* from users WHERE email=\\$1 OR username=\\$1"

	tests := []struct {
		name         string
		input        dto.LoginForm
		expected     *models.User
		errorMessage string
		setupMock    func(mock sqlmock.Sqlmock)
	}{
		{
			name: "user exist",
			input: dto.LoginForm{
				Username: expectedUser.Username,
				Password: password,
			},
			expected:     expectedUser,
			errorMessage: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := setMockRows()

				mock.ExpectQuery(expectedQuery).
					WithArgs(expectedUser.Username).
					WillReturnRows(rows)
			},
		},
		{
			name: "user does not exist",
			input: dto.LoginForm{
				Username: "testuser1",
				Password: "password",
			},
			expected:     nil,
			errorMessage: "user is not found",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password"})

				mock.ExpectQuery(expectedQuery).
					WithArgs("testuser1").
					WillReturnRows(rows)
			},
		},
		{
			name: "user exist 2",
			input: dto.LoginForm{
				Username: expectedUser.Username,
				Password: "password1",
			},
			expected:     nil,
			errorMessage: "password is not correct",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := setMockRows()

				mock.ExpectQuery(expectedQuery).
					WithArgs(expectedUser.Username).
					WillReturnRows(rows)
			},
		},
	}

	// Define the expected query and result

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			test.setupMock(mock)
			// Call the method being tested
			user, err := userService.VerifyUser(test.input)
			t.Logf("usersss: %v", user)
			// Assert the results
			if test.expected != nil {
				checkExpectedOutput(t, err, test.expected, user)
			} else {
				assert.Error(t, err)
				assert.Equal(t, err.Error(), test.errorMessage)
			}
			// Ensure all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})

	}

}
