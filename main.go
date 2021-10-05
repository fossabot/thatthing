package main

import (
	"fmt"
	//"io/ioutil"
	"log"
	"net/http"
	"strings"
	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
)

type app struct {
  Name  string
  Path  string
	Main  bool
	id string `gorm:"primaryKey"`
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

	db.Create(&app{Name: "Test", Path: "apps/test", Main: true, id: "test"})

	http.HandleFunc("/", root)
	http.HandleFunc("/apps/", apps)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func apps(w http.ResponseWriter, h *http.Request) {
	id := strings.Replace(h.URL.Path, "/apps/", "", 1)
	fmt.Println(id)
	var theapp app
	db.First(&theapp, "id = ?", id)
}
