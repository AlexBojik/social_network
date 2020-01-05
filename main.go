package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"html/template"
	"log"
	"net/http"
	"social_network/config"
)

var conf = config.NewConfig()
var database *sql.DB
var store = sessions.NewCookieStore([]byte(conf.SessionKey))

var GENDER = map[string]string{
	"1": "Мужской",
	"2": "Женский",
}

type ProfilesModel struct {
	CurrentUser string
	Profiles    []ProfileModel
}

type ProfileModel struct {
	Id        uint
	FirstName string
	LastName  string
	Age       string
	Gender    string
	About     string
	City      string
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/register.html")
}

func profilesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	query := "select first_name, last_name, age, gender, city, about from profiles WHERE id = ?"
	rows, err := database.Query(query, vars["id"])

	if err != nil {
		log.Println(err)
	}
	defer rows.Close()

	data := ProfileModel{}
	if rows.Next() {
		err := rows.Scan(&data.FirstName, &data.LastName, &data.Age, &data.Gender, &data.City, &data.About)
		if err != nil {
			fmt.Println(err)
		}
		data.Gender = GENDER[data.Gender]
	}

	tmpl, _ := template.ParseFiles("templates/profile.html")
	tmpl.Execute(w, data)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("indexHandler")
	session, err := store.Get(r, conf.SessionName)
	if err != nil {
		fmt.Println(err)
	}

	query := "select id, first_name, last_name, age, gender, city from profiles"
	rows, err := database.Query(query)
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()

	var profiles []ProfileModel
	for rows.Next() {
		p := ProfileModel{}
		err := rows.Scan(&p.Id, &p.FirstName, &p.LastName, &p.Age, &p.Gender, &p.City)
		if err != nil {
			fmt.Println(err)
		}
		p.Gender = GENDER[p.Gender]
		profiles = append(profiles, p)
	}

	var currentUser string
	if session.Values["currentUser"] != nil {
		currentUser = fmt.Sprintf("%v", session.Values["currentUser"])
	}
	data := ProfilesModel{
		CurrentUser: currentUser,
		Profiles:    profiles,
	}

	tmpl, _ := template.ParseFiles("templates/index.html")
	tmpl.Execute(w, data)
}

func registrationHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
	}

	firstName := r.FormValue("firstName")
	lastName := r.FormValue("lastName")
	age := r.PostForm.Get("age")
	gender := r.FormValue("gender")
	about := r.FormValue("about")
	city := r.FormValue("city")
	login := r.FormValue("login")
	password := r.FormValue("password")

	query := "insert into profiles (first_name, last_name, age, gender, about, city, login, password) values (?,?,?,?,?,?,?,?)"
	_, err = database.Exec(query, firstName, lastName, age, gender, about, city, login, password)

	if err != nil {
		fmt.Println(err)
	}

	http.Redirect(w, r, "/", 301)
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
	}
	login := r.FormValue("login")
	password := r.FormValue("password")

	query := "select first_name, last_name from profiles WHERE login=? AND password=?"
	rows, err := database.Query(query, login, password)
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()

	p := ProfileModel{}
	if rows.Next() {
		rows.Scan(&p.FirstName, &p.LastName)

		session, err := store.Get(r, conf.SessionName)
		if err != nil {
			fmt.Println(err)
		}
		session.Values["currentUser"] = p.LastName + " " + p.FirstName
		err = sessions.Save(r, w)
		if err != nil {
			fmt.Println(err)
		}
	}
	http.Redirect(w, r, "/", 301)
}

func main() {
	db, err := sql.Open("mysql", conf.DatabaseUser+":"+conf.DatabasePassword+"@tcp("+conf.DatabaseServer+")/"+conf.Database)
	if err != nil {
		fmt.Println(err)
	}

	database = db
	defer db.Close()

	router := mux.NewRouter()
	router.HandleFunc("/", indexHandler)
	router.HandleFunc("/auth", authHandler)
	router.HandleFunc("/register", registerHandler)
	router.HandleFunc("/registration", registrationHandler)
	router.HandleFunc("/profile/{id:[0-9]+}", profilesHandler)

	http.Handle("/", router)

	fmt.Println("Server is listening...")
	http.ListenAndServe(":80", nil)
}
