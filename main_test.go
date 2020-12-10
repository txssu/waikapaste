package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func createPost(site string, params map[string]string) (*http.Request, error) {
	form := url.Values{}
	for key, val := range params {
		form.Add(key, val)
	}
	req, err := http.NewRequest("POST", site, strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return req, err
}

func sendAndGet(hc *http.Client, site string, params map[string]string) (*http.Response, error) {
	req, err := createPost(site, params)
	if err != nil {
		return nil, err
	}

	res, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("status code not OK")
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	res, err = http.Get(site + "/" + string(body))
	return res, err
}

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

	expected := "Hello, world!"

	res, err := sendAndGet(&hc, srv.URL, map[string]string{"f": expected})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("status not OK")
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != expected {
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

	s := bytes.Buffer{}
	for s.Len() < 10<<20 {
		s.Write([]byte("s"))
	}

	req, err := createPost(srv.URL, map[string]string{"f": s.String()})
	if err != nil {
		t.Fatal(err)
	}
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

	expected := "Hello, world!"
	name := "hello_world"

	req, err := createPost(srv.URL, map[string]string{"f": expected, "name": name})
	if err != nil {
		t.Fatal(err)
	}

	res, err := hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("status not OK")
	}
	res.Body.Close()

	res, err = http.Get(srv.URL + "/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("status not OK")
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != expected {
		t.Fail()
	}
}
