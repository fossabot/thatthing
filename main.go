package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"gorm.io/gorm"
	"os"
	"gorm.io/driver/sqlite"
	"github.com/buger/jsonparser"
	"html/template"
	"os/exec"
	"plugin"
)

type app struct {
	Name  string
	Path  string
	Id string `gorm:"primaryKey"`
}

var db *gorm.DB

func main() {
	fmt.Println("Connecting to database...")
	dbb, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	db = dbb
	if err != nil {
		panic("Failed to connect to the database")
	}

	db.Table("apps").AutoMigrate(&app{})

	db.Create(&app{Name: "Test", Path: "apps/test", Id: "test"})

	http.HandleFunc("/", root)
	http.HandleFunc("/apps/", apps)

	fmt.Println("Building apps...")
	var pgs []app
	db.Find(&pgs)
	for _, v := range pgs {
		cmd := exec.Command("go", "build", "-buildmode=plugin", v.Path + "/main.go")
		cmd.Run()
		os.Rename("main.so", v.Path + "/main.so")
	}

	fmt.Println("Running...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func apps(w http.ResponseWriter, h *http.Request) {
	idd := strings.Replace(h.URL.Path, "/apps/", "", 1)
	if idd == "" {
		http.Redirect(w, h, "../", 301)
		return
	}
	id := strings.Split(idd, "/")
	var theapp app
	db.First(&theapp, "Id = ?", id[0])
	/*if er != nil {
	http.NotFound(w, h)
	return
	}*/
	dat, err := os.ReadFile(theapp.Path + "/app.json")
	if err != nil {
		panic(err)
	}
	public, _ := jsonparser.GetBoolean(dat, "public")
	if !public {
		http.NotFound(w, h)
		return
	}
	id = id[1:]
	file, e := jsonparser.GetString(dat, "stuff", "/" + strings.Join(id, "/"), "type")
	if e != nil {
		http.NotFound(w, h)
		return
	}
	if file == "file" {
		thing, _ := jsonparser.GetString(dat, "stuff", "/" + strings.Join(id, "/"), "thing")
		http.ServeFile(w, h, theapp.Path + "/" + thing)
	} else {
		theapp, _ := plugin.Open(theapp.Path + "/main.so")
		thing, _ := jsonparser.GetString(dat, "stuff", "/" + strings.Join(id, "/"), "thing")
		v, err := theapp.Lookup(thing)
		if err != nil {
			http.NotFound(w, h)
			return
		}
		v.(func(w http.ResponseWriter, h *http.Request))(w, h)
	}
}

func root(w http.ResponseWriter, h *http.Request) {
	templ := `<!DOCTYPE HTML><html><head><link href="https://fonts.googleapis.com/css2?family=Rubik:wght@400;700;800;900&display=swap" rel="stylesheet"><title>Homepage</title><style>body { font-family: 'Rubik', -apple-system, sans-serif; text-align: center; padding: 2rem; } ul, h1 { margin: .3em 0; padding: 0 } h1 { font-size: 3.2em; }</style></head><body><h1>Hi.</h1><div><div>While you're here, check some of these:</div><ul>{{range .}}<li><a href="/apps/{{.Id}}">{{.Name}}</a></li>{{else}}There's nothing here.{{end}}</ul></div></body></html>`
	t, _ := template.New("webpage").Parse(templ)
	var appss []app
	db.Find(&appss)
	t.Execute(w, appss)
}
