package main

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/appleboy/gofight"
	"github.com/stretchr/testify/assert"
)

func TestUploadAndGet(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	expected := "Hello, world!"
	var ID string
	r.POST("/").
		SetForm(gofight.H{
			"f": expected,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			ID = r.Body.String()
			assert.Equal(t, http.StatusOK, r.Code)
		})
	r.GET("/"+ID).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, expected, r.Body.String())
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestEmptyF(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	r.POST("/").
		SetForm(gofight.H{}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusBadRequest, r.Code)
		})

}

func TestUploadAndGetWithName(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	name := "testname"
	expected := "Hello, world!"
	r.POST("/").
		SetForm(gofight.H{
			"name": name,
			"f":    expected,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})
	r.GET("/"+name).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, expected, r.Body.String())
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestFileExpired(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	e := 1
	var ID string
	r.POST("/").
		SetForm(gofight.H{
			"f": "*uck. Duck. I said duck.",
			"e": strconv.Itoa(e),
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			ID = r.Body.String()
			assert.Equal(t, http.StatusOK, r.Code)
		})
	time.Sleep(time.Duration(e) * time.Second)
	r.GET("/"+ID).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusGone, r.Code)
		})
}

func TestExpiredNotInt(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	r.POST("/").
		SetForm(gofight.H{
			"f": "something",
			"e": "time.Time",
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnprocessableEntity, r.Code)
		})

}

func TestExpiredNegative(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	r.POST("/").
		SetForm(gofight.H{
			"f": "something",
			"e": "-5",
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusBadRequest, r.Code)
		})

}

func TestNotFoundError(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	name := "404"
	r.GET("/"+name).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusNotFound, r.Code)
		})
}

func TestSameNameError(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	name := "same"
	r.POST("/").
		SetForm(gofight.H{
			"f":    "No. I am your father.",
			"name": name,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})
	r.POST("/").
		SetForm(gofight.H{
			"f":    "No... No. That's not true! That's impossible!",
			"name": name,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusConflict, r.Code)
		})
}

func TestLargeFileError(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	f := strings.Repeat("0", 10<<20)
	r.POST("/").
		SetForm(gofight.H{
			"f": f,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusRequestEntityTooLarge, r.Code)
		})
}

func TestProtectedFile(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	password := "USA. Top secret"

	var name string
	r.POST("/").
		SetForm(gofight.H{
			"f":  "42",
			"ap": password,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			name = r.Body.String()
		})

	r.GET("/"+name).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnauthorized, r.Code)
		})

	r.GET("/"+name).
		SetQuery(gofight.H{
			"ap": "China. Top public",
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnauthorized, r.Code)
		})

	r.GET("/"+name).
		SetQuery(gofight.H{
			"ap": password,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestEditFile(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	data := "42"
	newData := "43"
	password := "USA. Top secret"

	var name string
	r.POST("/").
		SetForm(gofight.H{
			"f":  data,
			"ep": password,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			name = r.Body.String()
		})

	r.GET("/"+name).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			assert.Equal(t, data, r.Body.String())
		})

	r.PUT("/"+name).
		SetForm(gofight.H{
			"f": newData,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnauthorized, r.Code)
		})

	r.PUT("/"+name).
		SetForm(gofight.H{
			"f":  newData,
			"ep": "China. Top public",
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnauthorized, r.Code)
		})

	r.PUT("/"+name).
		SetForm(gofight.H{
			"f":  newData,
			"ep": password,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})

	r.GET("/"+name).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			assert.Equal(t, newData, r.Body.String())
		})

	f := strings.Repeat("0", 10<<20)
	r.PUT("/"+name).
		SetForm(gofight.H{
			"f":  f,
			"ep": password,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusRequestEntityTooLarge, r.Code)
		})

	r.PUT("/"+name).
		SetForm(gofight.H{
			"ep": password,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusBadRequest, r.Code)
		})

	r.PUT("/nnnnnnnn775").
		SetForm(gofight.H{
			"f":  "string",
			"ep": password,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusNotFound, r.Code)
		})
}

func TestEditFileWithoutEP(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	data := "42"
	newData := "43"
	password := "USA. Top secret"

	var name string
	r.POST("/").
		SetForm(gofight.H{
			"f": data,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			name = r.Body.String()
		})

	r.PUT("/"+name).
		SetForm(gofight.H{
			"f":  newData,
			"ep": password,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnauthorized, r.Code)
		})
}

func TestDeleteFile(t *testing.T) {
	Install()
	defer Close()
	r := gofight.New()

	data := "China"
	password := "maodzedun"

	var name string
	r.POST("/").
		SetForm(gofight.H{
			"f":  data,
			"ep": password,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			name = r.Body.String()
		})

	r.DELETE("/"+name).
		SetQuery(gofight.H{
			"ep": "password",
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnauthorized, r.Code)
		})

	r.DELETE("/"+name).
		SetQuery(gofight.H{
			"ep": password,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})

	r.GET("/"+name).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusNotFound, r.Code)
		})

	r.DELETE("/abcd").
		SetQuery(gofight.H{
			"ep": password,
		}).
		Run(WpasteRouter(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusNotFound, r.Code)
		})
}
