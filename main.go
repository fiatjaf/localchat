package main

import (
	"net/http"
	"net/url"
	"os"
	"time"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"gopkg.in/redis.v5"
)

var err error
var s Settings
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
var rds *redis.Client
var httpPublic = &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: ""}
var router = mux.NewRouter()

type Settings struct {
	Host       string `envconfig:"HOST" default:"0.0.0.0"`
	Port       string `envconfig:"PORT" required:"true"`
	ServiceURL string `envconfig:"SERVICE_URL" required:"true"`
	RedisURL   string `envconfig:"REDIS_URL" required:"true"`
}

func main() {
	err = envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
	}

	// redis connection
	rurl, _ := url.Parse(s.RedisURL)
	pw, _ := rurl.User.Password()
	rds = redis.NewClient(&redis.Options{
		Addr:     rurl.Host,
		Password: pw,
	})
	if err := rds.Ping().Err(); err != nil {
		log.Fatal().Err(err).Str("url", s.RedisURL).
			Msg("failed to connect to redis")
	}

	// routes
	router.PathPrefix("/static/").Methods("GET").Handler(http.FileServer(httpPublic))
	router.Path("/info").Methods("GET").HandlerFunc(info)
	router.Path("/{room}/stored").Methods("GET").HandlerFunc(storedMessages)
	router.Path("/{room}/receive").Methods("GET").HandlerFunc(messageStream)
	router.Path("/{room}/send").Methods("POST").HandlerFunc(newMessage)
	//	router.Path("/favicon.ico").Methods("GET").HandlerFunc(
	//		func(w http.ResponseWriter, r *http.Request) {
	//			w.Header().Set("Content-Type", "image/png")
	//			iconf, _ := httpPublic.Open("static/icon.png")
	//			fstat, _ := iconf.Stat()
	//			http.ServeContent(w, r, "static/icon.png", fstat.ModTime(), iconf)
	//			return
	//		})
	router.PathPrefix("/").Methods("GET").HandlerFunc(serveClient)

	// start http server
	log.Info().Str("host", s.Host).Str("port", s.Port).Msg("listening")
	srv := &http.Server{
		Handler:      router,
		Addr:         s.Host + ":" + s.Port,
		WriteTimeout: 300 * time.Second,
		ReadTimeout:  300 * time.Second,
	}
	err = srv.ListenAndServe()
	if err != nil {
		log.Error().Err(err).Msg("error serving http")
	}
}

func serveClient(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	indexf, err := httpPublic.Open("static/index.html")
	if err != nil {
		log.Error().Err(err).Str("file", "static/index.html").
			Msg("make sure you generated bindata.go without -debug")
		return
	}
	fstat, _ := indexf.Stat()
	http.ServeContent(w, r, "static/index.html", fstat.ModTime(), indexf)
	return
}
