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
)

type app struct {
  Name  string
  Path  string
	Main  bool
	Id string `gorm:"primaryKey"`
}

func root(w http.ResponseWriter, h *http.Request) {
	fmt.Fprintf(w, "test")
}

var db *gorm.DB

func main() {
	dbb, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	db = dbb
  if err != nil {
    panic("Failed to connect to the database")
  }

	db.Table("apps").AutoMigrate(&app{})

	db.Create(&app{Name: "Test", Path: "apps/test", Main: true, Id: "test"})

	http.HandleFunc("/", root)
	http.HandleFunc("/apps/", apps)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func apps(w http.ResponseWriter, h *http.Request) {
	idd := strings.Replace(h.URL.Path, "/apps/", "", 1)
	id := strings.Split(idd, "/")
	var theapp app
	db.First(&theapp, "Id = ?", id[0])
	dat, err := os.ReadFile(theapp.Path + "/app.json")
  if err != nil {
		panic(err)
	}
	id = id[1:]
	file, e := jsonparser.GetString(dat, "stuff", "/" + strings.Join(id, "/"))
	if e != nil {
		fmt.Println("Not found")
	}
	http.ServeFile(w, h, theapp.Path + "/" + file)
}
