// Copyright 2015 The Syncato Authors.  All rights reserved.
// Use of this source code is governed by a AGPL
// license that can be found in the LICENSE file.

// Package main defines the syncato daemon. This daemon is reponsible for
// serving the APIs and the frontend applications.
package main

import (
	"fmt"
	"net/http"
	"os"

	apiauth "github.com/syncato/apis/auth"
	apifiles "github.com/syncato/apis/files"
	apimux "github.com/syncato/lib/api/mux"
	authmux "github.com/syncato/lib/auth/mux"
	authjson "github.com/syncato/lib/auth/providers/json"
	config "github.com/syncato/lib/config"
	logger "github.com/syncato/lib/logger"
	storagemux "github.com/syncato/lib/storage/mux"
	storagelocal "github.com/syncato/lib/storage/providers/local"

	"code.google.com/p/go-uuid/uuid"
	"github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
)

func main() {
	// We parse the options from the command line
	opts, err := getServerOptions()
	if err != nil {
		logrus.WithFields(logrus.Fields{"server": "main.go", "err": err}).Error("error parsing cli options")
		os.Exit(1)
	}

	if opts.createconfig == true {
		err = createConfigFile(opts.config)
		if err != nil {
			logrus.WithFields(logrus.Fields{"server": "main.go", "err": err}).Error("error creating custom configuration file")
			os.Exit(1)
		}
		os.Exit(0)
	}

	router := httprouter.New()
	router.Handle("GET", "/*catchall", handleRequest(opts))
	router.Handle("POST", "/*catchall", handleRequest(opts))
	router.Handle("PUT", "/*catchall", handleRequest(opts))
	router.Handle("DELETE", "/*catchall", handleRequest(opts))
	router.Handle("OPTIONS", "/*catchall", handleRequest(opts))
	router.Handle("HEAD", "/*catchall", handleRequest(opts))

	fmt.Println("Listening at port ", opts.port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", opts.port), router)
	if err != nil {
		fmt.Println("error listening: ", err)
		os.Exit(1)
	}

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
		cfg, err := config.New(opts.config, log)
		if err != nil {
			logrus.WithFields(logrus.Fields{"server": "main.go", "err": err}).Error("error creating config provider")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		/***************************
		 ** AUTH BACKENDS **********
		****************************/

		// This is an instance of the JSON file based authentication
		authJSON, err := authjson.NewAuthJSON("json", cfg, log)
		if err != nil {
			log.Error(err, nil)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// This is the instance of the authentication multiplexer
		authMux, err := authmux.NewAuthMux(cfg, log)
		if err != nil {
			log.Error(err, nil)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// We register the JSON auth file based authentication to be used by the auth mux
		err = authMux.RegisterAuthProvider(authJSON)
		if err != nil {
			log.Error(err, nil)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		/******************************
		 ** STORAGE BACKENDS **********
		 ******************************/

		// This is an instance of the storages we are going to use
		storageLocal, err := storagelocal.NewStorageLocal("local", cfg, log)
		if err != nil {
			log.Error(err, nil)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// This is an instance of the storage multiplexer
		storageMux, err := storagemux.NewStorageMux(log)

		// We register the local storage inside our storage mux
		err = storageMux.AddStorageProvider(storageLocal)
		if err != nil {
			log.Error(err, nil)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		/***************************
		 ** REQ CONTEXT **********
		****************************/

		// This is the root context passed to all requests
		rootCtx := context.Background()
		ctx := context.WithValue(rootCtx, "log", log)
		ctx = context.WithValue(ctx, "authMux", authMux)
		ctx = context.WithValue(ctx, "storageMux", storageMux)
		ctx = context.WithValue(ctx, "cfg", cfg)

		/*******************
		 ** APIS  **********
		********************/
		//filesAPI := filesapi.NewFilesAPI("files")
		apiAuth := apiauth.NewAPIAuth("auth")
		apiFiles := apifiles.NewAPIFiles("files")
		apiMux, _ := apimux.NewAPIMux(log)
		apiMux.RegisterApi(apiAuth)
		apiMux.RegisterApi(apiFiles)

		apiMux.HandleRequest(ctx, w, r)

	}
	return fn
}
