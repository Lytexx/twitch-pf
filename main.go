package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

const VERSION = "0.1"

var domains = []string{
	"https://d1m7jfoe9zdc1j.cloudfront.net",
	"https://d2vi6trrdongqn.cloudfront.net",
	"https://d2nvs31859zcd8.cloudfront.net",
	"https://d3fi1amfgojobc.cloudfront.net",
	"https://dgeft87wbj63p.cloudfront.net",
	"https://d3stzm2eumvgb4.cloudfront.net",
	"https://d3vd9lfkzbru3h.cloudfront.net",
}

func main() {

	log.SetOutput(os.Stderr)
	http.DefaultClient.Timeout = 10 * time.Second

	cfg, err := parseConfig()
	if err != nil {
		log.Fatalln(err)
	}
	if err := run(cfg); err != nil {
		log.Fatalln(err)
	}
}
