// read-only registry API server

// TODO:
// * directly read from the .tar file (will break martini.Static)
// * UX friendly error handling

package main

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/martini"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

func main() {
	var dataDir string
	if len(os.Args) < 2 {
		var err error
		dataDir, err = os.Getwd()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(2)
		}
	} else {
		dataDir = os.Args[1]
	}

	m := martini.Classic()

	// Special headers expected by docker-pull.
	m.Use(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("X-Docker-Registry-Version", "activestate-0.1")
		res.Header().Set("X-Docker-Endpoints", req.Host)
	})

	// Using this only for fetching the layer and ancestry.
	m.Use(martini.Static(dataDir, martini.StaticOptions{Prefix: "/static/"}))

	m.Get("/", func() string {
		return "ActiveState's read-only docker-registry API server"
	})
	m.Get("/v1/_ping", func() string {
		return "true"
	})
	m.Get("/v1/repositories/:user/:name/images", func(params martini.Params, res http.ResponseWriter, req *http.Request) {
		http.Redirect(res, req, "/static/repositories/"+params["user"]+"/"+params["name"]+"/_index_images", http.StatusFound)
	})
	m.Get("/v1/repositories/:user/:name/tags", func(params martini.Params) (int, string) {
		pat := path.Join(dataDir, "repositories", params["user"], params["name"], "tag_*")
		if matches, err := filepath.Glob(pat); err != nil {
			return 404, fmt.Sprintf("No tags found: %v", err)
		} else {
			tags := map[string]string{}
			for _, f := range matches {
				fn := filepath.Base(f)
				tag := fn[len("tag_"):]
				if data, err := ioutil.ReadFile(f); err != nil {
					return 500, fmt.Sprintf("%v", err)
				} else {
					tags[tag] = string(data)
				}
			}
			if data, err := json.Marshal(tags); err != nil {
				return 500, fmt.Sprintf("%v", err)
			} else {
				return 200, string(data)
			}
		}
	})
	m.Get("/v1/images/:imgid/ancestry", func(params martini.Params, res http.ResponseWriter, req *http.Request) {
		http.Redirect(res, req, "/static/images/"+params["imgid"]+"/ancestry", http.StatusFound)
	})
	m.Get("/v1/images/:imgid/json", func(params martini.Params, res http.ResponseWriter) (int, string) {
		imgDir := path.Join(dataDir, "images", params["imgid"])
		jsonPath := path.Join(imgDir, "json")
		if data, err := ioutil.ReadFile(jsonPath); err != nil {
			return 404, fmt.Sprintf("%v", err)
		} else {
			// Store the layer size in X-Docker-Size
			layerPath := path.Join(imgDir, "layer")
			if fi, err := os.Stat(layerPath); err != nil {
				return 404, fmt.Sprintf("%v", err)
			} else {
				res.Header().Set("X-Docker-Size", fmt.Sprintf("%d", fi.Size()))
				return 200, string(data)
			}
		}
	})
	m.Get("/v1/images/:imgid/layer", func(params martini.Params, res http.ResponseWriter, req *http.Request) {
		http.Redirect(res, req, "/static/images/"+params["imgid"]+"/layer", http.StatusFound)
	})

	m.Run()
}
