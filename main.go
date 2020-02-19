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
	"os"
	"social_network/config"
)

var conf = config.NewConfig()
var masterDB *sql.DB
var slavesDB []*sql.DB
var database *sql.DB
var marker = 0
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
	db := getSlaveDatabase()

	vars := mux.Vars(r)
	query := "select first_name, last_name, age, gender, city, about from profiles WHERE id = ?"
	rows, err := db.Query(query, vars["id"])

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
	db := getSlaveDatabase()
	session, err := store.Get(r, conf.SessionName)
	if err != nil {
		fmt.Println(err)
	}

	query := "select id, first_name, last_name, age, gender, city from profiles limit 1000"
	rows, err := db.Query(query)
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

func getSlaveDatabase() *sql.DB {
	slavesCount := len(slavesDB)
	if slavesCount > 0 {
		marker = (marker + 1) % len(slavesDB)
		return slavesDB[marker]
	} else {
		return database
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	db := getSlaveDatabase()

	search := r.URL.Query().Get("search") + "%"
	query := "select id, first_name, last_name, age, gender, city from profiles where first_name like ? " +
		"union select id, first_name, last_name, age, gender, city from profiles where last_name like ? order by id limit 100"
	rows, err := db.Query(query, search, search)
	defer rows.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		var profiles []ProfileModel
		for rows.Next() {
			p := ProfileModel{}
			err := rows.Scan(&p.Id, &p.FirstName, &p.LastName, &p.Age, &p.Gender, &p.City)
			if err != nil {
				fmt.Print(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			p.Gender = GENDER[p.Gender]
			profiles = append(profiles, p)
		}

		data := ProfilesModel{
			Profiles: profiles,
		}

		tmpl, _ := template.ParseFiles("templates/search.html")
		err = tmpl.Execute(w, data)
		if err != nil {
			fmt.Print(err)
		}
	}
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
	db := getSlaveDatabase()
	err := r.ParseForm()
	if err != nil {
		fmt.Println(err)
	}
	login := r.FormValue("login")
	password := r.FormValue("password")

	query := "select first_name, last_name from profiles WHERE login=? AND password=?"
	rows, err := db.Query(query, login, password)
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
	masterDB, err := sql.Open("mysql", conf.DatabaseUser+":"+conf.DatabasePassword+"@tcp("+conf.DatabaseMasterServer+")/"+conf.Database)
	if err != nil {
		log.Println(err)
	}
	masterDB.SetMaxOpenConns(conf.MaxOpenConnections)
	database = masterDB

	for _, slaveServer := range conf.DatabaseSlaveServers {
		slave, err := sql.Open("mysql", conf.DatabaseUser+":"+conf.DatabasePassword+"@tcp("+slaveServer+")/"+conf.Database)
		defer slave.Close()
		if err != nil {
			log.Println(err)
		}
		slave.SetMaxOpenConns(conf.MaxOpenConnections)
		slavesDB = append(slavesDB, slave)
	}
	defer masterDB.Close()

	router := mux.NewRouter()
	router.HandleFunc("/", indexHandler)
	router.HandleFunc("/auth", authHandler)
	router.HandleFunc("/register", registerHandler)
	router.HandleFunc("/search", searchHandler)
	router.HandleFunc("/registration", registrationHandler)
	router.HandleFunc("/profile/{id:[0-9]+}", profilesHandler)

	http.Handle("/", router)

	fmt.Println("Server is listening...")
	port := os.Getenv("PORT")
	err = http.ListenAndServe(":"+port, nil)
}
