package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Book struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Price  int    `json:"price"`
}

type bookHandlers struct {
	sync.Mutex
	store map[string]Book
}

func (h *bookHandlers) books(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
		return
	case "POST":
		h.post(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("method not allowed"))
		return
	}
}

func (h *bookHandlers) get(w http.ResponseWriter, r *http.Request) {
	books := make([]Book, len(h.store))

	h.Lock()
	i := 0
	for _, book := range h.store {
		books[i] = book
		i++
	}
	h.Unlock()

	jsonBytes, err := json.Marshal(books)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *bookHandlers) getRandomCoaster(w http.ResponseWriter, r *http.Request) {
	ids := make([]string, len(h.store))
	h.Lock()
	i := 0
	for id := range h.store {
		ids[i] = id
		i++
	}
	defer h.Unlock()

	var target string
	if len(ids) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if len(ids) == 1 {
		target = ids[0]
	} else {
		rand.Seed(time.Now().UnixNano())
		target = ids[rand.Intn(len(ids))]
	}

	w.Header().Add("location", fmt.Sprintf("/books/%s", target))
	w.WriteHeader(http.StatusFound)
}

func (h *bookHandlers) getBook(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if parts[2] == "random" {
		h.getRandomCoaster(w, r)
		return
	}

	h.Lock()
	book, ok := h.store[parts[2]]
	h.Unlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	jsonBytes, err := json.Marshal(book)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *bookHandlers) post(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader((http.StatusInternalServerError))
		w.Write([]byte(err.Error()))
		return
	}

	ct := r.Header.Get("content-type")
	if ct != "application/json" {
		w.WriteHeader((http.StatusUnsupportedMediaType))
		w.Write([]byte(fmt.Sprintf("need content-type 'application/json', but got '%s'", ct)))
		return
	}

	var book Book
	err = json.Unmarshal(bodyBytes, &book)
	if err != nil {
		w.WriteHeader((http.StatusBadRequest))
		w.Write([]byte(err.Error()))
		return
	}

	book.ID = fmt.Sprintf("%d", time.Now().UnixNano())

	h.Lock()
	h.store[book.ID] = book
	defer h.Unlock()

}

func newBookHandlers() *bookHandlers {
	return &bookHandlers{
		store: map[string]Book{},
	}
}

type adminPortal struct {
	password string
}

func newAdminPortal() *adminPortal {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		panic("requred env var ADMIN_PASSWORD not set")
	}

	return &adminPortal{password: password}
}

func (a adminPortal) handler(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != "admin" || pass != a.password {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - unauthorized"))
		return
	}

	w.Write([]byte("<htmk><h1>Super secred admin portal</h1></html>"))
}

func main() {
	admin := newAdminPortal()
	bookHandlers := newBookHandlers()

	http.HandleFunc("/books", bookHandlers.books)
	http.HandleFunc("/books/", bookHandlers.getBook)
	http.HandleFunc("/admin", admin.handler)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
