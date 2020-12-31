package hive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

// Server is a hive server
type Server struct {
	*vk.Server
	h        *Hive
	inFlight map[string]*Result
	sync.Mutex
}

var client = &http.Client{}

func init() {
	client.Timeout = time.Duration(time.Second * 5)
}

func newServer(h *Hive, opts ...vk.OptionsModifier) *Server {
	s := vk.New(opts...)

	server := &Server{
		Server:   s,
		Mutex:    sync.Mutex{},
		h:        h,
		inFlight: make(map[string]*Result),
	}

	server.POST("/do/:jobtype", server.scheduleHandler())
	server.GET("/then/:id", server.thenHandler())

	return server
}

type doResponse struct {
	ResultID string `json:"resultId"`
}

func (s *Server) scheduleHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		jobType := ctx.Params.ByName("jobtype")
		if jobType == "" {
			return nil, vk.E(http.StatusBadRequest, "missing jobtype")
		}

		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, vk.E(http.StatusInternalServerError, "failed to read request body")
		}
		defer r.Body.Close()

		res := s.h.Do(NewJob(jobType, data))

		callback := r.URL.Query().Get("callback")
		if callback != "" {
			callbackURL, err := url.Parse(callback)
			if err != nil {
				return nil, vk.E(http.StatusBadRequest, errors.Wrap(err, "failed to parse callback URL").Error())
			}

			res.ThenDo(webhookCallback(callbackURL, ctx.Log))

			return vk.R(http.StatusOK, nil), nil
		}

		then := r.URL.Query().Get("then")
		if then == "true" {
			result, err := res.Then()
			if err != nil {
				return nil, vk.E(http.StatusInternalServerError, errors.Wrap(err, "job resulted in error").Error())
			}

			return result, nil
		}

		s.addInFlight(res)

		resp := doResponse{
			ResultID: res.UUID(),
		}

		return resp, nil
	}
}

func (s *Server) thenHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		id := ctx.Params.ByName("id")
		if len(id) != 24 {
			return nil, vk.E(http.StatusBadRequest, "invalid result ID")
		}

		res := s.getInFlight(id)
		if res == nil {
			return nil, vk.E(http.StatusNotFound, fmt.Sprintf("result with ID %s not found", id))
		}

		defer s.removeInFlight(id)

		result, err := res.Then()
		if err != nil {
			return nil, vk.E(http.StatusInternalServerError, errors.Wrap(err, "job resulted in error").Error())
		}

		return result, nil
	}
}

func webhookCallback(callbackURL *url.URL, log *vlog.Logger) ResultFunc {
	return func(res interface{}, err error) {
		var body []byte
		var contentType = "application/octet-stream"

		if err != nil {
			body = []byte(errors.Wrap(err, "job_err_result").Error())
		} else {
			// if result is bytes, send that
			if bytes, isBytes := res.([]byte); isBytes {
				body = bytes
			} else {
				// if not, attempt to Marshal it from a struct or error out
				json, err := json.Marshal(res)
				if err != nil {
					body = []byte(errors.Wrap(err, "job_err_result failed to Marshal result").Error())
				} else {
					contentType = "application/json"
					body = json
				}
			}
		}

		log.Info("sending callback to", callbackURL.String())

		_, postErr := client.Post(callbackURL.String(), contentType, bytes.NewBuffer(body))
		if postErr != nil {
			log.Error(errors.Wrap(postErr, "failed to Post"))
		}
	}
}

func (s *Server) addInFlight(r *Result) {
	s.Lock()
	defer s.Unlock()

	s.inFlight[r.UUID()] = r
}

func (s *Server) getInFlight(id string) *Result {
	s.Lock()
	defer s.Unlock()

	r, ok := s.inFlight[id]
	if !ok {
		return nil
	}

	return r
}

func (s *Server) removeInFlight(id string) {
	s.Lock()
	defer s.Unlock()

	delete(s.inFlight, id)
}
