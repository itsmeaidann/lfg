package http

import (
	"bytes"
	"io"
	"net/http"
)

func GetRequest(url string, token string) (status string, resBody []byte, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", nil, err
	}

	// set headers
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// send request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	resBody, err = io.ReadAll(res.Body)
	if err != nil {
		return "", nil, err
	}
	return res.Status, resBody, nil
}

func PostRequest(url string, token string, reqBody []byte) (status string, resBody []byte, err error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", nil, err
	}

	// set headers
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// send request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	resBody, err = io.ReadAll(res.Body)
	if err != nil {
		return "", nil, err
	}
	return res.Status, resBody, nil
}
