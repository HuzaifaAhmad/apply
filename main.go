package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"errors"
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
	IslamicEducation string `json:"islamicEdu"`
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
	Course    string
	Verified  bool
}

var db *sql.DB

var cwd = "/home/huzaifa/go/src/github.com/HuzaifaAhmad/apply"

func main() {
	db = connect()
	r := mux.NewRouter()
	r.HandleFunc("/apply/go", handleApplication).Methods("POST")
	r.HandleFunc("/apply/go/", handleVerification).Queries("token", "{token}")

	err := http.ListenAndServe(":8080", r) // set listen port
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

var errInvlidCourse = errors.New("Invalid Course")

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
	if err == errInvlidCourse {
		w.Write([]byte("Invalid Course"))
		return
	} else if err != nil {
		w.Write([]byte("Failed to add user to database"))
		return
	}

	//send admin email
	htmlContent, err := ParseTemplate(filepath.Join(cwd+"/adminEmail.gohtml"), app)
	if err != nil {
		log.Fatalln(err)
	}

	mailRequest := NewMailRequest(
		app.FirstName+" "+app.LastName+" <apply@nakhlahusa.org>",
		"New Application - "+app.FirstName,
		htmlContent,
		[]string{"nakhlahusa@gmail.com"},
	)

	// send mail
	ok, err := mailRequest.SendMail()
	if !ok {
		log.Fatal(err)
	}

	//send user email
	userEmail, err := ParseTemplate(filepath.Join(cwd+"/userEmail.gohtml"), app)
	if err != nil {
		log.Fatalln(err)
	}

	userReq := NewMailRequest(
		"Nakhlah Institute Administration <apply@nakhlahusa.org>",
		"We have recieved your application",
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

	//Checking if course is valid
	courses := []string{"Arabic Forensics 1", "Arabic Forensics 2", "Arabic Forensics 3", "Al Mutūn Study Group", "Tafsir (Urdu)", "Tafsir in English, The Forty Ahadith of Imam Al-Nawawi"}
	var ok = false
	for _, item := range courses {
		if item == app.Course {
			ok = true
		}
	}
	if !ok {
		return "", errInvlidCourse
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

//-----------------------------------------------------------------------------------------------------------------------
//to Verify users
var errUserVerified = errors.New("User already verified")

func handleVerification(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	token := vars["token"]
	user, err := getUserByToken(token)

	if err == sql.ErrNoRows {
		fmt.Fprint(w, "No User Found")
		return
	} else if err == errUserVerified {
		fmt.Fprint(w, errUserVerified)
		return
	} else if err != nil {
		log.Fatal(err)
	}
	//Parsing email template
	userEmail, err := ParseTemplate(filepath.Join(cwd+"/userVerifiedEmail.gohtml"), user)
	if err != nil {
		log.Fatalln(err)
	}
	if ok := verifyUser(user); ok {

		//Email user that they have been verfified
		userReq := NewMailRequest(
			"Nakhlah Institue Administration <apply@nakhlahusa.org>",
			"Welcome to Nakhlah Institute",
			userEmail,
			[]string{user.Email},
		)

		// send mail
		ok, err = userReq.SendMail()
		if !ok {
			log.Fatal(err)
		}

		fmt.Fprint(w, "User Verified")

	} else {
		fmt.Fprintln(w, "User not verified")
	}
}

func getUserByToken(token string) (User, error) {
	var usr User
	sqlSmnt := `SELECT id, firstName, lastName, email, username, password, course, verified FROM tempUsers WHERE token = $1`
	err := db.QueryRow(sqlSmnt, token).Scan(&usr.ID, &usr.FirstName, &usr.LastName, &usr.Email, &usr.Username, &usr.Password, &usr.Course, &usr.Verified)
	if err != nil {
		return User{}, err
	} else if usr.Verified == true {
		return User{}, errUserVerified
	}
	return usr, nil
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

	//insert user in user table for moodle authentication
	sqlSmnt = `INSERT INTO users (id, firstName, lastName, email, username, password) VALUES($1, $2, $3, $4, $5, $6)`
	_, err = db.Exec(sqlSmnt, usr.ID, usr.FirstName, usr.LastName, usr.Email, usr.Username, usr.Password)
	if err != nil {
		log.Fatal(err)
		return false
	}

	//insert user in course table for moodle enrolment
	sqlSmnt = `INSERT INTO course (userid, coursename) VALUES($1, $2)`
	_, err = db.Exec(sqlSmnt, usr.ID, usr.Course)
	if err != nil {
		log.Fatal(err)
		return false
	}
	return true
}
