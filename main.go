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
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// Help return message about
func Help(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://github.com/waika28/wpaste.cyou", http.StatusSeeOther)
}

// UploadFile to server and return link to it
func UploadFile(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	data := r.FormValue("f")
	file := bytes.NewReader([]byte(data))

	if file.Len() > 5*10e6 {
		fmt.Fprintln(w, "File is too large")
		return
	}

	tempFile, err := ioutil.TempFile(FilesDir(), "ufile-*")
	if err != nil {
		fmt.Println(err)
	}
	defer tempFile.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}

	tempFile.Write(fileBytes)
	fmt.Fprintln(w, "https://wpaste.cyou/"+strings.TrimPrefix(filepath.Base(tempFile.Name()), "ufile-"))
}

// SendFile return file by it ID
func SendFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	file, err := ioutil.ReadFile(fmt.Sprintf(FilesDir() + "/ufile-" + vars["id"]))
	if err != nil {
		fmt.Fprintf(w, "File %v not found\n", vars["id"])
		return
	}
	fmt.Fprintln(w, string(file))
}

// WpasteRouter return server router
func WpasteRouter() *mux.Router {
	Router := mux.NewRouter().StrictSlash(true)

	Router.HandleFunc("/", Help).Methods("GET")
	Router.HandleFunc("/", UploadFile).Methods("POST")

	Router.HandleFunc("/{id}", SendFile)
	return Router
}

// Basedir return root working directory
func Basedir() string {
	basedir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return basedir
}

// FilesDir return directory where uploaded files stored
func FilesDir() string {
	return Basedir() + "/files"
}

// Install prepare to start
func Install() {
	os.Mkdir(FilesDir(), 0766)
}

func main() {
	Install()
	http.ListenAndServe(":9990", WpasteRouter())
}
