package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	apiURL           = "https://discord.com/api/v9/oauth2/applications/@me"
	inviteURLPattern = "https://discord.com/oauth2/authorize?client_id=%s&permissions=%s&scope=bot+applications.commands"
	cacheTime        = time.Minute * 10
)

var (
	spideyToken = os.Getenv("spidey_token")
	serverPort  = "7777"

	httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
)

func main() {
	serveMux := http.NewServeMux()
	serveMux.Handle("/invite", &InviteHandler{})

	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: serveMux,
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Println("error while listening & serving: ", err)
	}
}

type InviteHandler struct {
	cachedResponse []byte
	generated      time.Time
}

func (i *InviteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if i.cachedResponse == nil || time.Since(i.generated) > cacheTime {
		application, err := getApplicationMe(w)
		if err != nil {
			log.Println("error while getting application:", err)
			http.Error(w, "error while getting application", http.StatusInternalServerError)
			return
		}
		response, err := json.Marshal(Response{
			URL: fmt.Sprintf(inviteURLPattern, application.ID, application.InstallParams.Permissions),
		})
		if err != nil {
			log.Println("error while marshalling json: ", err)
			http.Error(w, "error while marshalling json", http.StatusInternalServerError)
			return
		}
		i.cachedResponse = response
		i.generated = time.Now()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(i.cachedResponse)
	if err != nil {
		log.Println("error while writing response: ", err)
	}
}

type Response struct {
	URL string `json:"url"`
}

type Application struct {
	ID            string `json:"id"`
	InstallParams struct {
		Permissions string `json:"permissions"`
	} `json:"install_params"`
}

func getApplicationMe(w http.ResponseWriter) (application Application, err error) {
	var rq *http.Request
	rq, err = http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return
	}
	rq.Header.Set("Authorization", spideyToken)
	var rs *http.Response
	rs, err = httpClient.Do(rq)
	if err != nil {
		return
	}
	statusCode := rs.StatusCode
	if statusCode != 200 {
		err = fmt.Errorf("received code %d while running the invite request", statusCode)
		return
	}
	defer rs.Body.Close()
	err = json.NewDecoder(rs.Body).Decode(&application)
	return
}
