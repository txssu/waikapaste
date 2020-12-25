// wpaste - easy code sharing
// Copyright (C) 2020  Evgeniy Rybin
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gorilla/mux"
	"go.etcd.io/bbolt"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomString creates a random string with the charset
// that contains all letters and digits
func RandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// WpasteFile is data about file
type WpasteFile struct {
	Name           string        `json:"name"`
	Data           string        `json:"data"`
	Created        time.Time     `json:"created"`
	AccessPassword string        `json:"acesspass"`
	EditPassword   string        `json:"editpass"`
	Edited         time.Time     `json:"edited"`
	ExpiresAfter   time.Duration `json:"expires"`
	Deleted        bool          `json:"deleted"`
}

// Expired return true if file expired
func (w *WpasteFile) Expired() bool {
	return w.Created.Add(w.ExpiresAfter).Before(time.Now())
}

// OpenWpasteByName return Wpaste if exist else nil
func OpenWpasteByName(name string) (file *WpasteFile, err error) {
	tx, err := db.Begin(false)
	if err != nil {
		return
	}
	defer tx.Rollback()
	files := tx.Bucket([]byte("files"))
	for id := files.Sequence(); id > 0; id-- {
		v := files.Get([]byte(strconv.FormatUint(id, 10)))
		var f WpasteFile
		if err = json.Unmarshal(v, &f); err != nil {
			return
		}
		if f.Name == name {
			if f.Deleted {
				return
			}
			file = &f
			return
		}
	}
	return
}

// CreateWpaste create in database
func CreateWpaste(wpaste *WpasteFile) (err error) {
	tx, err := db.Begin(true)
	if err != nil {
		return
	}
	defer tx.Rollback()
	files := tx.Bucket([]byte("files"))

	f, err := json.Marshal(wpaste)
	if err != nil {
		return err
	}
	id, _ := files.NextSequence()

	files.Put([]byte(strconv.FormatUint(id, 10)), f)
	return tx.Commit()
}

// CheckUnique return true to *unique if value unique
func CheckUnique(field string, value interface{}) (unique bool) {
	tx, err := db.Begin(false)
	if err != nil {
		return
	}
	defer tx.Rollback()

	unique = true

	files := tx.Bucket([]byte("files"))
	cur := files.Cursor()

	for k, v := cur.First(); k != nil; k, v = cur.Next() {
		var f WpasteFile
		if err := json.Unmarshal(v, &f); err != nil {
			continue
		}
		field := reflect.ValueOf(f).FieldByName(field)
		if field.String() == value {
			unique = false
			break
		}
	}

	return
}

// HTTPError write status code to header and description to body
func HTTPError(w http.ResponseWriter, code int, description string) {
	w.WriteHeader(code)
	w.Write([]byte(description))
}

// HTTPServerError id equivalent for HTTPError which write http.StatusInternalServerError
func HTTPServerError(w http.ResponseWriter) {
	HTTPError(w, http.StatusInternalServerError, "500 - Something bad happened")
}

// Help return README.md
func Help(w http.ResponseWriter, r *http.Request) {
	file, err := ioutil.ReadFile(filepath.Join(BaseDir, "README.md"))
	if err != nil {
		log.Println(err)
		HTTPServerError(w)
		return
	}
	w.Write([]byte(markdown.ToHTML(file, nil, nil)))
}

// UploadFile save file and response it ID
func UploadFile(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > 2<<20 {
		HTTPError(w, http.StatusRequestEntityTooLarge, "413 - Max content size is 10MiB")
		return
	} else if len(r.FormValue("f")) == 0 {
		HTTPError(w, http.StatusBadRequest, `400 - "f" field required`)
		return
	}

	wpaste := &WpasteFile{Created: time.Now()}

	wpaste.Data = r.FormValue("f")

	name := r.FormValue("name")

	if len(name) == 0 {
		name = RandomString(3)
		for !CheckUnique("Name", name) {
			name = RandomString(3)
		}
	} else {
		if !CheckUnique("Name", name) {
			HTTPError(w, http.StatusConflict, "409 - This filename already taken!")
			return
		}
	}
	wpaste.Name = name

	e := r.FormValue("e")
	var expires time.Duration
	if len(e) != 0 {
		addTime, err := strconv.Atoi(e)
		if err != nil {
			HTTPError(w, http.StatusUnprocessableEntity, "422 - Invalid time format")
			return
		} else if addTime < 0 {
			HTTPError(w, http.StatusBadRequest, "400 - Time shold be positive")
			return
		}
		expires = time.Duration(addTime) * time.Second
	} else {
		expires = time.Duration(30*24) * time.Hour
	}

	wpaste.ExpiresAfter = expires

	wpaste.AccessPassword = r.FormValue("ap")
	wpaste.EditPassword = r.FormValue("ep")

	if err := CreateWpaste(wpaste); err != nil {
		HTTPServerError(w)
		return
	}

	w.Write([]byte(name))
}

// SendFile respond file by it ID
func SendFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ID := vars["id"]
	file, err := OpenWpasteByName(ID)
	if err != nil {
		HTTPServerError(w)
		return
	}
	r.ParseForm()
	if file == nil {
		HTTPError(w, http.StatusNotFound, "404 - File not found")
		return
	} else if file.Expired() {
		HTTPError(w, http.StatusGone, "410 - File is no longer available")
		return
	} else if len(file.AccessPassword) != 0 && file.AccessPassword != r.Form.Get("ap") {
		HTTPError(w, http.StatusUnauthorized, "401 - Invalid password")
		return
	}

	w.Write([]byte((*file).Data))
}

// EditFile put new file
func EditFile(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > 10<<20 {
		HTTPError(w, http.StatusRequestEntityTooLarge, "413 - Max content size is 10MiB")
		return
	} else if len(r.FormValue("f")) == 0 {
		HTTPError(w, http.StatusBadRequest, `400 - "f" field required`)
	}
	vars := mux.Vars(r)
	ID := vars["id"]

	file, err := OpenWpasteByName(ID)
	if err != nil {
		HTTPServerError(w)
		return
	}

	if file == nil {
		HTTPError(w, http.StatusNotFound, "404 - File not found")
		return
	} else if file.Expired() {
		HTTPError(w, http.StatusGone, "410 - File is no longer available")
		return
	} else if len(file.EditPassword) == 0 || file.EditPassword != r.FormValue("ep") {
		HTTPError(w, http.StatusUnauthorized, "401 - Invalid password")
		return
	}

	file.Data = r.FormValue("f")
	file.Edited = time.Now()

	if err := CreateWpaste(file); err != nil {
		HTTPServerError(w)
		return
	}
}

// DeleteFile set deleted flag to true
func DeleteFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ID := vars["id"]

	file, err := OpenWpasteByName(ID)
	if err != nil {
		HTTPServerError(w)
		return
	}

	r.ParseForm()
	if file == nil {
		HTTPError(w, http.StatusNotFound, "404 - File not found")
		return
	} else if len(file.EditPassword) == 0 || file.EditPassword != r.FormValue("ep") {
		HTTPError(w, http.StatusUnauthorized, "401 - Invalid password")
		return
	}

	file.Deleted = true

	if err := CreateWpaste(file); err != nil {
		HTTPServerError(w)
		return
	}
}

// WpasteRouter make router with all needed Handlers
func WpasteRouter() *mux.Router {
	Router := mux.NewRouter().StrictSlash(true)

	Router.HandleFunc("/", Help).Methods("GET")
	Router.HandleFunc("/", UploadFile).Methods("POST")

	Router.HandleFunc("/{id}", SendFile).Methods("GET")
	Router.HandleFunc("/{id}", EditFile).Methods("PUT")
	Router.HandleFunc("/{id}", DeleteFile).Methods("DELETE")
	return Router
}

// Working directory
var (
	BaseDir string
)

// SetDirectories specify working directories
func SetDirectories() {
	var err error
	BaseDir, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
}

var db *bbolt.DB

func initDB() {
	var err error
	db, err = bbolt.Open(filepath.Join(BaseDir, "data.db"), 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("files"))
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
}

// Install prepare to start
func Install() {
	SetDirectories()
	rand.Seed(time.Now().UTC().UnixNano())
	initDB()
}

// Close all connections
func Close() {
	db.Close()
}

func main() {
	Install()
	defer Close()
	http.ListenAndServe(":9990", WpasteRouter())
}
