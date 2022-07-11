// Implements a server for distributing Cesium terrain tilesets
package main

import (
"flag"
"fmt"
myhandlers "github.com/geo-data/cesium-terrain-server/handlers"
"github.com/geo-data/cesium-terrain-server/log"
"github.com/geo-data/cesium-terrain-server/stores/fs"
"github.com/gorilla/handlers"
"github.com/gorilla/mux"
l "log"
"net/http"
"os"
)

func main(){
port := flag.Uint("port", 8000, "the port on which the server listens")
tilesetRoot := flag.String("dir", ".", "the root directory under which tileset directories reside")
webRoot := flag.String("web-dir", "", "(optional) the root directory containing static files to be served")
memcached := flag.String("memcached", "", "(optional) memcached connection string for caching tiles e.g. localhost:11211")
baseTerrainUrl := flag.String("base-terrain-url", "/tilesets", "base url prefix under which all tilesets are served")
noRequestLog := flag.Bool("no-request-log", false, "do not log client requests for resources")
https := flag.Bool("https", false, "if https is enabled")
pemPath := flag.String("pemPath", "", "Path to pem files cert.pem and key.pem")
logging := NewLogOpt()
flag.Var(logging, "log-level", "level at which logging occurs. One of crit, err, notice, debug")
limit := NewLimitOpt()
limit.Set("1MB")
flag.Var(limit, "cache-limit", the memory size in bytes beyond which resources are not cached. Other memory units can be specified by suffixing the number with kB, MB, GB or TB)
flag.Parse()

// Set the logging
log.SetLog(l.New(os.Stderr, "", l.LstdFlags), logging.Priority)

// Get the tileset store
store := fs.New(*tilesetRoot)

r := mux.NewRouter()
r.HandleFunc(*baseTerrainUrl+"/{tileset}/layer.json", myhandlers.LayerHandler(store))
r.HandleFunc(*baseTerrainUrl+"/{tileset}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.terrain", myhandlers.TerrainHandler(store))
if len(*webRoot) > 0 {
	log.Debug(fmt.Sprintf("serving static resources from %s", *webRoot))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(*webRoot)))
}

handler := myhandlers.AddCorsHeader(r)
if len(*memcached) > 0 {
	log.Debug(fmt.Sprintf("memcached enabled for all resources: %s", *memcached))
	handler = myhandlers.NewCache(*memcached, handler, limit.Value, myhandlers.NewLimit)
}

if *noRequestLog == false {
	handler = handlers.CombinedLoggingHandler(os.Stdout, handler)
}

http.Handle("/", handler)

if *https {

	if len(*pemPath) == 0 {
		log.Notice(fmt.Sprintf("Https Server Cannot be run with out  -pemPath specified %s", *pemPath))
		os.Exit(1)
	}

	keyFilesExist := true
	_, err := os.Stat(*pemPath + "/key.pem")
	if err != nil {
		keyFilesExist = false
	} else if os.IsNotExist(err) {
		keyFilesExist = false
	}

	_, err = os.Stat(*pemPath + "/cert.pem")
	if err != nil {
		keyFilesExist = false
	} else if os.IsNotExist(err) {
		keyFilesExist = false
	}

	if !keyFilesExist {
		log.Notice(fmt.Sprintf("Https Server Cannot be run key.pem and cert.pem do not exist in dir %s" + *pemPath))
		os.Exit(1)
	}

	log.Notice(fmt.Sprintf("Https Server listening on port %d", *port))
	
	err_http := http.ListenAndServeTLS(fmt.Sprintf(":%d", *port), *pemPath+"/cert.pem", *pemPath+"/key.pem", nil)
	if err_http != nil {
		fmt.Println(err_http)
	}

	os.Exit(1)
} else {
	log.Notice(fmt.Sprintf("Http Server listening on port %d", *port))

	err_http := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err_http != nil {
		fmt.Println(err_http)
	}
	os.Exit(1)
}

}
