package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strings"
)

type StreamInfo struct {
	Username  string
	StreamID  string
	StartTime string
	Thumbnail string
	URL       string
}

type YTDLPStreamInfo struct {
	ID         string `json:"id"`
	UploaderID string `json:"uploader_id"`
	Timestamp  int    `json:"timestamp"`
	Thumbnail  string `json:"thumbnail"`
	URL        string `json:"url"`
	Version    struct {
		Version string `json:"version"`
	} `json:"_version"`
}

func ParseInputURL(raw string) (path string, isVideo bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return path, isVideo, fmt.Errorf("empty input URL")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return path, isVideo, fmt.Errorf("invalid URL: %s", err)
	}

	if !parsed.IsAbs() {
		return path, isVideo, fmt.Errorf("url needs to include schema (https://)")
	}

	path = strings.Trim(parsed.Path, "/")
	if path == "" {
		return path, isVideo, fmt.Errorf("no channel name or video ID in URL: %s", raw)
	}

	path, isVideo = strings.CutPrefix(path, "videos/")
	return
}

func ResolveStreamInfo(cfg config) (StreamInfo, error) {

	switch {
	case cfg.username != "" && cfg.streamid != "":
		return fromFlags(cfg)
	case cfg.clientID != "" && cfg.clientSecret != "":
		return fromTwitchAPI(cfg)
	case cfg.inputURL != "":
		return fromYTDLP(cfg.inputURL)
	default:
		return StreamInfo{}, fmt.Errorf(
			"provide a twitch URL, --clientid and --clientsecret, or --username/--streamid/--starttime",
		)
	}
}

func fromFlags(cfg config) (StreamInfo, error) {
	if cfg.bruteforce == "" && cfg.starttime == "" {
		return StreamInfo{}, fmt.Errorf("starttime flag is required, try using --bruteforce min-max if you know the rough start time of the stream")
	}
	return StreamInfo{
		Username:  cfg.username,
		StreamID:  cfg.streamid,
		StartTime: cfg.starttime,
	}, nil
}

func fromTwitchAPI(cfg config) (StreamInfo, error) {

	path, isVideo, err := ParseInputURL(cfg.inputURL)
	if err != nil {
		return StreamInfo{}, err
	}

	log.Println("twitch api client ID and secret provided, generating access token")
	token, err := GenerateToken(cfg.clientID, cfg.clientSecret)
	if err != nil {
		return StreamInfo{}, fmt.Errorf("failed to generate access token: %s", err)
	}

	if isVideo {
		log.Println("fetching info from twitch api for video ", path)
		video, err := FetchVideo(path, cfg.clientID, token)
		if err != nil {
			return StreamInfo{}, fmt.Errorf("failed to get info about video: %s", err)
		}

		if !strings.Contains(video.ThumbnailURL, "cf_vods") {
			return StreamInfo{}, fmt.Errorf("cannot extract playlist from vod when using api and the stream it belongs to is still live")
		}

		return StreamInfo{
			Username:  video.UserLogin,
			StreamID:  video.StreamID,
			Thumbnail: video.ThumbnailURL,
		}, nil
	} else {
		log.Println("fetching info from twitch api for user ", path)
		stream, err := FetchStream(path, cfg.clientID, token)
		if err != nil {
			return StreamInfo{}, fmt.Errorf("failed to get info about current stream: %s", err)
		}
		return StreamInfo{
			Username:  stream.UserLogin,
			StreamID:  stream.ID,
			StartTime: fmt.Sprint(stream.StartedAt.Unix()),
		}, nil
	}

}

func fromYTDLP(twitchURL string) (StreamInfo, error) {
	log.Println("extracting stream info with yt-dlp")
	info, err := ExtractWithYTDLP(twitchURL)
	if err != nil {
		return StreamInfo{}, fmt.Errorf(
			"yt-dlp extraction failed: %s; provide --clientid and --clientsecret, or use --username/--streamid/--starttime to skip extraction",
			err,
		)
	}
	return info, nil
}

func ExtractWithYTDLP(twitchURL string) (StreamInfo, error) {
	cmd := exec.Command("yt-dlp", "--no-download", "--dump-single-json", twitchURL)
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return StreamInfo{}, fmt.Errorf("failed to execute yt-dlp: %s: %s", err, stderr.String())
	}

	var data YTDLPStreamInfo
	if err := json.Unmarshal(out, &data); err != nil {
		return StreamInfo{}, fmt.Errorf("failed to parse yt-dlp output: %s: %s", err, stderr.String())
	}

	info := StreamInfo{
		Username:  data.UploaderID,
		StreamID:  data.ID,
		StartTime: fmt.Sprint(data.Timestamp),
		URL:       data.URL,
	}

	return info, nil
}
