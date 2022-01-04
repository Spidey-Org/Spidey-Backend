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
	cachedData  *InviteData
	generated   time.Time

	httpClient = &http.Client{
		Timeout: time.Second * 10,
	}
)

func main() {
	serveMux := http.NewServeMux()
	serveMux.Handle("/invite", &InviteHandler{})
	serveMux.Handle("/invite_redirect", &RedirectHandler{})

	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: serveMux,
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Println("error while listening & serving: ", err)
	}
}

type InviteHandler struct{}

type RedirectHandler struct{}

type Application struct {
	ID            string `json:"id"`
	InstallParams struct {
		Permissions string `json:"permissions"`
	} `json:"install_params"`
}

type InviteData struct {
	response Response
	data     []byte
}

type Response struct {
	URL string `json:"url"`
}

func (i InviteHandler) ServeHTTP(wr http.ResponseWriter, _ *http.Request) {
	inviteData, err := getInviteUrl(wr)
	if err == nil {
		wr.Header().Set("Content-Type", "application/json")
		_, err := wr.Write(inviteData.data)
		if err != nil {
			log.Println("there was an error while writing the response", err)
			http.Error(wr, "there was an error while writing the response", http.StatusInternalServerError)
		}
	}
}

func (r RedirectHandler) ServeHTTP(wr http.ResponseWriter, rq *http.Request) {
	inviteData, err := getInviteUrl(wr)
	if err == nil {
		http.Redirect(wr, rq, inviteData.response.URL, http.StatusPermanentRedirect)
	}
}

func getInviteUrl(wr http.ResponseWriter) (*InviteData, error) {
	if cachedData == nil || time.Since(generated) > cacheTime {
		application, err := getApplication()
		if err != nil {
			log.Println("there was an error while getting the application", err)
			http.Error(wr, "there was an error while getting the application", http.StatusInternalServerError)
			return nil, err
		}
		response := Response{
			URL: fmt.Sprintf(inviteURLPattern, application.ID, application.InstallParams.Permissions),
		}
		marshalled, err := json.Marshal(response)
		if err != nil {
			log.Println("there was an error while marshalling the data", err)
			http.Error(wr, "there was an error while marshalling the data", http.StatusInternalServerError)
			return nil, err
		}
		cachedData = &InviteData{
			data:     marshalled,
			response: response,
		}
		generated = time.Now()
	}
	return cachedData, nil
}

func getApplication() (application Application, err error) {
	request, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return
	}
	request.Header.Set("Authorization", spideyToken)
	response, err := httpClient.Do(request)
	if err != nil {
		return
	}
	statusCode := response.StatusCode
	if statusCode != 200 {
		err = fmt.Errorf("received code %d while running the invite request", statusCode)
		return
	}
	body := response.Body
	defer body.Close()
	err = json.NewDecoder(body).Decode(&application)
	return
}
