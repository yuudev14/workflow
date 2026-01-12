package repository_test

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/internal/repository"
	"github.com/yuudev14-workflow/workflow-service/models"
)

var (
	sqlxDB       *sqlx.DB
	mock         sqlmock.Sqlmock
	repo         repository.UserRepository
	expectedUser *models.User
)

// func TestMain(m *testing.M) {
// 	environment.Setup()
// 	logging.Setup("DEBUG")

// 	// Create a new mock database connection
// 	mockDB, sqlmock, mockErr := sqlmock.New()
// 	mock = sqlmock
// 	if mockErr != nil {
// 		logging.Sugar.Fatalf("an error '%s' was not expected when opening a stub database connection", mockErr)
// 	}
// 	defer mockDB.Close()

// 	// Wrap the mock database with sqlx
// 	sqlxDB = sqlx.NewDb(mockDB, "sqlmock")

// 	// Create an instance of UserRepositoryImpl with the mock database
// 	repo = repository.NewUserRepository(sqlxDB)

// 	// Set up the expectation
// 	id, _ := uuid.NewUUID()
// 	expectedUser = &models.User{
// 		ID:       id,
// 		Username: "testuser",
// 		Email:    "test@example.com",
// 	}

// 	// Run tests
// 	code := m.Run()

// 	// Exit
// 	os.Exit(code)
// }

func TestCreateUserWhereUserIsNotAvaliable(t *testing.T) {
	test := struct {
		user     models.User
		expected *models.User
	}{
		user: models.User{
			Email:    "test111@gmail.com",
			Username: "test111",
			Password: "password",
		},
		expected: &models.User{
			Email:    "test111@gmail.com",
			Username: "test111",
			Password: "password",
		},
	}

	expectedQuery := "INSERT INTO users \\(email, username, password\\) VALUES \\(\\$1, \\$2, \\$3\\)"

	rows := sqlmock.NewRows([]string{"username", "email"}).AddRow(test.user.Username, test.user.Email)

	mock.ExpectExec(expectedQuery).
		WithArgs(test.user.Email, test.user.Username, test.user.Password).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery("SELECT id, username, email, first_name, last_name from users WHERE username=\\$1").
		WithArgs(test.user.Username).
		WillReturnRows(rows)
	user, err := repo.CreateUser(&test.user)
	t.Logf("error: %v, user, %v", err, user)

	assert.Equal(t, test.expected.Email, user.Email)
	assert.Equal(t, test.expected.Username, user.Username)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateUserWhereUserIsAvaliable(t *testing.T) {
	user := &models.User{
		Email:    "test111@gmail.com",
		Username: "test111",
		Password: "password",
	}

	expectedQuery := "INSERT INTO users \\(email, username, password\\) VALUES \\(\\$1, \\$2, \\$3\\)"

	mock.ExpectExec(expectedQuery).
		WithArgs(user.Email, user.Username, user.Password).WillReturnResult(sqlmock.NewResult(1, 1)).WillReturnError(fmt.Errorf("some error"))

	user, err := repo.CreateUser(user)
	t.Logf("error: %v, user, %v", err, user)

	assert.Error(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}
