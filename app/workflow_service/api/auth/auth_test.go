package auth_api_test

import (
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/db"
	"github.com/yuudev14-workflow/workflow-service/environment"
	"github.com/yuudev14-workflow/workflow-service/models"
	"github.com/yuudev14-workflow/workflow-service/pkg/logging"
	"github.com/yuudev14-workflow/workflow-service/pkg/repository"
	"github.com/yuudev14-workflow/workflow-service/service"
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
	gin.SetMode(gin.TestMode)
	db.SetupDB("")

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
	id, _ := uuid.NewUUID()
	expectedUser = &models.User{
		ID:       id,
		Username: "testuser",
		Email:    "test@example.com",
	}

	// Run tests
	code := m.Run()

	// Exit
	os.Exit(code)
}

// func TestSignupRoute(t *testing.T) {
// 	test := struct {
// 		user     models.User
// 		expected *models.User
// 	}{
// 		user: models.User{
// 			Email:    "test111@gmail.com",
// 			Username: "test111",
// 			Password: "password",
// 		},
// 		expected: &models.User{
// 			Email:    "test111@gmail.com",
// 			Username: "test111",
// 			Password: "password",
// 		},
// 	}
// 	router := api.InitRouter()

// 	w := httptest.NewRecorder()

// 	users := map[string]string{
// 		"username": test.user.Username,
// 		"email":    test.user.Email,
// 		"password": test.user.Password,
// 	}

// 	// Marshal the slice of users to JSON
// 	jsonData, err := json.Marshal(users)
// 	if err != nil {
// 		panic(err)
// 	}

// 	expectedQuery := "INSERT INTO users \\(email, username, password\\) VALUES \\(\\$1, \\$2, \\$3\\)"

// 	rows := sqlmock.NewRows([]string{"username", "email"}).AddRow(test.user.Username, test.user.Email)

// 	mock.ExpectExec(expectedQuery).
// 		WithArgs(test.user.Email, test.user.Username, test.user.Password).WillReturnResult(sqlmock.NewResult(1, 1))

// 	mock.ExpectQuery("SELECT id, username, email, first_name, last_name from users WHERE username=\\$1").
// 		WithArgs(test.user.Username).
// 		WillReturnRows(rows)

// 	req, _ := http.NewRequest("POST", "/api/auth/v1/sign-up", bytes.NewBuffer(jsonData))
// 	router.ServeHTTP(w, req)
// 	var responseBody map[string]interface{}
// 	err = json.Unmarshal(w.Body.Bytes(), &responseBody)
// 	if err != nil {
// 		t.Errorf("error in unmarshaling response body... %v", err)
// 		return
// 	}

// 	assert.Equal(t, 200, w.Code)
// }

// import (
// 	"bytes"
// 	"encoding/json"
// 	"net/http"
// 	"net/http/httptest"
// 	"os"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/yuudev14-workflow/workflow-service/api"
// 	"github.com/yuudev14-workflow/workflow-service/pkg/logging"
// )

// func TestMain(m *testing.M) {
// 	test_helpers.SetupTestEnvironment("../../.env.test")
// 	exitCode := m.Run()

// 	os.Exit(exitCode)
// }

// func TestSignupRoute(t *testing.T) {
// 	router := api.InitRouter()

// 	w := httptest.NewRecorder()

// 	users := map[string]string{
// 		"username": "john_auth_doe",
// 		"email":    "john_auth@example.com",
// 		"password": "password",
// 	}

// 	// Marshal the slice of users to JSON
// 	jsonData, err := json.Marshal(users)
// 	if err != nil {
// 		panic(err)
// 	}
// 	req, _ := http.NewRequest("POST", "/api/auth/v1/sign-up", bytes.NewBuffer(jsonData))
// 	router.ServeHTTP(w, req)
// 	var responseBody map[string]interface{}
// 	err = json.Unmarshal(w.Body.Bytes(), &responseBody)
// 	if err != nil {
// 		t.Errorf("error in unmarshaling response body... %v", err)
// 		return
// 	}
// 	logging.Logger.Info(responseBody)

// 	assert.Equal(t, 200, w.Code)
// }

// func ptrStr(s string) *string {
// 	return &s
// }

// func TestLoginRoute(t *testing.T) {
// 	tests := []struct {
// 		username     string
// 		password     string
// 		expectedCode int
// 		expectedMsg  *string
// 	}{
// 		{
// 			username:     "john_auth_doe",
// 			password:     "password",
// 			expectedCode: 200,
// 			expectedMsg:  nil,
// 		},
// 		{
// 			username:     "john_auth_doe",
// 			password:     "passwor",
// 			expectedCode: 400,
// 			expectedMsg:  ptrStr("password is not correct"),
// 		},
// 		{
// 			username:     "john_auth_e",
// 			password:     "password",
// 			expectedCode: 400,
// 			expectedMsg:  ptrStr("user is not found"),
// 		},
// 	}
// 	router := api.InitRouter()

// 	for _, e := range tests {
// 		w := httptest.NewRecorder()
// 		users := map[string]string{
// 			"username": e.username,
// 			"password": e.password,
// 		}

// 		logging.Logger.Debug(users, e)

// 		// Marshal the slice of users to JSON
// 		jsonData, err := json.Marshal(users)
// 		if err != nil {
// 			panic(err)
// 		}
// 		req, _ := http.NewRequest("POST", "/api/auth/v1/login", bytes.NewBuffer(jsonData))
// 		router.ServeHTTP(w, req)
// 		var responseBody map[string]interface{}
// 		err = json.Unmarshal(w.Body.Bytes(), &responseBody)
// 		if err != nil {
// 			t.Errorf("error in unmarshaling response body... %v", err)
// 			return
// 		}
// 		logging.Logger.Info(responseBody)

// 		assert.Equal(t, e.expectedCode, w.Code)

// 		value, ok := responseBody["error"]
// 		if ok {
// 			assert.Equal(t, *e.expectedMsg, value)
// 		}
// 	}

// }
