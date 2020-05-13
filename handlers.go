package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/microlib/simple"
)

const (
	CONTENTTYPE     string = "Content-Type"
	APPLICATIONJSON string = "application/json"
)

var (
	payload ProjectDetail
)

// Response schema
type Response struct {
	Name       string     `json:"name"`
	StatusCode string     `json:"statuscode"`
	Status     string     `json:"status"`
	Message    string     `json:"message"`
	Payload    []Pipeline `json:"payload"`
}

type Repository struct {
	Name     string `json:"name"`
	MetaInfo string `json:"metainfo"`
	WorkDir  string `json:"workdir"`
	Path     string `json:"path"`
	Scm      string `json:"scm"`
	RawUrl   string `json:"cicd-raw-url"`
	Skip     bool   `json:"skip"`
	Force    bool   `json:"force"`
}

type ProjectDetail struct {
	Project      string       `json:"project"`
	Repositories []Repository `json:"repositories"`
}

type Pipeline struct {
	Project    string        `json:"project"`
	Scm        string        `json:"scm"`
	Workdir    string        `json:"workdir"`
	Force      bool          `json:"force"`
	Stages     []StageDetail `json:"stages"`
	LastUpdate int64         `json:"lastupdate,omitempty"`
	MetaInfo   string        `json:"metainfo,omitempty"`
}

type StageDetail struct {
	Id       int           `json:"id"`
	Name     string        `json:"name"`
	Exec     string        `json:"exec"`
	Wait     int           `json:"wait"`
	Service  string        `json:"service"`
	Replicas int           `json:"replicas"`
	Skip     bool          `json:"skip"`
	Envars   []EnvarDetail `json:"envars"`
	Commands []string      `json:"commands"`
}

type EnvarDetail struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func JsonHandler(w http.ResponseWriter, r *http.Request, logger *simple.Logger) {
	var response Response

	addHeaders(w, r)

	pipelines, err := buildSchema(logger)
	if err != nil {
		response = Response{Name: os.Getenv("NAME"), StatusCode: "500", Status: "KO", Message: "Error buildSchema", Payload: pipelines}
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response = Response{Name: os.Getenv("NAME"), StatusCode: "200", Status: "OK", Message: "Payload buildSchema succeeded", Payload: pipelines}
		w.WriteHeader(http.StatusOK)
	}

	b, _ := json.MarshalIndent(response, "", "	")
	logger.Debug(fmt.Sprintf("JsonHandler response : %s", string(b)))
	fmt.Fprintf(w, string(b))
}

func buildSchema(logger *simple.Logger) ([]Pipeline, error) {
	var pipeline Pipeline
	var pipelines []Pipeline

	file, err := ioutil.ReadFile("project.json")
	if err != nil {
		logger.Error(fmt.Sprintf("Reading project.json %v", err))
		return pipelines, err
	}
	err = json.Unmarshal(file, &payload)
	if err != nil {
		logger.Error(fmt.Sprintf("Unmarshalling project.json %v", err))
		return pipelines, err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	for x, _ := range payload.Repositories {
		req, _ := http.NewRequest("GET", payload.Repositories[x].RawUrl, nil)
		req.Header.Set("X-Api-Key", os.Getenv("APIKEY"))
		req.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Error(fmt.Sprintf("Http request %v", err))
			continue
		}
		defer resp.Body.Close()
		body, e := ioutil.ReadAll(resp.Body)
		if e != nil {
			logger.Error(fmt.Sprintf("Could not read cicd.json file %v", e))
			continue
		}
		err = json.Unmarshal(body, &pipeline)
		if err != nil {
			logger.Error(fmt.Sprintf("Unmarshalling project.json %v", err))
			continue
		}
		pipelines = append(pipelines, pipeline)
	}
	return pipelines, nil
}

func IsAlive(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "{ \"version\" : \""+os.Getenv("VERSION")+"\" , \"name\": \""+os.Getenv("NAME")+"\" }")
}

// headers (with cors) utility
func addHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(CONTENTTYPE, APPLICATIONJSON)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}
