package main

import (
	"github.com/julienschmidt/httprouter"
	"github.com/syncato/syncato-api/filesapi"
	"golang.org/x/net/context"
	"net/http"
)

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/syncato/syncato-lib/auth/jsonauth"
	"github.com/syncato/syncato-lib/auth/muxauth"
	"github.com/syncato/syncato-lib/config"
	"github.com/syncato/syncato-lib/logger"
	"github.com/syncato/syncato-lib/storage/localstorage"
	"github.com/syncato/syncato-lib/storage/muxstorage"
	"os"
)

var APIRoutes = make(map[string]map[string]map[string]func(ctx context.Context, w http.ResponseWriter, r *http.Request))

func main() {

	// We parse the options from the command line
	opts, err := getServerOptions()
	if err != nil {
		logrus.WithFields(logrus.Fields{"server": "main.go", "err": err}).Error("error parsing cli options")
		os.Exit(1)
	}

	if opts.createconfig == true {
		if err != nil {
			logrus.WithFields(logrus.Fields{"server": "main.go", "err": err}).Error("error creating custom configuration file")
			os.Exit(1)
		}
		os.Exit(0)
	}

	router := httprouter.New()
	router.Handle("GET", "/api/:id/:op/*path", handleRequest(opts))

	http.ListenAndServe(fmt.Sprintf(":%d", opts.port), router)

}

func handleRequest(opts *Options) httprouter.Handle {
	fn := func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		// This is the log instance passed on the request inside the context
		// we use this log to have a common log per request, so libraries can use this log
		// and we can track the flow based on the request id
		log := logger.NewLogger(uuid.New(), opts.loglevel)
		log.Info("Request started", map[string]interface{}{"URL": r.RequestURI})
		defer func() {
			log.Info("Request finished", nil)

		}()

		// This config provider instance give us access to the configuration file
		cp, err := config.NewConfigProvider(opts.config, log)
		if err != nil {
			logrus.WithFields(logrus.Fields{"server": "main.go", "err": err}).Error("error creating config provider")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// This is an instance of the JSON file based authentication
		jsonAuth, err := jsonauth.NewJSONAuth("json", cp, log)
		if err != nil {
			log.Error(err, nil)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// This is the instance of the authentication multiplexer
		authMux, err := muxauth.NewMuxAuth(cp, log)
		if err != nil {
			log.Error(err, nil)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// We register the JSON auth file based authentication to be used by the auth mux
		err = authMux.RegisterAuthProvider(jsonAuth)
		if err != nil {
			log.Error(err, nil)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// This is an instance of the storages we are going to use
		localStorage, err := localstorage.NewLocalStorage("local", cp, log)
		if err != nil {
			log.Error(err, nil)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// This is an instance of the storage multiplexer
		storageMux, err := muxstorage.NewMuxStorage(log)

		// We register the local storage inside our storage mux
		err = storageMux.RegisterStorage(localStorage)
		if err != nil {
			log.Error(err, nil)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// This is the root context passed to all requests
		rootCtx := context.Background()
		ctx := context.WithValue(rootCtx, "log", log)
		ctx = context.WithValue(ctx, "authMux", authMux)
		ctx = context.WithValue(ctx, "storageMux", storageMux)
		ctx = context.WithValue(ctx, "cp", cp)

		// We get the API information based on the path
		//apiID := params.ByName("id")
		//path := params.ByName("path")

		// We load our apis
		//filesAPI := filesapi.NewFilesAPI()
		//filesAPI
		/*
			path := params.ByName("path")
			if strings.HasPrefix(path, "/api/files/") {
				filesapi.HandleRequest(ctx, w, r)
			} else {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
		*/
		filesAPI := filesapi.NewFilesAPI()
		op := filesAPI.GetOps()[0]
		log.Info("op", map[string]interface{}{"op": op})
		op.Execute(ctx, w, r)

	}
	return fn
}

/*
// parse return API information
// All APIs are reached trought the pattern
// "/api/:api_name/:api_operation/:api_path"
// Given a clean URL after splitting by "/" we should get at minimum 5 parts
func parse(url string) (string, error) {
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		return "", errors.New("path does not match any API")
	}
	path := strings.Join(parts[1:], sep)
	toret := strings.Join(a, sep)
}

*/
