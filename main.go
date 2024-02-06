package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type user struct {
	Id         uuid.UUID  `json:"id,omitempty" db:"id"`
	Name       string     `json:"name,omitempty" db:"name"`
	Email      string     `json:"email,omitempty" db:"email"`
	CreatedAt  time.Time  `json:"createdAt,omitempty" db:"created_at"`
	ApprovedAt *time.Time `json:"approvedAt,omitempty" db:"approved_for_exam_at"`
	ArchivedAt *time.Time `json:"archivedAt,omitempty" db:"archived_at"`
}

var users = make([]user, 0)

func userList(w http.ResponseWriter, r *http.Request) {
	var users []user
	SQL := "select id, name, email, created_at, approved_for_exam_at, archived_at from student"
	err := DB.Select(&users, SQL)
	if err != nil {
		fmt.Println("error reading users", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	RespondJSON(w, http.StatusOK, users)
}

func addUser(w http.ResponseWriter, r *http.Request) {
	var body user
	if err := ParseBody(r.Body, &body); err != nil {
		RespondJSON(w, http.StatusBadRequest, err)
		return
	}

	SQL := `insert into student (id, name, email, created_at) values ($1, $2, $3, $4)`
	_, err := DB.Queryx(SQL, uuid.New(), body.Name, body.Email, time.Now())
	if err != nil {
		RespondJSON(w, http.StatusInternalServerError, err)
		return
	}

	RespondJSON(w, http.StatusCreated, nil)
}

func approveUser(w http.ResponseWriter, r *http.Request) {
	var body user
	if err := ParseBody(r.Body, &body); err != nil {
		fmt.Println("error parsing body", err)
		RespondJSON(w, http.StatusBadRequest, err)
		return
	}
	fmt.Println("body", body)
	SQL := `update student set approved_for_exam_at = $1 where id = $2`
	_, err := DB.Queryx(SQL, time.Now(), body.Id)
	if err != nil {
		RespondJSON(w, http.StatusInternalServerError, err)
		return
	}
	RespondJSON(w, http.StatusCreated, nil)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	var body user
	if err := ParseBody(r.Body, &body); err != nil {
		RespondJSON(w, http.StatusBadRequest, err)
		return
	}
	SQL := `update student set archived_at = now() where id = $1`
	_, err := DB.Queryx(SQL, body.Id)
	if err != nil {
		fmt.Println("error archiving user", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)

	RespondJSON(w, http.StatusOK, nil)
}

var DB *sqlx.DB

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		"localhost", "5433", "postgres", "local", "todo")

	var err error
	DB, err = sqlx.Open("postgres", psqlInfo)
	if err != nil {
		fmt.Println("Unable to Connect to the Database, ", err)
		return
	}
	err = DB.Ping()
	if err != nil {
		fmt.Println("Ping Panic", err)
		return
	}

	router := chi.NewRouter()

	router.Route("/user", func(r chi.Router) {
		r.Get("/", userList)
		r.Post("/", addUser)
		r.Route("/{id}", func(userIdRouter chi.Router) {
			userIdRouter.Put("/approve", approveUser)
			userIdRouter.Delete("/", deleteUser)
		})
	})

	fmt.Println("starting server at port 8080")
	http.ListenAndServe(":8080", router)
}

func RespondJSON(w http.ResponseWriter, statusCode int, body interface{}) {
	w.WriteHeader(statusCode)
	if body != nil {
		if err := EncodeJSONBody(w, body); err != nil {
			fmt.Println(fmt.Errorf("failed to respond JSON with error: %+v", err))
		}
	}
}

func EncodeJSONBody(resp http.ResponseWriter, data interface{}) error {
	return json.NewEncoder(resp).Encode(data)
}

func ParseBody(body io.Reader, out interface{}) error {
	err := json.NewDecoder(body).Decode(out)
	if err != nil {
		return err
	}

	return nil
}
