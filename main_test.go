package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRouting_UploadFiles(t *testing.T) {
	Install()
	srv := httptest.NewServer(WpasteRouter())
	defer srv.Close()
	hc := http.Client{}

	form := url.Values{}
	form.Add("f", "Hello, world!")

	req, err := http.NewRequest("POST", srv.URL, strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	postRes, err := hc.Do(req)

	if err != nil {
		t.Fatal(err)
	}

	if postRes.StatusCode != http.StatusOK {
		t.Errorf("status not OK")
	}
	defer postRes.Body.Close()

	bodyRes, err := ioutil.ReadAll(postRes.Body)

	res, err := http.Get(srv.URL + "/" + string(bodyRes))

	if err != nil {
		t.Fatal(err)
	}

	if postRes.StatusCode != http.StatusOK {
		t.Errorf("status not OK")
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "Hello, world!" {
		t.Fatal(string(body))
	}
}
