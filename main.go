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
	"bytes"
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
	Name         string        `json:"name"`
	FileName     string        `json:"filename"`
	Created      time.Time     `json:"created"`
	ExpiresAfter time.Duration `json:"expires"`
}

// Expired return true if file expired
func (w *WpasteFile) Expired() bool {
	return w.Created.Add(w.ExpiresAfter).Before(time.Now())
}

// Data from file
func (w *WpasteFile) Data() ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(FilesDir, w.FileName))
}

// OpenWpasteByName return Wpaste if exist else nil
func OpenWpasteByName(name string, file *WpasteFile) func(tx *bbolt.Tx) error {
	return func(tx *bbolt.Tx) error {
		files := tx.Bucket([]byte("files"))
		cur := files.Cursor()
		for k, v := cur.First(); k != nil; k, v = cur.Next() {
			var f WpasteFile
			if err := json.Unmarshal(v, &f); err != nil {
				return err
			}
			if f.Name == name {
				*file = f
				return nil
			}
		}
		return nil
	}
}

// CreateWpaste create in database
func CreateWpaste(wpaste *WpasteFile) func(tx *bbolt.Tx) error {
	return func(tx *bbolt.Tx) error {
		files := tx.Bucket([]byte("files"))

		f, err := json.Marshal(wpaste)
		if err != nil {
			return err
		}
		id, _ := files.NextSequence()

		return files.Put([]byte(strconv.FormatUint(id, 10)), f)
	}
}

// CheckUnique return true to *unique if value unique
func CheckUnique(field string, value interface{}, unique *bool) func(tx *bbolt.Tx) error {
	*unique = true
	return func(tx *bbolt.Tx) error {
		files := tx.Bucket([]byte("files"))
		cur := files.Cursor()

		for k, v := cur.First(); k != nil; k, v = cur.Next() {
			var f WpasteFile
			if err := json.Unmarshal(v, &f); err != nil {
				return err
			}
			field := reflect.ValueOf(f).FieldByName(field)
			if field.String() == value {
				*unique = false
			}
		}
		return nil
	}
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

// Help redirect to github
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
	if r.ContentLength > 10<<20 {
		HTTPError(w, http.StatusRequestEntityTooLarge, "413 - Max content size is 10MiB")
		return
	} else if len(r.FormValue("f")) == 0 {
		HTTPError(w, http.StatusBadRequest, `400 - "f" field required`)
	}

	wpaste := WpasteFile{Created: time.Now()}

	data := r.FormValue("f")
	file := bytes.NewReader([]byte(data))

	var unique bool
	var filename string

	for !unique {
		filename = RandomString(3)
		db.View(CheckUnique("FileName", filename, &unique))
	}

	name := r.FormValue("name")
	if len(name) == 0 {
		name = filename
	} else {
		db.View(CheckUnique("Name", name, &unique))
		if !unique {
			HTTPError(w, http.StatusConflict, "409 - This filename already taken!")
			return
		}
	}

	wpaste.Name = name
	wpaste.FileName = filename

	servFile, err := os.OpenFile(filepath.Join(FilesDir, filename), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)

	if err != nil {
		HTTPServerError(w)
		return
	}

	defer servFile.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		HTTPServerError(w)
		return
	}

	servFile.Write(fileBytes)
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

	if err = db.Update(CreateWpaste(&wpaste)); err != nil {
		HTTPServerError(w)
		return
	}

	w.Write([]byte(name))
}

// SendFile respond file by it ID
func SendFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ID := vars["id"]
	var file WpasteFile
	if err := db.View(OpenWpasteByName(ID, &file)); err != nil {
		HTTPServerError(w)
		return
	}
	if len(file.FileName) == 0 {
		HTTPError(w, http.StatusNotFound, "404 - File not found")
		return
	}
	if file.Expired() {
		HTTPError(w, http.StatusGone, "410 - File is no longer available")
		return
	}
	if data, err := file.Data(); err == nil {
		w.Write(data)
	} else {
		HTTPServerError(w)
		return
	}
}

// WpasteRouter make router with all needed Handlers
func WpasteRouter() *mux.Router {
	Router := mux.NewRouter().StrictSlash(true)

	Router.HandleFunc("/", Help).Methods("GET")
	Router.HandleFunc("/", UploadFile).Methods("POST")

	Router.HandleFunc("/{id}", SendFile)
	return Router
}

// Working directory
var (
	FilesDir string
	BaseDir  string
)

// SetDirectories specify working directories
func SetDirectories() {
	var err error
	BaseDir, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	FilesDir = filepath.Join(BaseDir, "files")
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

// Install prepare to start:
// 1. Set working dirs
// 2. Make directory for user files
// 3. Set random seed
func Install() {
	SetDirectories()
	os.Mkdir(FilesDir, 0766)
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
