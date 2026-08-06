package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type config struct {
	bruteforce   string
	clientID     string
	clientSecret string
	streamid     string
	username     string
	starttime    string
	inputURL     string
	ignoreConfig bool
	printversion bool
}

func parseConfig() (config, error) {
	var cfg config
	flag.BoolVar(&cfg.printversion, "version", false, "print the current version")
	flag.StringVar(&cfg.bruteforce, "bruteforce", "",
		"try to find a playlist by trying all possible playlists between {min}-{max} (example: 1784000100-1784000200)")
	flag.StringVar(&cfg.clientID, "clientid", "", "client ID for the twitch api")
	flag.StringVar(&cfg.clientSecret, "clientsecret", "", "client secret for the twitch api")
	flag.StringVar(&cfg.streamid, "streamid", "", "the streams ID")
	flag.StringVar(&cfg.username, "username", "", "the username of the streamer")
	flag.StringVar(&cfg.starttime, "starttime", "", "time the stream started at as unix timestamp (example: 1786021094)")
	flag.BoolVar(&cfg.ignoreConfig, "ignoreconfig", false, "do not load the config file")
	flag.Parse()

	if cfg.printversion {
		log.Fatalln("current version:", VERSION)
	}

	args := flag.Args()
	if len(args) > 0 {
		cfg.inputURL = args[0]
	}

	if !cfg.ignoreConfig {
		fc, err := loadConfigFile()
		if err != nil {
			return config{}, err
		}
		if cfg.clientID == "" {
			cfg.clientID = fc.ClientID
		}
		if cfg.clientSecret == "" {
			cfg.clientSecret = fc.ClientSecret
		}
	}
	return cfg, nil
}

type fileConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func loadConfigFile() (fileConfig, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return fileConfig{}, nil
	}

	path := filepath.Join(dir, "twitch-pf", "config.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fileConfig{}, nil
		}
		return fileConfig{}, fmt.Errorf("failed to read config file %s: %s", path, err)
	}

	var fc fileConfig
	if err := json.Unmarshal(b, &fc); err != nil {
		return fileConfig{}, fmt.Errorf("failed to parse config file %s: %s", path, err)
	}
	return fc, nil
}

func run(cfg config) error {

	if parsed, err := url.Parse(cfg.inputURL); err == nil && strings.HasSuffix(parsed.Host, "cloudfront.net") {
		return Rewrite(cfg.inputURL, parsed)
	}

	info, err := ResolveStreamInfo(cfg)
	if err != nil {
		return err
	}

	playlistURL, content, err := fetchPlaylist(cfg, info)
	if err != nil {
		return err
	}

	if strings.Contains(content, "muted") {
		log.Println("warning: playlist contains muted segments")
	}

	fmt.Println(playlistURL.String())
	return nil
}

func parseBruteforceRange(s string) (min, max int, err error) {
	minStr, maxStr, found := strings.Cut(s, "-")
	if !found || minStr == "" || maxStr == "" {
		return 0, 0, fmt.Errorf("invalid bruteforce range, expected format: 1784030000-1784040000")
	}

	min, err = strconv.Atoi(minStr)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse min value (%s) as int: %s", minStr, err)
	}
	max, err = strconv.Atoi(maxStr)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse max value (%s) as int: %s", maxStr, err)
	}

	if max <= min {
		return 0, 0, fmt.Errorf("max (%d) needs to be bigger than min (%d)", max, min)
	}

	return min, max, nil
}
