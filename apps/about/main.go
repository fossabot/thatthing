package main

import (
  "net/http"
  "text/template"
  "os"
  "fmt"
  "strings"
  "github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

var AppId string
var AppPath string
var AppName string

func Main() {
  _, err := os.ReadFile("data/aboutapp_"+AppId+".md")
  if err != nil {
    e := os.WriteFile("data/aboutapp_"+AppId+".md", []byte(""), 0666)
    if e != nil {
      fmt.Println("Cannot create aboutapp_*.md")
      return
    }
  }
}

func Mn(w http.ResponseWriter, h *http.Request) {
  data, _ := os.ReadFile("data/aboutapp_"+AppId+".md")
  ns := blackfriday.MarkdownCommon(data)
  cont := bluemonday.UGCPolicy().SanitizeBytes(ns)
  templ := `<!DOCTYPE html>
  <html lang="en" dir="ltr">
    <head>
      <meta charset="utf-8">
      <title>{{.B}}</title>
      <link rel="stylesheet" href="/style.css">
    </head>
    <body>
      <h1>{{.B}}</h1>
      <div class="a">{{if lt (len .A) 1}}Nothing here 🤷{{else}}{{.A}}{{end}}</div>
    </body>
  </html>
`
  strings.Replace(templ, "{{.}}", string(data), 1)
  f, _ := template.New("main").Parse(templ)
  f.Execute(w, struct{
    A string
    B string}{
    A: string(cont),
    B: AppName,
  })
}

func Conf(w http.ResponseWriter, h *http.Request) {
  if !strings.HasSuffix(h.URL.Path, "/") {
    http.Redirect(w, h, "/settings/apps/" + AppId + "/", 303)
    return
  }
  data, _ := os.ReadFile("data/aboutapp_"+AppId+".md")
    if h.Method == "POST" {
      h.ParseForm()
      e := os.WriteFile("data/aboutapp_"+AppId+".md", []byte(h.Form["main"][0]), 0666)
      if e != nil {
        fmt.Fprintf(w, "Failed to save")
        return
      }
      http.Redirect(w, h, ".", 303)
    } else {
      f, _ := template.New("conf").Parse(`<!DOCTYPE html>
<html lang="en" dir="ltr">
  <head>
    <meta charset="utf-8">
    <link rel="stylesheet" href="/style.css">
    <title>About</title>
  </head>
  <body>
    <h1>About</h1>
    <form action="./" method="POST">
      <label for="main">About page contents (Markdown is supported!)</label>
      <textarea name="main" id="main">{{.}}</textarea>
      <div class="">
        <input type="submit" value="Save">
      </div>
    </form>
  </body>
</html>
`)
      f.Execute(w, string(data))
    }
}
