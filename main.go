package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"html/template"
	"log"
	"math/big"
	"net/http"
	"path/filepath"
	"os"
	"os/exec"
	"sort"
	"path"
	"plugin"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type app struct {
	Name   string
	Path   string
	Public bool
	Id     string `gorm:"primaryKey"`
}

type kv struct {
	Key   string `gorm:"primaryKey"`
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
		passss, _ := bcrypt.GenerateFromPassword(pass, bcrypt.DefaultCost)
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
		s, _ := rand.Int(rand.Reader, big.NewInt(int64(999999999999999999)))
		db.Table("config").Create(&kv{"jwtthingidk", s.String()})
	}

	db.Table("config").Create(&kv{"desc", ""})
	db.Table("config").Create(&kv{"img", ""})

	http.HandleFunc("/", root)
	http.HandleFunc("/apps/", apps)
	http.HandleFunc("/settings/", settings)
	http.HandleFunc("/settings/logout/", func(w http.ResponseWriter, h *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "login", Value: "", MaxAge: -1, Path: "/"})
		http.Redirect(w, h, "/settings", 303)
	})
	http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "style.css")
	})
	http.HandleFunc("/settings/uninstall/", uninst)
	http.HandleFunc("/settings/install/", instal)
	http.HandleFunc("/settings/apps/", appconf)

	fmt.Println("Building apps...")
	var pgs []app
	db.Find(&pgs)
	exec.Command("go", "mod", "tidy").Run()
	for _, v := range pgs {
		cmd := exec.Command("go", "build", "-buildmode=plugin", v.Path+"/main.go")
		cmd.Run()
		os.Rename("main.so", v.Path+"/main.so")
		thatapp, _ := plugin.Open(v.Path + "/main.so")
		id, err := thatapp.Lookup("AppId")
		if err == nil {
		 	*id.(*string) = v.Id
		}
		pt, err := thatapp.Lookup("AppPath")
		if err == nil {
			*pt.(*string) = v.Path
		}
		nm, err := thatapp.Lookup("AppName")
		if err == nil {
			*nm.(*string) = v.Name
		}
		ve, err := thatapp.Lookup("Main")
		if err == nil {
			ve.(func())()
		}
	}

	fmt.Println("Running...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func instal(w http.ResponseWriter, h *http.Request) {
	if !CheckLogin(h) {
		http.Redirect(w, h, "/settings", 303)
		return
	}
	if h.Method == "POST" {
		rgx := regexp.MustCompile("^[a-zA-Z0-9]*$")
		h.ParseForm()
		pth := h.Form["path"][0]
		var find app
		e := db.First(&find, "Id = ?", h.Form["id"][0])
		if e.Error == nil {
			fmt.Fprintf(w, "Failed, app is already installed (ID)")
			return
		}
		e = db.First(&find, "Path = ?", h.Form["path"][0])
		if e.Error == nil {
			fmt.Fprintf(w, "Failed, app is already installed (Path)")
			return
		}
		if !rgx.MatchString(h.Form["id"][0]) {
			fmt.Fprintf(w, "Failed, ID must be alphanumeric")
			return
		}
		if !testread(pth + "/app.json") {
			fmt.Fprintf(w, "Failed, app.json does not exist")
			return
		}
		if !testread(pth + "/main.go") {
			fmt.Fprintf(w, "Failed, main.go does not exist")
			return
		}
		db.Create(&app{
			Name: h.Form["name"][0],
			Path: h.Form["path"][0],
			Public: true,
			Id: h.Form["id"][0],
		})
		http.Redirect(w, h, "/settings/apps", 303)
	} else {
		pth, _ := filepath.Abs("./")
		templ, _ := template.ParseFiles("install.html")
		templ.ExecuteTemplate(w, "install.html", pth + "/")
	}
}

func testread(path string) bool {
	_, err := os.ReadFile(path)
	return err == nil
}

func appconf(w http.ResponseWriter, h *http.Request) {
	if !CheckLogin(h) {
		http.Redirect(w, h, "/settings", 303)
		return
	}
	if h.URL.Path == "/settings/apps/" {
		var ap []app
		db.Find(&ap)
		templ := `<!DOCTYPE HTML><html><head><title>Apps</title><link rel="stylesheet" href="/style.css"></head><body><h1>Apps</h1><div class="a"><a href="/settings/install"><button>Install</button></a><hr /><ul>{{range .}}<li><a href="/settings/apps/{{.Id}}">{{.Name}}</a> − <a href="/settings/uninstall/{{.Id}}">Uninstall</a></li>{{else}}No apps.{{end}}</ul></div></body></html>`
		v, _ := template.New("webpage").Parse(templ)
		v.Execute(w, ap)
		return
	}
	ap := strings.Replace(h.URL.Path, "/settings/apps/", "", 1)
	id := strings.Split(ap, "/")
	var theapp app
	db.First(&theapp, "Id = ?", id[0])
	data, err := os.ReadFile(theapp.Path + "/app.json")
	if err != nil {
		panic(err)
	}
	id = id[1:]
	thatapp, _ := plugin.Open(theapp.Path + "/main.so")
	found := false
	var thing string
	jsonparser.ObjectEach(data, func(k []byte, v []byte, dataType jsonparser.ValueType, o int) error {
		if !found {
			if strings.HasPrefix("/" + strings.Join(id, "/"), string(k)) {
				found = true
				thing = string(v)
			}
		}

		return nil
	}, "conf")
	v, err := thatapp.Lookup(thing)
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, h)
		return
	}
	v.(func(w http.ResponseWriter, h *http.Request))(w, h)
}

func apps(w http.ResponseWriter, h *http.Request) {
	idd := strings.Replace(h.URL.Path, "/apps/", "", 1)
	if idd == "" {
		http.Redirect(w, h, "../", 301)
		return
	}
	id := strings.Split(idd, "/")
	var theapp app
	e := db.First(&theapp, "Id = ?", id[0])
	if e.Error != nil {
		http.NotFound(w, h)
		return
	}
	dat, err := os.ReadFile(theapp.Path + "/app.json")
	if err != nil {
		panic(err)
	}
	public, _ := jsonparser.GetBoolean(dat, "public")
	if !public {
		if !CheckLogin(h) {
			http.NotFound(w, h)
			return
		}
	}
	id = id[1:]
	thatapp, _ := plugin.Open(theapp.Path + "/main.so")
	found := false
	var thing string
	jsonparser.ObjectEach(dat, func(k []byte, v []byte, dataType jsonparser.ValueType, o int) error {
		if !found {
			if strings.HasPrefix("/" + strings.Join(id, "/"), string(k)) {
				found = true
				thing = string(v)
			}
		}

		return nil
	}, "func")

	v, err := thatapp.Lookup(thing)
	if err != nil {
		fmt.Println(err)
		http.NotFound(w, h)
		return
	}
	v.(func(w http.ResponseWriter, h *http.Request))(w, h)
}

func root(w http.ResponseWriter, h *http.Request) {
	d := make(map[string]string)
	var thingss []kv
	db.Table("config").Find(&thingss)
	for _, v := range thingss {
		d[v.Key] = v.Value
	}
	templ := `<!DOCTYPE HTML><html><head><link href="style.css" rel="stylesheet"><title>{{.Data.name}}'s site</title></head><body><div class="a">{{if gt (len .Data.img) 0}}<div><img src="{{.Data.img}}" class="pfp"></div>{{end}}<h1>Hi, I'm {{.Data.name}}.</h1><div>{{.Data.desc}}</div></div><div class="a">{{ $length := len .Apps }}{{if gt $length 0}}<div>While you're here, check some of these:</div>{{end}}<ul>{{range .Apps}}{{if .Public}}<li><a href="/apps/{{.Id}}">{{.Name}}</a></li>{{end}}{{else}}There's nothing here.{{end}}</ul></div><div class="tt">ThatThing</div></body></html>`
	t, _ := template.New("webpage").Parse(templ)
	var appss []app
	db.Find(&appss)
	var filtered []app
	if !CheckLogin(h) {
		for _, v := range appss {
			thedata, err := os.ReadFile(v.Path + "/app.json")
			if err != nil {
				panic(err)
			}
			if thing, _ := jsonparser.GetBoolean(thedata, "public"); thing {
				filtered = append(filtered, v)
			}
		}
	} else {
		filtered = appss
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	things := struct {
		Apps []app
		Data map[string]string
	}{Apps: filtered, Data: d}
	t.Execute(w, things)
}

func check(name string) bool {
	var g []kv
	res := db.Table("config").Find(&g, "key = ?", name)
	return res.RowsAffected == 0
}

func CheckLogin(h *http.Request) bool {
	var jwtth kv
	db.Table("config").First(&jwtth, "key = ?", "jwtthingidk")
	cook, er := h.Cookie("login")
	if er != nil {
		return false
	}
	token, err := jwt.Parse(cook.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(jwtth.Value), nil
	})

	if err != nil {
		return false
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		aft, _ := claims["now"].(int64)
		if time.Unix(aft, 0).After(time.Now()) {
			return false
		}
		return claims["login"].(bool)
	} else {
		fmt.Println(err)
		return false
	}
}

func settings(w http.ResponseWriter, h *http.Request) {
	_, err := h.Cookie("login")
	templ := `<!DOCTYPE HTML><html><head><title>Login</title><link rel="stylesheet" href="../style.css"></head><body>{{if gt (len .img.Value) 0}}<div><img src="{{.img.Value}}" class="pfp"></div>{{end}}<h1>Welcome back, {{.name.Value}}</h1><form action="." method="POST"><label for="p">Password</label><input id="p" type="password" name="pass"><div><input type="submit" value="Sign In"></div></form><div class="tt">ThatThing</div></body></html>`
	t, _ := template.New("webpage").Parse(templ)
	var name kv
	db.Table("config").First(&name, "key = ?", "name")
	var img kv
	db.Table("config").First(&img, "key = ?", "img")
	d := map[string]kv{"name": name, "img": img}
	var jwtth kv
	db.Table("config").First(&jwtth, "key = ?", "jwtthingidk")
	if err != nil {
		if h.Method == "GET" {
			t.Execute(w, d)
		} else if h.Method == "POST" {
			h.ParseForm()
			var thepass kv
			db.Table("config").First(&thepass, "key = ?", "pass")
			pass := bcrypt.CompareHashAndPassword([]byte(thepass.Value), []byte(h.Form["pass"][0]))
			if pass == nil {
				dur, _ := time.ParseDuration("168h")
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"login": true,
					"now":   time.Now().Add(dur).Unix()})
				realtoken, err := token.SignedString([]byte(jwtth.Value))
				if err != nil {
					fmt.Println("Failed to login")
					fmt.Println(err)
					t.Execute(w, d)
					return
				}
				http.SetCookie(w, &http.Cookie{Name: "login", Value: realtoken, Expires: time.Now().Add(dur), Path: "/"})
				http.Redirect(w, h, ".", 303)
			} else {
				t.Execute(w, d)
			}
		}
		return
	} else {
		if CheckLogin(h) {
			if h.Method == "POST" {
				h.ParseForm()
				for k, v := range h.Form {
					var upd kv
					db.Table("config").First(&upd, "key = ?", k)
					upd.Value = v[0]
					db.Table("config").Save(&upd)
				}
				http.Redirect(w, h, ".", 303)
			} else {
				d := make(map[string]string)
				var things []kv
				db.Table("config").Find(&things)
				for _, v := range things {
					d[v.Key] = v.Value
				}
				page := `<!DOCTYPE HTML><html><head><link rel="stylesheet" href="../style.css"><title>Settings</title></head><body><h1>Settings</h1><form action="." method="POST"><label for="f1">Name</label><input required id="f1" type="text" name="name" value="{{.name}}"><br /><label for="desc">Description</label><textarea id="desc" name="desc">{{.desc}}</textarea><br /><label for="img">URL of image (square is recommended)</label><input type="url" id="img" name="img" value="{{.img}}"><div><input type="submit" value="Done"></div></form><div class="a"><a href="/settings/apps">See apps >>></a></div><div><a href="logout"><button style="margin: 1em">Log out</button></a></div><div class="tt">ThatThing</div></body></html>`
				p, _ := template.New("webpage").Parse(page)
				p.Execute(w, d)
			}
		} else {
			http.SetCookie(w, &http.Cookie{Name: "login", Value: "", MaxAge: -1, Path: "/"})
			http.Redirect(w, h, ".", 303)
		}
	}
}

func uninst(w http.ResponseWriter, h *http.Request) {
	if CheckLogin(h) {
		e := db.Delete(&app{}, "Id = ?", path.Base(h.URL.Path))
		if e.Error != nil {
			http.NotFound(w, h)
			return
		}
		http.Redirect(w, h, "/settings/apps", 303)
	} else {
		http.Redirect(w, h, "/settings", 303)
	}
}
