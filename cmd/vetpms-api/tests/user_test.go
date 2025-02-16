package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/os-foundry/vetpms/cmd/vetpms-api/internal/handlers"
	"github.com/os-foundry/vetpms/internal/platform/auth"
	"github.com/os-foundry/vetpms/internal/platform/web"
	productBolt "github.com/os-foundry/vetpms/internal/product/bolt"
	productPq "github.com/os-foundry/vetpms/internal/product/postgres"
	"github.com/os-foundry/vetpms/internal/tests"
	"github.com/os-foundry/vetpms/internal/user"
	userBolt "github.com/os-foundry/vetpms/internal/user/bolt"
	userPq "github.com/os-foundry/vetpms/internal/user/postgres"
)

// TestUsers is the entry point for testing user management functions.
func TestUsers(t *testing.T) {
	tt := []string{"postgres", "bolt"}
	for _, tc := range tt {
		test := tests.NewIntegration(t, tc)
		defer test.Teardown()

		var handler http.Handler
		shutdown := make(chan os.Signal, 1)
		switch tc {
		case "postgres":
			handler = handlers.API(shutdown, test.Log, userPq.Postgres{test.Pq}, productPq.Postgres{test.Pq}, test.Authenticator)
		case "bolt":
			handler = handlers.API(shutdown, test.Log, userBolt.Bolt{test.Bolt}, productBolt.Bolt{test.Bolt}, test.Authenticator)
		default:
			t.Fatalf("test case should be bolt or postgres")
		}

		tests := UserTests{
			app:        handler,
			userToken:  test.Token("user@example.com", "gophers"),
			adminToken: test.Token("admin@example.com", "gophers"),
		}

		t.Run("getToken401", tests.getToken401)
		t.Run("getToken200", tests.getToken200)
		t.Run("postUser400", tests.postUser400)
		t.Run("postUser401", tests.postUser401)
		t.Run("postUser403", tests.postUser403)
		t.Run("getUser400", tests.getUser400)
		t.Run("getUser403", tests.getUser403)
		t.Run("getUser404", tests.getUser404)
		t.Run("deleteUserNotFound", tests.deleteUserNotFound)
		t.Run("putUser404", tests.putUser404)
		t.Run("crudUsers", tests.crudUser)
	}
}

// UserTests holds methods for each user subtest. This type allows passing
// dependencies for tests while still providing a convenient syntax when
// subtests are registered.
type UserTests struct {
	app        http.Handler
	userToken  string
	adminToken string
}

// getToken401 ensures an unknown user can't generate a token.
func (ut *UserTests) getToken401(t *testing.T) {
	r := httptest.NewRequest("GET", "/v1/users/token", nil)
	w := httptest.NewRecorder()

	r.SetBasicAuth("unknown@example.com", "some-password")

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to deny tokens to unknown users.")
	{
		t.Log("\tTest 0:\tWhen fetching a token with an unrecognized email.")
		{
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("\t%s\tShould receive a status code of 401 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 401 for the response.", tests.Success)
		}
	}
}

// getToken200
func (ut *UserTests) getToken200(t *testing.T) {

	r := httptest.NewRequest("GET", "/v1/users/token", nil)
	w := httptest.NewRecorder()

	r.SetBasicAuth("admin@example.com", "gophers")

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to issues tokens to known users.")
	{
		t.Log("\tTest 0:\tWhen fetching a token with valid credentials.")
		{
			if w.Code != http.StatusOK {
				t.Fatalf("\t%s\tShould receive a status code of 200 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 200 for the response.", tests.Success)

			var got struct {
				Token string `json:"token"`
			}
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("\t%s\tShould be able to unmarshal the response : %v", tests.Failed, err)
			}
			t.Logf("\t%s\tShould be able to unmarshal the response.", tests.Success)

			// TODO(jlw) Should we ensure the token is valid?
		}
	}
}

// postUser400 validates a user can't be created with the endpoint
// unless a valid user document is submitted.
func (ut *UserTests) postUser400(t *testing.T) {
	body, err := json.Marshal(&user.NewUser{})
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("POST", "/v1/users", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate a new user can't be created with an invalid document.")
	{
		t.Log("\tTest 0:\tWhen using an incomplete user value.")
		{
			if w.Code != http.StatusBadRequest {
				t.Fatalf("\t%s\tShould receive a status code of 400 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 400 for the response.", tests.Success)

			// Inspect the response.
			var got web.ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("\t%s\tShould be able to unmarshal the response to an error type : %v", tests.Failed, err)
			}
			t.Logf("\t%s\tShould be able to unmarshal the response to an error type.", tests.Success)

			// Define what we want to see.
			want := web.ErrorResponse{
				Error: "field validation error",
				Fields: []web.FieldError{
					{Field: "name", Error: "name is a required field"},
					{Field: "email", Error: "email is a required field"},
					{Field: "roles", Error: "roles is a required field"},
					{Field: "password", Error: "password is a required field"},
				},
			}

			// We can't rely on the order of the field errors so they have to be
			// sorted. Tell the cmp package how to sort them.
			sorter := cmpopts.SortSlices(func(a, b web.FieldError) bool {
				return a.Field < b.Field
			})

			if diff := cmp.Diff(want, got, sorter); diff != "" {
				t.Fatalf("\t%s\tShould get the expected result. Diff:\n%s", tests.Failed, diff)
			}
			t.Logf("\t%s\tShould get the expected result.", tests.Success)
		}
	}
}

// postUser401 validates a user can't be created unless the calling user is
// authenticated.
func (ut *UserTests) postUser401(t *testing.T) {
	body, err := json.Marshal(&user.User{})
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("POST", "/v1/users", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.userToken)

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate a new user can't be created with an invalid document.")
	{
		t.Log("\tTest 0:\tWhen using an incomplete user value.")
		{
			if w.Code != http.StatusForbidden {
				t.Fatalf("\t%s\tShould receive a status code of 403 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 403 for the response.", tests.Success)
		}
	}
}

// postUser403 validates a user can't be created unless the calling user is
// an admin user. Regular users can't do this.
func (ut *UserTests) postUser403(t *testing.T) {
	body, err := json.Marshal(&user.User{})
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("POST", "/v1/users", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	// Not setting the Authorization header

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate a new user can't be created with an invalid document.")
	{
		t.Log("\tTest 0:\tWhen using an incomplete user value.")
		{
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("\t%s\tShould receive a status code of 401 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 401 for the response.", tests.Success)
		}
	}
}

// getUser400 validates a user request for a malformed userid.
func (ut *UserTests) getUser400(t *testing.T) {
	id := "12345"

	r := httptest.NewRequest("GET", "/v1/users/"+id, nil)
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate getting a user with a malformed userid.")
	{
		t.Logf("\tTest 0:\tWhen using the new user %s.", id)
		{
			if w.Code != http.StatusBadRequest {
				t.Fatalf("\t%s\tShould receive a status code of 400 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 400 for the response.", tests.Success)

			recv := w.Body.String()
			resp := `{"error":"ID is not in its proper form"}`
			if resp != recv {
				t.Log("Got :", recv)
				t.Log("Want:", resp)
				t.Fatalf("\t%s\tShould get the expected result.", tests.Failed)
			}
			t.Logf("\t%s\tShould get the expected result.", tests.Success)
		}
	}
}

// getUser403 validates a regular user can't fetch anyone but themselves
func (ut *UserTests) getUser403(t *testing.T) {
	t.Log("Given the need to validate regular users can't fetch other users.")
	{
		t.Logf("\tTest 0:\tWhen fetching the admin user as a regular user.")
		{
			r := httptest.NewRequest("GET", "/v1/users/"+tests.AdminID, nil)
			w := httptest.NewRecorder()

			r.Header.Set("Authorization", "Bearer "+ut.userToken)

			ut.app.ServeHTTP(w, r)

			if w.Code != http.StatusForbidden {
				t.Fatalf("\t%s\tShould receive a status code of 403 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 403 for the response.", tests.Success)

			recv := w.Body.String()
			resp := `{"error":"Attempted action is not allowed"}`
			if resp != recv {
				t.Log("Got :", recv)
				t.Log("Want:", resp)
				t.Fatalf("\t%s\tShould get the expected result.", tests.Failed)
			}
			t.Logf("\t%s\tShould get the expected result.", tests.Success)
		}

		t.Logf("\tTest 1:\tWhen fetching the user as themselves.")
		{

			r := httptest.NewRequest("GET", "/v1/users/"+tests.UserID, nil)
			w := httptest.NewRecorder()

			r.Header.Set("Authorization", "Bearer "+ut.userToken)

			ut.app.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("\t%s\tShould receive a status code of 200 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 200 for the response.", tests.Success)
		}
	}
}

// getUser404 validates a user request for a user that does not exist with the endpoint.
func (ut *UserTests) getUser404(t *testing.T) {
	id := "c50a5d66-3c4d-453f-af3f-bc960ed1a503"

	r := httptest.NewRequest("GET", "/v1/users/"+id, nil)
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate getting a user with an unknown id.")
	{
		t.Logf("\tTest 0:\tWhen using the new user %s.", id)
		{
			if w.Code != http.StatusNotFound {
				t.Fatalf("\t%s\tShould receive a status code of 404 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 404 for the response.", tests.Success)

			recv := w.Body.String()
			resp := "User not found"
			if !strings.Contains(recv, resp) {
				t.Log("Got :", recv)
				t.Log("Want:", resp)
				t.Fatalf("\t%s\tShould get the expected result.", tests.Failed)
			}
			t.Logf("\t%s\tShould get the expected result.", tests.Success)
		}
	}
}

// deleteUserNotFound validates deleting a user that does not exist is not a failure.
func (ut *UserTests) deleteUserNotFound(t *testing.T) {
	id := "a71f77b2-b1ae-4964-a847-f9eecba09d74"

	r := httptest.NewRequest("DELETE", "/v1/users/"+id, nil)
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate deleting a user that does not exist.")
	{
		t.Logf("\tTest 0:\tWhen using the new user %s.", id)
		{
			if w.Code != http.StatusNoContent {
				t.Fatalf("\t%s\tShould receive a status code of 204 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 204 for the response.", tests.Success)
		}
	}
}

// putUser404 validates updating a user that does not exist.
func (ut *UserTests) putUser404(t *testing.T) {
	u := user.UpdateUser{
		Name: tests.StringPointer("Doesn't Exist"),
	}

	id := "3097c45e-780a-421b-9eae-43c2fda2bf14"

	body, err := json.Marshal(&u)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("PUT", "/v1/users/"+id, bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate updating a user that does not exist.")
	{
		t.Logf("\tTest 0:\tWhen using the new user %s.", id)
		{
			if w.Code != http.StatusNotFound {
				t.Fatalf("\t%s\tShould receive a status code of 404 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 404 for the response.", tests.Success)

			recv := w.Body.String()
			resp := "User not found"
			if !strings.Contains(recv, resp) {
				t.Log("Got :", recv)
				t.Log("Want:", resp)
				t.Fatalf("\t%s\tShould get the expected result.", tests.Failed)
			}
			t.Logf("\t%s\tShould get the expected result.", tests.Success)
		}
	}
}

// crudUser performs a complete test of CRUD against the api.
func (ut *UserTests) crudUser(t *testing.T) {
	nu := ut.postUser201(t)
	defer ut.deleteUser204(t, nu.ID)

	ut.getUser200(t, nu.ID)
	ut.putUser204(t, nu.ID)
	ut.putUser403(t, nu.ID)
}

// postUser201 validates a user can be created with the endpoint.
func (ut *UserTests) postUser201(t *testing.T) user.User {
	nu := user.NewUser{
		Name:            "William Doe",
		Email:           "bill@example.com",
		Roles:           []string{auth.RoleAdmin},
		Password:        "gophers",
		PasswordConfirm: "gophers",
	}

	body, err := json.Marshal(&nu)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("POST", "/v1/users", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)

	ut.app.ServeHTTP(w, r)

	// u is the value we will return.
	var u user.User

	t.Log("Given the need to create a new user with the users endpoint.")
	{
		t.Log("\tTest 0:\tWhen using the declared user value.")
		{
			if w.Code != http.StatusCreated {
				t.Fatalf("\t%s\tShould receive a status code of 201 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 201 for the response.", tests.Success)

			if err := json.NewDecoder(w.Body).Decode(&u); err != nil {
				t.Fatalf("\t%s\tShould be able to unmarshal the response : %v", tests.Failed, err)
			}

			// Define what we wanted to receive. We will just trust the generated
			// fields like ID and Dates so we copy u.
			want := u
			want.Name = "William Doe"
			want.Email = "bill@example.com"
			want.Roles = []string{auth.RoleAdmin}

			if diff := cmp.Diff(want, u); diff != "" {
				t.Fatalf("\t%s\tShould get the expected result. Diff:\n%s", tests.Failed, diff)
			}
			t.Logf("\t%s\tShould get the expected result.", tests.Success)
		}
	}

	return u
}

// deleteUser200 validates deleting a user that does exist.
func (ut *UserTests) deleteUser204(t *testing.T, id string) {
	r := httptest.NewRequest("DELETE", "/v1/users/"+id, nil)
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate deleting a user that does exist.")
	{
		t.Logf("\tTest 0:\tWhen using the new user %s.", id)
		{
			if w.Code != http.StatusNoContent {
				t.Fatalf("\t%s\tShould receive a status code of 204 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 204 for the response.", tests.Success)
		}
	}
}

// getUser200 validates a user request for an existing userid.
func (ut *UserTests) getUser200(t *testing.T, id string) {
	r := httptest.NewRequest("GET", "/v1/users/"+id, nil)
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to validate getting a user that exsits.")
	{
		t.Logf("\tTest 0:\tWhen using the new user %s.", id)
		{
			if w.Code != http.StatusOK {
				t.Fatalf("\t%s\tShould receive a status code of 200 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 200 for the response.", tests.Success)

			var u user.User
			if err := json.NewDecoder(w.Body).Decode(&u); err != nil {
				t.Fatalf("\t%s\tShould be able to unmarshal the response : %v", tests.Failed, err)
			}

			// Define what we wanted to receive. We will just trust the generated
			// fields like Dates so we copy p.
			want := u
			want.ID = id
			want.Name = "William Doe"
			want.Email = "bill@example.com"
			want.Roles = []string{auth.RoleAdmin}

			if diff := cmp.Diff(want, u); diff != "" {
				t.Fatalf("\t%s\tShould get the expected result. Diff:\n%s", tests.Failed, diff)
			}
			t.Logf("\t%s\tShould get the expected result.", tests.Success)
		}
	}
}

// putUser204 validates updating a user that does exist.
func (ut *UserTests) putUser204(t *testing.T, id string) {
	body := `{"name": "John Doe"}`

	r := httptest.NewRequest("PUT", "/v1/users/"+id, strings.NewReader(body))
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.adminToken)

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to update a user with the users endpoint.")
	{
		t.Log("\tTest 0:\tWhen using the modified user value.")
		{
			if w.Code != http.StatusNoContent {
				t.Fatalf("\t%s\tShould receive a status code of 204 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 204 for the response.", tests.Success)

			r = httptest.NewRequest("GET", "/v1/users/"+id, nil)
			w = httptest.NewRecorder()

			r.Header.Set("Authorization", "Bearer "+ut.adminToken)

			ut.app.ServeHTTP(w, r)

			if w.Code != http.StatusOK {
				t.Fatalf("\t%s\tShould receive a status code of 200 for the retrieve : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 200 for the retrieve.", tests.Success)

			var ru user.User
			if err := json.NewDecoder(w.Body).Decode(&ru); err != nil {
				t.Fatalf("\t%s\tShould be able to unmarshal the response : %v", tests.Failed, err)
			}

			if ru.Name != "John Doe" {
				t.Fatalf("\t%s\tShould see an updated Name : got %q want %q", tests.Failed, ru.Name, "John Doe")
			}
			t.Logf("\t%s\tShould see an updated Name.", tests.Success)

			if ru.Email != "bill@example.com" {
				t.Fatalf("\t%s\tShould not affect other fields like Email : got %q want %q", tests.Failed, ru.Email, "bill@example.com")
			}
			t.Logf("\t%s\tShould not affect other fields like Email.", tests.Success)
		}
	}
}

// putUser403 validates that a user can't modify users unless they are an admin.
func (ut *UserTests) putUser403(t *testing.T, id string) {
	body := `{"name": "Jane Doe"}`

	r := httptest.NewRequest("PUT", "/v1/users/"+id, strings.NewReader(body))
	w := httptest.NewRecorder()

	r.Header.Set("Authorization", "Bearer "+ut.userToken)

	ut.app.ServeHTTP(w, r)

	t.Log("Given the need to update a user with the users endpoint.")
	{
		t.Log("\tTest 0:\tWhen a non-admin user makes a request")
		{
			if w.Code != http.StatusForbidden {
				t.Fatalf("\t%s\tShould receive a status code of 403 for the response : %v", tests.Failed, w.Code)
			}
			t.Logf("\t%s\tShould receive a status code of 403 for the response.", tests.Success)
		}
	}
}
