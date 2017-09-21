package main

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/jawher/mow.cli"
	log "github.com/sirupsen/logrus"

	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/gorilla/mux"
	"github.com/rcrowley/go-metrics"

	health "github.com/Financial-Times/go-fthealth/v1_1"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	_ "github.com/joho/godotenv/autoload"
)

const appDescription = "Hackathon for MumMe team to create a topic and annotate"

func main() {
	app := cli.App("hackathonTopicCreateAndAnnotate", appDescription)

	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "hackathonTopicCreateAndAnnotate",
		Desc:   "System Code of the application",
		EnvVar: "APP_SYSTEM_CODE",
	})

	appName := app.String(cli.StringOpt{
		Name:   "app-name",
		Value:  "hackathonTopicCreateAndAnnotate",
		Desc:   "Application name",
		EnvVar: "APP_NAME",
	})

	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "Port to listen on",
		EnvVar: "APP_PORT",
	})

	smartlogicAPIKey := app.String(cli.StringOpt{
		Name:   "smartlogicAPIKey",
		Desc:   "Smartlogic model to read from",
		EnvVar: "SMARTLOGIC_API_KEY",
	})

	APIKey := app.String(cli.StringOpt{
		Name:   "APIKey",
		Desc:   "API Key",
		EnvVar: "API_KEY",
	})
	slRequestURL := app.String(cli.StringOpt{
		Name:   "slRequestURL",
		Desc:   "SL_REQUEST_URL",
		EnvVar: "SL_REQUEST_URL",
	})

	log.SetLevel(log.InfoLevel)
	log.Infof("[Startup] hackathonTopicCreateAndAnnotate is starting ")

	app.Action = func() {
		log.Infof("System code: %s, App Name: %s, Port: %s", *appSystemCode, *appName, *port)

		requestHandler := RequestHandler{NewSmartlogicService(*smartlogicAPIKey, *slRequestURL), NewAnnotationsService(*APIKey)}

		go func() {
			serveEndpoints(*appSystemCode, *appName, *port, requestHandler)
		}()

		waitForSignal()
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("App could not start, error=[%s]\n", err)
		return
	}
}

func serveEndpoints(appSystemCode string, appName string, port string, requestHandler RequestHandler) {
	healthService := newHealthService(&healthConfig{appSystemCode: appSystemCode, appName: appName, port: port})

	serveMux := http.NewServeMux()

	hc := health.HealthCheck{SystemCode: appSystemCode, Name: appName, Description: appDescription, Checks: healthService.checks}
	serveMux.HandleFunc(healthPath, health.Handler(hc))
	serveMux.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(healthService.gtgCheck))
	serveMux.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)

	servicesRouter := mux.NewRouter()
	servicesRouter.HandleFunc("/topic", requestHandler.createTopic).Methods("PUT")

	servicesRouter.HandleFunc("/annotations", requestHandler.sendAnnotations).Methods("PUT")

	var monitoringRouter http.Handler = servicesRouter
	//monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(*log.StandardLogger(), monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)

	serveMux.Handle("/", monitoringRouter)

	server := &http.Server{Addr: ":" + port, Handler: serveMux}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Infof("HTTP server closing with message: %v", err)
		}
		wg.Done()
	}()

	waitForSignal()
	log.Infof("[Shutdown] hackathonTopicCreateAndAnnotate is shutting down")

	if err := server.Close(); err != nil {
		log.Errorf("Unable to stop http server: %v", err)
	}

	wg.Wait()
}

func waitForSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
