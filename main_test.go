package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/appleboy/gofight"
	"github.com/stretchr/testify/assert"
)

type Env struct {
	r      *gofight.RequestConfig
	router http.Handler
	info   map[string]string
}

var env *Env

func TestStart(t *testing.T) {
	run("test.db", time.Second, 2*time.Second, false)
	env = &Env{
		r:      gofight.New(),
		router: logging(WpasteRouter()),
	}
	log.SetOutput(ioutil.Discard)
}

func TestMainPage(t *testing.T) {
	env.r.GET("/").
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestUploadAndGet(t *testing.T) {
	expected := "Hello, world!"
	var ID string
	env.r.POST("/").
		SetForm(gofight.H{
			"f": expected,
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			ID = r.Body.String()
			assert.Equal(t, http.StatusOK, r.Code)
		})
	env.r.GET("/"+ID).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, expected, r.Body.String())
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestEmptyF(t *testing.T) {
	env.r.POST("/").
		SetForm(gofight.H{}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusBadRequest, r.Code)
		})

}

func TestUploadAndGetWithName(t *testing.T) {
	name := "testname"
	expected := "Hello, world!"
	env.r.POST("/").
		SetForm(gofight.H{
			"name": name,
			"f":    expected,
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})
	env.r.GET("/"+name).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, expected, r.Body.String())
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func TestExpiredNotInt(t *testing.T) {
	env.r.POST("/").
		SetForm(gofight.H{
			"f": "something",
			"e": "time.Time",
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnprocessableEntity, r.Code)
		})

}

func TestExpiredNegative(t *testing.T) {
	env.r.POST("/").
		SetForm(gofight.H{
			"f": "something",
			"e": "-5",
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusBadRequest, r.Code)
		})

}

func TestNotFoundError(t *testing.T) {
	name := "404"
	env.r.GET("/"+name).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusNotFound, r.Code)
		})
}

func TestSameNameError(t *testing.T) {
	name := "same"
	env.r.POST("/").
		SetForm(gofight.H{
			"f":    "No. I am your father.",
			"name": name,
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})
	env.r.POST("/").
		SetForm(gofight.H{
			"f":    "No... No. That's not true! That's impossible!",
			"name": name,
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusConflict, r.Code)
		})
}

func TestLargeFileError(t *testing.T) {
	f := strings.Repeat("0", 10<<20)
	env.r.POST("/").
		SetForm(gofight.H{
			"f": f,
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusRequestEntityTooLarge, r.Code)
		})
}

func TestProtectedFile(t *testing.T) {
	password := "USA. Top secret"

	var name string
	env.r.POST("/").
		SetForm(gofight.H{
			"f":  "42",
			"ap": password,
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			name = r.Body.String()
		})

	testCases := []struct {
		params gofight.H
		code   int
	}{
		{gofight.H{}, http.StatusUnauthorized},                          // Without password
		{gofight.H{"ap": "China. Top public"}, http.StatusUnauthorized}, // Invalid password
		{gofight.H{"ap": password}, http.StatusOK},
	}

	for _, cs := range testCases {
		env.r.GET("/"+name).
			SetQuery(cs.params).
			Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
				assert.Equal(t, cs.code, r.Code)
			})
	}
}

func TestEditFile(t *testing.T) {
	data := "42"
	newData := "43"
	password := "USA. Top secret"
	e := "1"

	var name string
	env.r.POST("/").
		SetForm(gofight.H{
			"f":  data,
			"e":  e,
			"ep": password,
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			name = r.Body.String()
		})

	testCases := []struct {
		method  string
		path    string
		params  gofight.H
		asserts func(gofight.HTTPResponse, gofight.HTTPRequest)
	}{
		{"GET", "/" + name, gofight.H{}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			assert.Equal(t, data, r.Body.String())
		}},
		// Without password
		{"PUT", "/" + name, gofight.H{"f": newData}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnauthorized, r.Code)
		}},
		// Invalid password
		{"PUT", "/" + name, gofight.H{"f": newData, "ep": "China. Top public"}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnauthorized, r.Code)
		}},
		// Success change
		{"PUT", "/" + name, gofight.H{"f": newData, "ep": password}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		}},
		// Check
		{"GET", "/" + name, gofight.H{}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			assert.Equal(t, newData, r.Body.String())
		}},
		// Large data
		{"PUT", "/" + name, gofight.H{"f": strings.Repeat("0", 10<<20), "ep": password}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusRequestEntityTooLarge, r.Code)
		}},
		// Without Data
		{"PUT", "/" + name, gofight.H{"ep": password}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusBadRequest, r.Code)
		}},
		// Invalid name
		{"PUT", "/nnnnnnnn775", gofight.H{"f": newData, "ep": password}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusNotFound, r.Code)
			time.Sleep(1 * time.Second)
		}},
		// Edit expired file
		{"PUT", "/" + name, gofight.H{"f": data, "ep": password}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusGone, r.Code)
		}},
	}

	for _, cs := range testCases {
		var rq *gofight.RequestConfig
		switch cs.method {
		case "PUT":
			rq = env.r.PUT(cs.path)
		case "GET":
			rq = env.r.GET(cs.path)
		}
		rq.SetForm(cs.params).
			Run(env.router, cs.asserts)
	}
}

func TestEditFileWithoutEP(t *testing.T) {
	data := "42"
	newData := "43"
	password := "USA. Top secret"

	var name string
	env.r.POST("/").
		SetForm(gofight.H{
			"f": data,
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			name = r.Body.String()
		})

	env.r.PUT("/"+name).
		SetForm(gofight.H{
			"f":  newData,
			"ep": password,
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnauthorized, r.Code)
		})
}

func TestDeleteFile(t *testing.T) {
	data := "China"
	password := "maodzedun"

	var name string
	env.r.POST("/").
		SetForm(gofight.H{
			"f":  data,
			"ep": password,
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			name = r.Body.String()
		})

	testCases := []struct {
		method  string
		path    string
		params  gofight.H
		asserts func(gofight.HTTPResponse, gofight.HTTPRequest)
	}{
		// Invalid password
		{"DELETE", "/" + name, gofight.H{"ep": "password"}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnauthorized, r.Code)
		}},
		// OK
		{"DELETE", "/" + name, gofight.H{"ep": password}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		}},
		// Check deleted
		{"GET", "/" + name, gofight.H{}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusNotFound, r.Code)
		}},
		// Delete not exist
		{"DELETE", "/abcd", gofight.H{"ep": password}, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusNotFound, r.Code)
		}},
	}

	for _, cs := range testCases {
		var rq *gofight.RequestConfig
		switch cs.method {
		case "DELETE":
			rq = env.r.DELETE(cs.path)
		case "GET":
			rq = env.r.GET(cs.path)
		}
		rq.SetQuery(cs.params).
			Run(env.router, cs.asserts)
	}
}

func TestFileExpired(t *testing.T) {
	e := 1
	var ID string
	env.r.POST("/").
		SetForm(gofight.H{
			"f": "*uck. Duck. I said duck.",
			"e": strconv.Itoa(e),
		}).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			ID = r.Body.String()
			assert.Equal(t, http.StatusOK, r.Code)
		})
	time.Sleep(time.Duration(e) * time.Second)
	env.r.GET("/"+ID).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusGone, r.Code)
		})
	time.Sleep(time.Duration(3) * time.Second)
	env.r.GET("/"+ID).
		Run(env.router, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusNotFound, r.Code)
		})
}

func TestFinish(t *testing.T) {
	db.Close()

	e := os.Remove("test.db")
	if e != nil {
		log.Fatal(e)
	}
}
