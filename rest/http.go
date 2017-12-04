package rest

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

	"github.com/hashicorp/consul/api"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

type router struct {
	*httprouter.Router
}

func (r *router) Get(path string, handler http.Handler) {
	r.GET(path, wrapHandler(handler))
}

func (r *router) Post(path string, handler http.Handler) {
	r.POST(path, wrapHandler(handler))
}

func (r *router) Put(path string, handler http.Handler) {
	r.PUT(path, wrapHandler(handler))
}

func (r *router) Delete(path string, handler http.Handler) {
	r.DELETE(path, wrapHandler(handler))
}

func (r *router) Head(path string, handler http.Handler) {
	r.HEAD(path, wrapHandler(handler))
}

func wrapHandler(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		h.ServeHTTP(w, r.WithContext(context.WithValue(ctx, "params", ps)))
	}
}

func newRouter() *router {
	return &router{httprouter.New()}
}

// A Server is an HTTP server that runs the Janus REST API
type Server struct {
	router         *router
	listener       net.Listener
	consulClient   *api.Client
	tasksCollector *tasks.Collector
	config         config.Configuration
}

// Shutdown stops the HTTP server
func (s *Server) Shutdown() {
	if s != nil {
		log.Printf("Shutting down http server")
		err := s.listener.Close()
		if err != nil {
			log.Print(errors.Wrap(err, "Failed to close server listener"))
		}
	}
}

// NewServer create a Server to serve the REST API
func NewServer(configuration config.Configuration, client *api.Client, shutdownCh chan struct{}) (*Server, error) {
	addr, err := getAddress(configuration)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen(addr.Network(), addr.String())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to bind on %s", addr)
	}

	if configuration.CertFile != "" && configuration.KeyFile != "" {
		listener, err = wrapListenerTLS(listener, configuration)
		if err != nil {
			return nil, err
		}
	}

	httpServer := &Server{
		router:         newRouter(),
		listener:       listener,
		consulClient:   client,
		tasksCollector: tasks.NewCollector(client),
		config:         configuration,
	}

	httpServer.registerHandlers()
	if configuration.CertFile != "" && configuration.KeyFile != "" {
		log.Printf("Starting HTTPServer over TLS on address %s", listener.Addr())
		log.Debugf("TLS KeyFile in use: %q. TLS CertFile in use: %q", configuration.KeyFile, configuration.CertFile)
	} else {
		log.Printf("Starting HTTPServer on address %s", listener.Addr())
	}
	go http.Serve(httpServer.listener, httpServer.router)

	return httpServer, nil
}

func (s *Server) registerHandlers() {
	commonHandlers := alice.New(telemetryHandler, loggingHandler, recoverHandler)
	s.router.Post("/deployments", commonHandlers.Append(contentTypeHandler("application/zip")).ThenFunc(s.newDeploymentHandler))
	s.router.Put("/deployments/:id", commonHandlers.Append(contentTypeHandler("application/zip")).ThenFunc(s.newDeploymentHandler))
	s.router.Delete("/deployments/:id", commonHandlers.ThenFunc(s.deleteDeploymentHandler))
	s.router.Get("/deployments/:id", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.getDeploymentHandler))
	s.router.Get("/deployments", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.listDeploymentsHandler))
	s.router.Get("/deployments/:id/events", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.pollEvents))
	s.router.Get("/events", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.pollEvents))
	s.router.Head("/deployments/:id/events", commonHandlers.ThenFunc(s.headEventsIndex))
	s.router.Head("/events", commonHandlers.ThenFunc(s.headEventsIndex))
	s.router.Get("/deployments/:id/logs", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.pollLogs))
	s.router.Get("/logs", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.pollLogs))
	s.router.Head("/deployments/:id/logs", commonHandlers.ThenFunc(s.headLogsEventsIndex))
	s.router.Head("/logs", commonHandlers.ThenFunc(s.headLogsEventsIndex))
	s.router.Get("/deployments/:id/nodes/:nodeName", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.getNodeHandler))
	s.router.Get("/deployments/:id/nodes/:nodeName/instances/:instanceId", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.getNodeInstanceHandler))
	s.router.Get("/deployments/:id/outputs", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.listOutputsHandler))
	s.router.Get("/deployments/:id/outputs/:opt", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.getOutputHandler))
	s.router.Get("/deployments/:id/tasks/:taskId", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.getTaskHandler))
	s.router.Get("/deployments/:id/tasks/:taskId/steps", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.getTaskStepsHandler))
	s.router.Delete("/deployments/:id/tasks/:taskId", commonHandlers.ThenFunc(s.cancelTaskHandler))
	s.router.Put("/deployments/:id/tasks/:taskId", commonHandlers.ThenFunc(s.resumeTaskHandler))
	s.router.Put("/deployments/:id/tasks/:taskId/steps/:stepId", commonHandlers.Append(contentTypeHandler("application/json")).ThenFunc(s.updateTaskStepStatusHandler))
	s.router.Post("/deployments/:id/scale/:nodeName", commonHandlers.ThenFunc(s.scaleHandler))
	s.router.Get("/deployments/:id/nodes/:nodeName/instances/:instanceId/attributes", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.getNodeInstanceAttributesListHandler))
	s.router.Get("/deployments/:id/nodes/:nodeName/instances/:instanceId/attributes/:attributeName", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.getNodeInstanceAttributeHandler))
	s.router.Post("/deployments/:id/custom", commonHandlers.Append(contentTypeHandler("application/json")).ThenFunc(s.newCustomCommandHandler))
	s.router.Post("/deployments/:id/workflows/:workflowName", commonHandlers.ThenFunc(s.newWorkflowHandler))
	s.router.Get("/deployments/:id/workflows/:workflowName", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.getWorkflowHandler))
	s.router.Get("/deployments/:id/workflows", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.listWorkflowsHandler))

	s.router.Get("/registry/delegates", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.listRegistryDelegatesHandler))
	s.router.Get("/registry/implementations", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.listRegistryImplementationsHandler))
	s.router.Get("/registry/definitions", commonHandlers.Append(acceptHandler("application/json")).ThenFunc(s.listRegistryDefinitionsHandler))

	if s.config.Telemetry.PrometheusEndpoint {
		s.router.Get("/metrics", commonHandlers.Then(promhttp.Handler()))
	}
}

func encodeJSONResponse(w http.ResponseWriter, r *http.Request, resp interface{}) {
	jEnc := json.NewEncoder(w)
	if _, ok := r.URL.Query()["pretty"]; ok {
		jEnc.SetIndent("", "  ")
	}
	w.Header().Set("Content-Type", "application/json")
	jEnc.Encode(resp)
}

func getAddress(configuration config.Configuration) (net.Addr, error) {

	var port int
	if configuration.HTTPPort == 0 {
		// Use default value
		port = config.DefaultHTTPPort
	} else if configuration.HTTPPort < 0 {
		// Use random port
		port = 0
	} else {
		port = configuration.HTTPPort
	}
	var ip net.IP
	if configuration.HTTPAddress != "" {
		ip = net.ParseIP(configuration.HTTPAddress)
	} else {
		ip = net.ParseIP(config.DefaultHTTPAddress)
	}
	if ip == nil {
		return nil, errors.Errorf("Failed to parse IP: %v", configuration.HTTPAddress)
	}
	return &net.TCPAddr{IP: ip, Port: port}, nil
}
