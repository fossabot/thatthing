package main

import (
  "net/http"
  "fmt"
)

func RunMe(w http.ResponseWriter, h *http.Request) {
  fmt.Fprintf(w, "hello world")
}
