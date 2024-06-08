package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func getClientId() string {
	return os.Getenv("SPOTIFY_CLIENT_ID")
}

func getClientSecret() string {
	return os.Getenv("SPOTIFY_CLIENT_SECRET")
}

func getClientToken() string {
	clientId := getClientId()
	clientSecret := getClientSecret()
	token := fmt.Sprintf("%s:%s", clientId, clientSecret)
	return base64.StdEncoding.EncodeToString([]byte(token))
}

func getToken() string {
	body := url.Values{
		"grant_type": {"client_credentials"},
	}

	request, err := http.NewRequest("POST",
		"https://accounts.spotify.com/api/token",
		strings.NewReader(body.Encode()),
	)

	if err != nil {
		log.Fatal(err)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Authorization", fmt.Sprintf("Basic %s", getClientToken()))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(response.Body)

	var result map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&result)

	if response.StatusCode != http.StatusOK {
		log.Fatalf("error %d: %s", response.StatusCode, result)
	}

	if err != nil {
		log.Fatal(err)
	}

	return result["access_token"].(string)
}

func main() {
	token := getToken()
	UpdateCache(token)
}
