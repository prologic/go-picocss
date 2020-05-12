package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	log "github.com/sirupsen/logrus"

	// Logging
	"github.com/unrolled/logger"

	// Stats/Metrics
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	"github.com/thoas/stats"

	rice "github.com/GeertJohan/go.rice"
	"github.com/NYTimes/gziphandler"
	"github.com/julienschmidt/httprouter"
)

// Router ...
type Router struct {
	httprouter.Router
}

// NewRouter ...
func NewRouter() *Router {
	return &Router{
		httprouter.Router{
			RedirectTrailingSlash:  true,
			RedirectFixedPath:      true,
			HandleMethodNotAllowed: true,
			HandleOPTIONS:          true,
		},
	}
}

// ServeFilesWithCacheControl ...
func (r *Router) ServeFilesWithCacheControl(path string, root http.FileSystem) {
	if len(path) < 10 || path[len(path)-10:] != "/*filepath" {
		panic("path must end with /*filepath in path '" + path + "'")
	}

	fileServer := http.FileServer(root)

	r.GET(path, func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Cache-Control", "public, max-age=7776000")
		req.URL.Path = ps.ByName("filepath")
		fileServer.ServeHTTP(w, req)
	})
}

// Counters ...
type Counters struct {
	r metrics.Registry
}

func NewCounters() *Counters {
	counters := &Counters{
		r: metrics.NewRegistry(),
	}
	return counters
}

func (c *Counters) Inc(name string) {
	metrics.GetOrRegisterCounter(name, c.r).Inc(1)
}

func (c *Counters) Dec(name string) {
	metrics.GetOrRegisterCounter(name, c.r).Dec(1)
}

func (c *Counters) IncBy(name string, n int64) {
	metrics.GetOrRegisterCounter(name, c.r).Inc(n)
}

func (c *Counters) DecBy(name string, n int64) {
	metrics.GetOrRegisterCounter(name, c.r).Dec(n)
}

// Server ...
type Server struct {
	bind      string
	templates *Templates
	router    *Router

	// Logger
	logger *logger.Logger

	// Stats/Metrics
	counters *Counters
	stats    *stats.Stats
}

func (s *Server) render(name string, w http.ResponseWriter, ctx interface{}) {
	buf, err := s.templates.Exec(name, ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = buf.WriteTo(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// IndexHandler ...
func (s *Server) IndexHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		s.counters.Inc("n_index")

		s.render("index", w, nil)
	}
}

// FormResult ...
type FormResult struct {
	Data map[string]string
}

// StatsHandler ...
func (s *Server) StatsHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		bs, err := json.Marshal(s.stats.Data())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Write(bs)
	}
}

// ListenAndServe ...
func (s *Server) ListenAndServe() {
	log.Fatal(
		http.ListenAndServe(
			s.bind,
			s.logger.Handler(
				s.stats.Handler(
					gziphandler.GzipHandler(
						s.router,
					),
				),
			),
		),
	)
}

func (s *Server) initRoutes() {
	s.router.Handler("GET", "/debug/metrics", exp.ExpHandler(s.counters.r))
	s.router.GET("/debug/stats", s.StatsHandler())

	s.router.ServeFilesWithCacheControl(
		"/css/*filepath",
		rice.MustFindBox("static/css").HTTPBox(),
	)

	s.router.ServeFilesWithCacheControl(
		"/img/*filepath",
		rice.MustFindBox("static/img").HTTPBox(),
	)

	s.router.ServeFilesWithCacheControl(
		"/js/*filepath",
		rice.MustFindBox("static/js").HTTPBox(),
	)

	s.router.GET("/favicon.ico", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		box, err := rice.FindBox("static")
		if err != nil {
			http.Error(w, "404 file not found", http.StatusNotFound)
			return
		}

		buf, err := box.Bytes("favicon.ico")
		if err != nil {
			msg := fmt.Sprintf("error reading favicon: %s", err)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/x-icon")
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Cache-Control", "public, max-age=7776000")

		n, err := w.Write(buf)
		if err != nil {
			log.Errorf("error writing response for favicon: %s", err)
		} else if n != len(buf) {
			log.Warnf(
				"not all bytes of favicon response were written: %d/%d",
				n, len(buf),
			)
		}
	})

	s.router.GET("/", s.IndexHandler())
}

// NewServer ...
func NewServer(bind string) *Server {
	server := &Server{
		bind:      bind,
		router:    NewRouter(),
		templates: NewTemplates("base"),

		// Logger
		logger: logger.New(logger.Options{
			Prefix:               "go-picocss",
			RemoteAddressHeaders: []string{"X-Forwarded-For"},
		}),

		// Stats/Metrics
		counters: NewCounters(),
		stats:    stats.New(),
	}

	// Templates
	box := rice.MustFindBox("templates")

	indexTemplate := template.New("index")
	template.Must(indexTemplate.Parse(box.MustString("index.html")))
	template.Must(indexTemplate.Parse(box.MustString("base.html")))
	server.templates.Add("index", indexTemplate)

	server.initRoutes()

	return server
}
