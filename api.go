package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GetStreamResponse struct {
	Data       []Stream `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
	Total int `json:"total"`
}

type Stream struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	UserLogin    string    `json:"user_login"`
	UserName     string    `json:"user_name"`
	GameName     string    `json:"game_name"`
	Title        string    `json:"title"`
	StartedAt    time.Time `json:"started_at"`
	ThumbnailURL string    `json:"thumbnail_url"`
}

type GetVideosResponse struct {
	Data       []Video `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
	Total int `json:"total"`
}

type Video struct {
	StreamID     string `json:"stream_id"`
	UserLogin    string `json:"user_login"`
	ThumbnailURL string `json:"thumbnail_url"`
}

func FetchStream(username, clientID, accesstoken string) (Stream, error) {
	query := url.Values{"user_login": {username}, "first": {"100"}}.Encode()
	streams, err := helixGet[GetStreamResponse]("/streams", query, clientID, accesstoken)
	if err != nil {
		return Stream{}, err
	}
	if len(streams.Data) == 0 {
		return Stream{}, fmt.Errorf("no active stream found")
	}
	return streams.Data[0], nil
}

func FetchVideo(videoID, clientID, accesstoken string) (Video, error) {
	query := url.Values{"id": {videoID}, "first": {"100"}}.Encode()
	videos, err := helixGet[GetVideosResponse]("/videos", query, clientID, accesstoken)
	if err != nil {
		return Video{}, err
	}
	if len(videos.Data) == 0 {
		return Video{}, fmt.Errorf("no video found")
	}
	return videos.Data[0], nil
}

func helixGet[T any](path, query, clientID, accesstoken string) (T, error) {

	var responseJSON T

	url := "https://api.twitch.tv/helix"+path

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return responseJSON, err
	}

	req.URL.RawQuery = query

	req.Header.Set("Client-Id", clientID)
	req.Header.Set("Authorization", "Bearer "+accesstoken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return responseJSON, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return responseJSON, err
	}

	if resp.StatusCode != http.StatusOK {
		return responseJSON, fmt.Errorf("unexpected statuscode %s: %s", resp.Status, b)
	}

	err = json.Unmarshal(b, &responseJSON)
	if err != nil {
		return responseJSON, fmt.Errorf("failed to parse response as JSON: %s (%s)", resp.Status, b)
	}

	return responseJSON, nil
}

func GenerateToken(clientID, clientSecret string) (string, error) {
	req, err := http.NewRequest(http.MethodPost, "https://id.twitch.tv/oauth2/token", strings.NewReader(url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"client_credentials"},
	}.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected statuscode %s: %s", resp.Status, b)
	}

	var tokenresponse struct {
		AccessToken string `json:"access_token"`
	}
	err = json.Unmarshal(b, &tokenresponse)
	if err != nil {
		return "", fmt.Errorf("failed to parse response body as JSON: %s (%s)", err, b)
	}

	return tokenresponse.AccessToken, nil
}
