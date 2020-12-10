package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRouting_RedirectFromMain(t *testing.T) {
	Install()
	srv := httptest.NewServer(WpasteRouter())
	defer srv.Close()

	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if res.Request.URL.String() == srv.URL {
		t.Fail()
	}
}

func TestRouting_UploadAndGetFiles(t *testing.T) {
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
		t.Fail()
	}
}

func TestRouting_Errors(t *testing.T) {
	Install()
	srv := httptest.NewServer(WpasteRouter())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/testtest")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 404 {
		t.Fail()
	}

	hc := http.Client{}

	form := url.Values{}

	s := bytes.Buffer{}
	for s.Len() < 10<<20 {
		s.Write([]byte("s"))
	}

	form.Add("f", s.String())

	req, err := http.NewRequest("POST", srv.URL, strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err = hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 413 {
		t.Fail()
	}
}

func TestRouting_UploadWithName(t *testing.T) {
	Install()
	srv := httptest.NewServer(WpasteRouter())
	defer srv.Close()

	hc := http.Client{}

	form := url.Values{}
	form.Add("f", "Hello, world!")
	form.Add("name", "hi")

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
}