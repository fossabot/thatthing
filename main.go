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
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"syscall"
	"bufio"
	//"github.com/golang-jwt/jwt/v4"
	"crypto/rand"
	"math/big"
)

type app struct {
	Name  string
	Path  string
	Public bool
	Id string `gorm:"primaryKey"`
}

type kv struct {
	Key string `gorm:"primaryKey"`
	Value string
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
	db.Table("config").AutoMigrate(&kv{})

	if check("pass") {
		fmt.Print("Insert a password: ")
		pass, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			panic(err)
		}
		passss, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		db.Table("config").Create(&kv{"pass", string(passss)})
		fmt.Println("")
	}

	if check("name") {
		fmt.Print("Insert name: ")
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		db.Table("config").Create(&kv{"name", strings.TrimSpace(text)})
	}

	if check("jwtthingidk") {
		s, _ := rand.Int(rand.Reader, big.NewInt(int64(999999999999999)))
		db.Table("config").Create(&kv{"jwtthingidk", s.String()})
	}

	db.Create(&app{Name: "Test", Path: "apps/test", Id: "test", Public: true})

	http.HandleFunc("/", root)
	http.HandleFunc("/apps/", apps)
	http.HandleFunc("/settings/", settings)
	http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "style.css")
	})

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
	thatapp, _ := plugin.Open(theapp.Path + "/main.so")
	thing, _ := jsonparser.GetString(dat, "func")
	v, err := thatapp.Lookup(thing)
	if err != nil {
		http.NotFound(w, h)
		return
	}
	v.(func(w http.ResponseWriter, h *http.Request))(w, h)
}

func root(w http.ResponseWriter, h *http.Request) {
	var name kv
	db.Table("config").First(&name, "key = ?", "name")
	templ := `<!DOCTYPE HTML><html><head><link href="https://fonts.googleapis.com/css2?family=Rubik:wght@400;700;800;900&display=swap" rel="stylesheet"><title>{{.Name.Value}}'s site</title><style>body { font-family: 'Rubik', -apple-system, sans-serif; text-align: center; padding: 2rem; } ul, h1 { margin: .3em 0; padding: 0 } h1 { font-size: 3.2em; }</style></head><body><h1>Hi, I'm {{.Name.Value}}.</h1><div>{{ $length := len .Apps }}{{if gt $length 0}}<div>While you're here, check some of these:</div>{{end}}<ul>{{range .Apps}}{{if .Public}}<li><a href="/apps/{{.Id}}">{{.Name}}</a></li>{{end}}{{else}}There's nothing here.{{end}}</ul></div></body></html>`
	t, _ := template.New("webpage").Parse(templ)
	var appss []app
	db.Find(&appss)
	var filtered []app
	for _, v := range appss {
		thedata, err := os.ReadFile(v.Path + "/app.json")
		if err != nil {
			panic(err)
		}
		if thing, _ := jsonparser.GetBoolean(thedata, "public"); thing {
			filtered = append(filtered, v)
		}
	}
	things := struct{
		Apps []app
		Name kv
	}{ Apps: filtered, Name: name }
	t.Execute(w, things)
}

func check(name string) bool {
	var g []kv
	res := db.Table("config").Find(&g, "key = ?", name)
	return res.RowsAffected == 0
}

func settings(w http.ResponseWriter, h *http.Request) {
	cooki, err := h.Cookie("login")
	if err != nil {
		templ := `<!DOCTYPE HTML><html><head><title>Login</title><link rel="stylesheet" href="../style.css"></head><body><h1>Hello, {{.}}</h1><form action="settings" method="POST"><label for="p">Password</label><input id="p" type="password" name="pass"><div><input type="submit" value="Sign In"></div></form></body></html>`
		t, _ := template.New("webpage").Parse(templ)
		var name kv
		db.Table("config").First(&name, "key = ?", "name")
		t.Execute(w, name.Value)
		return
	}
	cookie := cooki.Value
	var jwtth kv
	db.Table("config").First(&jwtth, "key = ?", "jwtthingidk")
	fmt.Println(cookie)
	/*token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtth.Value), nil
	})
	fmt.Println(token)*/
}
