package main

import (
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"html/template"
	"math/rand"
	"net/http"
	"path"
	"strings"
	"time"
	"strconv"
)

type blog struct {
	Title string
	Cont  string
	Publ  string
	Id    string `gorm:"primaryKey"`
}

var db *gorm.DB

func Main() {
	dbb, _ := gorm.Open(sqlite.Open("blog.db"), &gorm.Config{})
	db = dbb
	db.AutoMigrate(&blog{})
}

func RunMe(w http.ResponseWriter, h *http.Request) {
	var sh []blog
	db.Find(&sh)
	templ := `<!DOCTYPE HTML><html><head><title>Blog</title><link href="/style.css" rel="stylesheet"></head><body><h1>Blog</h1><div class="a"><ul>{{range .}}<li><a href="/apps/blog/blogs/{{.Id}}">{{.Title}}</a></li>{{else}}No blogs{{end}}</ul></div></body></html>`
	t, _ := template.New("webpage").Parse(templ)
	t.Execute(w, sh)
	return
}

func See(w http.ResponseWriter, h *http.Request) {
	var t blog
	e := db.First(&t, "id = ?", path.Base(h.URL.Path))
	if e.Error != nil {
		http.NotFound(w, h)
		return
	}
	i, _ := strconv.ParseInt(t.Publ, 10, 64)
	tim := time.Unix(i, 0).Local().Format(time.UnixDate)
	t.Publ = tim
	templ := `<!DOCTYPE HTML><html><head><title>{{.Title}} - Blog</title><link href="/style.css" rel="stylesheet"></head><body><h1>{{.Title}}</h1><div>{{.Publ}}</div><div class="a" style="text-align: left">{{.Cont}}</div></body></html>`
	templ = strings.Replace(templ, "{{.Cont}}", t.Cont, 1)
	te, _ := template.New("webpage").Parse(templ)
	te.Execute(w, t)
}

func Setting(w http.ResponseWriter, h *http.Request) {
	var k []blog
	db.Find(&k)
	templ := `<!DOCTYPE HTML><html><head><title>Blog</title><link href="/style.css" rel="stylesheet"></head><body><h1>Blog</h1><div class="a"><a href="/settings/apps/blog/new"><button>New</button></a><hr /><ul>{{range .}}<li><a href="/apps/blog/blogs/{{.Id}}">{{.Title}} ({{.Id}})</a> − <a href="/settings/apps/blog/del/{{.Id}}">Delete</a></li>{{else}}No blogs{{end}}</ul></div></body></html>`
	t, _ := template.New("webpage").Parse(templ)
	t.Execute(w, k)
}

func Nwblg(w http.ResponseWriter, h *http.Request) {
	if h.Method == "POST" {
		h.ParseForm()
		timee := time.Now().Unix()
		rand.Seed(time.Now().UnixNano())
		id := fmt.Sprint(rand.Intn(999999999999999))
		ns := blackfriday.MarkdownCommon([]byte(h.Form["content"][0]))
		cont := bluemonday.UGCPolicy().SanitizeBytes(ns)
		db.Create(&blog{
			Title: h.Form["title"][0],
			Cont:  string(cont),
			Publ:  fmt.Sprint(timee),
			Id:    id,
		})
		http.Redirect(w, h, ".", 303)
	} else {
		http.ServeFile(w, h, "apps/blog/new.html")
	}
}

func Del(w http.ResponseWriter, h *http.Request) {
	var blg blog
	e := db.First(&blg, "Id = ?", path.Base(h.URL.Path))
	if e.Error != nil {
		fmt.Fprintf(w, "Failed")
		return
	}
	db.Delete(&blog{}, "Id = ?", path.Base(h.URL.Path))
	http.Redirect(w, h, "..", 303)
}
