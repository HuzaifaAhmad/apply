package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	uuid "github.com/nu7hatch/gouuid"

	_ "github.com/lib/pq"
)

//TODO: Error if uesr exist

//Application stores information from application form
type Application struct {
	FirstName        string `json:"firstName"`
	LastName         string `json:"lastName"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	Phone            string `json:"phone"`
	Age              string `json:"age"`
	Location         string `json:"location"`
	Course           string `json:"course"`
	Education        string `json:"ed"`
	IslamicEducation string `json:"islamic-ed"`
	Expectations     string `json:"expectations"`
	Hear             string `json:"hear"`
	Read             string `json:"read"`
	Username         string `json:"username"`
	Token            string
}

//User stores user info from db
type User struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
	Username  string
	Password  string
}

var db *sql.DB

func main() {
	db = connect()
	r := mux.NewRouter()

	r.HandleFunc("/apply", handleApplication).Methods("POST")
	r.HandleFunc("/apply/", handleVerification).Queries("token", "{token}")

	err := http.ListenAndServe(":8080", r) // set listen port
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleApplication(w http.ResponseWriter, r *http.Request) {
	var app Application
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&app)
	if err != nil {
		log.Fatal(err)
		w.Write([]byte("Failed to parse JSON"))
		return
	}

	//check if User exist
	exist := checkIfUserExist(app)
	if exist == 100 {
		w.Write([]byte("Email already exist"))
		return
	} else if exist == 200 {
		w.Write([]byte("Username already exist"))
		return
	}

	app.Token, err = addTempUser(app)
	if err != nil {
		w.Write([]byte("Failed to add user to database"))
	}

	htmlContent, err := ParseTemplate(filepath.Join("./adminEmail.gohtml"), app)
	if err != nil {
		log.Fatalln(err)
	}

	mailRequest := NewMailRequest(
		app.FirstName+" "+app.LastName+" <apply@nakhlahusa.org>",
		"New Application - "+app.FirstName,
		htmlContent,
		[]string{"huzi8014@gmail.com"},
	)

	// send mail
	ok, err := mailRequest.SendMail()
	if !ok {
		log.Fatal(err)
	}

	//send user emial
	userEmail, err := ParseTemplate(filepath.Join("./userEmail.gohtml"), app)
	if err != nil {
		log.Fatalln(err)
	}

	userReq := NewMailRequest(
		app.FirstName+" "+app.LastName+" <apply@nakhlahusa.org>",
		"New Application - "+app.FirstName,
		userEmail,
		[]string{app.Email},
	)

	// send mail
	ok, err = userReq.SendMail()
	if !ok {
		log.Fatal(err)
	}

}

func checkIfUserExist(app Application) int {
	var id int

	//Email
	sqlSmnt := `SELECT id FROM tempUsers WHERE email=$1`
	_ = db.QueryRow(sqlSmnt, app.Email).Scan(&id)
	if id > 0 {
		fmt.Println("got here")
		return 100
	}

	sqlSmnt = `SELECT id FROM tempUsers WHERE username=$1`
	_ = db.QueryRow(sqlSmnt, app.Username).Scan(&id)
	if id > 0 {
		return 200
	}

	return 0
}

func addTempUser(app Application) (string, error) {
	token, err := generateToken()
	if err != nil {
		log.Println(err)
		return "", err
	}

	//Hash Password
	app.Password, err = encryptPass(app.Password)
	if err != nil {
		log.Println(err)
		return "", err
	}

	//Add user to database
	sqlSmnt := `INSERT INTO tempUsers (firstName, lastName, email, password, phone, age, location, course, education, islamicEducation, expectations, hear, read, dateCreated, token, verified, username) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, CURRENT_TIMESTAMP, $14, false, $15)`

	_, err = db.Exec(sqlSmnt, app.FirstName, app.LastName, app.Email, app.Password, app.Phone, app.Age, app.Location, app.Course, app.Education, app.IslamicEducation, app.Expectations, app.Hear, app.Read, token, app.Username)

	if err != nil {
		log.Println(err)
		return "", err
	}

	return token, nil
}

func generateToken() (string, error) {
	u4, err := uuid.NewV4()
	if err != nil {
		fmt.Println("error:", err)
		return "", err
	}
	return u4.String(), nil
}

func encryptPass(pass string) (string, error) {
	h := sha1.New()
	io.WriteString(h, pass)
	hash := fmt.Sprintf("%x", h.Sum(nil))
	return hash, nil
}

func handleVerification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	user, err := getUserByToken(token)
	if err == sql.ErrNoRows {
		fmt.Fprint(w, "No User Found")
		return
	} else if err != nil {
		log.Fatal(err)
	}
	if ok := verifyUser(user); ok {
		fmt.Fprint(w, "User Verified")
	} else {
		fmt.Fprint(w, "User not verified")
	}
}

func getUserByToken(token string) (User, error) {
	var usr User
	sqlSmnt := `SELECT id, firstName, lastName, email, username, password FROM tempUsers WHERE token = $1`
	err := db.QueryRow(sqlSmnt, token).Scan(&usr.ID, &usr.FirstName, &usr.LastName, &usr.Email, &usr.Username, &usr.Password)
	switch {
	case err == sql.ErrNoRows:
		return User{}, err
	case err != nil:
		return User{}, err
	default:
		return usr, nil
	}

}

func verifyUser(usr User) bool {
	// add to real Users table
	// change verified status on tempUsers table'
	sqlSmnt := `UPDATE tempUsers SET verified = true WHERE id = $1`
	_, err := db.Exec(sqlSmnt, usr.ID)
	if err != nil {
		log.Fatal(err)
		return false
	}

	sqlSmnt = `INSERT INTO users (firstName, lastName, email, username, password) VALUES($1, $2, $3, $4, $5)`
	_, err = db.Exec(sqlSmnt, usr.FirstName, usr.LastName, usr.Email, usr.Username, usr.Password)
	if err != nil {
		return false
	}
	return true
}
