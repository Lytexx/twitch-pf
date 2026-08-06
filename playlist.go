package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func ResolvePlaylist(path string) (chunkedURL *url.URL, content string, err error) {

	for _, domain := range domains {

		chunkedURL, err = url.Parse(fmt.Sprintf("%s/%s/chunked/index-dvr.m3u8", domain, path))
		if err != nil {
			continue
		}

		content, err = getPlaylist(chunkedURL.String())
		if err != nil {
			continue
		}

		return chunkedURL, content, nil
	}
	return nil, "", errors.New("no valid url found")
}

// format is {hash}_{username}_{streamid}_{starttime} where {hash}
// is the first 20 characters of the hex encoded sha1 of {username}_{streamid}_{starttime}
// Source: https://github.com/TwitchRecover/TwitchRecover/blob/ebf0bd413216e6ddcba72e9947b9cadd3110fe6d/src/TwitchRecover.Core/Compute.java#L83
func ComputePlaylistPath(username string, streamid string, starttime string) string {
	username = strings.ToLower(username) // the username in playlist URLs is always lowercase
	basepath := username + "_" + streamid + "_" + starttime

	hash := sha1.New()
	hash.Write([]byte(basepath))
	hashString := hex.EncodeToString(hash.Sum(nil))

	return hashString[:20] + "_" + basepath
}

// example: https://static-cdn.jtvnw.net/cf_vods/d1m7jfoe9zdc1j/cfa63f10b50855777324_bunnyayu_320548559963_1784419024//thumb/thumb0-%{width}x%{height}.jpg
// returns: cfa63f10b50855777324_bunnyayu_320548559963_1784419024
func PathFromThumbnailURL(thumbnailURL string) string {
	parts := strings.Split(thumbnailURL, "/")
	return parts[5]
}

func getPlaylist(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return string(content), nil
}

// rewritePlaylist rewrites the playlist with absolute urls and replaces inaccessible unmuted segments with muted ones
func rewritePlaylist(content string, playlistURL *url.URL) (string, error) {

	pl := strings.Builder{}

	for line := range strings.SplitSeq(content, "\n") {

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "#") {

			line = strings.Replace(line, "unmuted", "muted", 1)

			if parsed, err := url.Parse(line); err != nil {
				return "", fmt.Errorf("failed to parse line as url: %s [%s]", err, line)
			} else {
				line = playlistURL.ResolveReference(parsed).String()
			}
		}

		if after, found := strings.CutPrefix(line, "#EXT-X-MAP:URI="); found {
			trimmed := strings.Trim(after, `"`)
			if parsed, err := url.Parse(trimmed); err != nil {
				return "", fmt.Errorf("failed to parse init segment url: %s [%s]", err, trimmed)
			} else {
				line = fmt.Sprintf(`#EXT-X-MAP:URI="%s"`, playlistURL.ResolveReference(parsed).String())
			}
		}

		pl.WriteString(line)
		pl.WriteString("\n")
	}
	return pl.String(), nil
}

func Rewrite(input string, parsed *url.URL) error {
	playlist, err := getPlaylist(input)
	if err != nil {
		return fmt.Errorf("failed to fetch playlist: %s", err)
	}

	rewritten, err := rewritePlaylist(playlist, parsed)
	if err != nil {
		return fmt.Errorf("failed to rewrite playlist: %s", err)
	}

	log.Println("writing playlist to rewritten-index-dvr.m3u8")
	if err = os.WriteFile("rewritten-index-dvr.m3u8", []byte(rewritten), 0600); err != nil {
		return fmt.Errorf("failed to write playlist to file: %s", err)
	}

	return nil
}

func Bruteforce(tsrange string, username, streamid string) (*url.URL, string, error) {
	minInt, maxInt, err := parseBruteforceRange(tsrange)
	if err != nil {
		return nil, "", err
	}

	total := maxInt-minInt
	log.Printf("starting bruteforce, trying %d paths with a total of %d requests\n", total, total*len(domains))

	for i := minInt; i <= maxInt; i++ {
		path := ComputePlaylistPath(username, streamid, fmt.Sprint(i))
		playlist, content, err := ResolvePlaylist(path)
		if err != nil {
			log.Printf("[%d/%d] trying path %s", i-minInt, total, path)
		} else {
			return playlist, content, nil
		}
	}

	return nil, "", fmt.Errorf("no valid playlist found")

}
