package daemon

import (
	"encoding/json"
	"net/http"

	"github.com/elastic/stack-operators/local-volume/pkg/driver/daemon/pathutil"
	"github.com/elastic/stack-operators/local-volume/pkg/driver/protocol"
	log "github.com/sirupsen/logrus"
)

// SetupRoutes returns an http ServeMux to handle all our HTTP routes
func (s *Server) SetupRoutes() *http.ServeMux {
	handler := http.NewServeMux()
	handler.HandleFunc("/init", s.InitHandler())
	handler.HandleFunc("/mount", s.MountHandler())
	handler.HandleFunc("/unmount", s.UnmountHandler())
	return handler
}

// InitHandler handles init HTTP calls
func (s *Server) InitHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("Init request")

		resp := s.driver.Init()
		log.Infof("%+v", resp)

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			err500(w, err)
		}
	}
}

// MountHandler handles mount HTTP calls
func (s *Server) MountHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("Mount request")

		defer r.Body.Close()
		var params protocol.MountRequest
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			err500(w, err)
			return
		}

		pvName := pathutil.ExtractPVCID(params.TargetDir)
		log.Infof("Updating PV %s with affinity for node %s", pvName, s.nodeName)
		if err := s.k8sClient.UpdatePVNodeAffinity(pvName, s.nodeName); err != nil {
			log.WithError(err).Error("Cannot update Persistent Volume node affinity")
			err500(w, err)
		}

		log.Info("Mounting volume to the host")
		resp := s.driver.Mount(params)
		log.Infof("%+v", resp)

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			err500(w, err)
		}
	}
}

// UnmountHandler handles unmount HTTP calls
func (s *Server) UnmountHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("Unmount request")

		defer r.Body.Close()
		var params protocol.UnmountRequest
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			err500(w, err)
			return
		}

		resp := s.driver.Unmount(params)
		log.Infof("%+v", resp)

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			err500(w, err)
		}
	}
}

// err500 logs an error and writes it in the http response
func err500(w http.ResponseWriter, err error) {
	log.WithError(err).Error()
	http.Error(w, err.Error(), 500)
}
