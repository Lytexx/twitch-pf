Twitch-pf is a CLI that tries to find DVR playlists for twitch livestreams and videos

## Usage

extract info using yt-dlp and print url to terminal
```
twitch-pf https://twitch.tv/kiara
```

using the official twitch api to extract info
```
twitch-pf --clientid f2nz3nobs6on450y... --clientsecret 9cgif2max8p3o5x... https://twitch.tv/kiara
```

providing the required information manually
```
twitch-pf --username raora --streamid 317775607911 --starttime 1784099101
```

bruteforce the start time if you don't know exactly when the stream started
```
twitch-pf --username raora --streamid 317775607911 --bruteforce 1784099100-1784099200
```

if a stream had muted segments, then the DVR playlist will contain segments that are named `unmuted` but are not accessible.
to get around this, you can give twitch-pf a cloudfront url as input, and it will rewrite it with muted segments and absolute URLs.
```
twitch-pf https://d3stzm2eumvgb4.cloudfront.net/8e2958fbbafed0d3e...
ffmpeg -protocol_whitelist file,https,tls,tcp -i rewritten-index-dvr.m3u8 -c copy output.ts
```


## Flags:

| Flag | Description |
|------|-------------|
| `--clientid <id>` | Twitch API client ID for info extraction |
| `--clientsecret <secret>` | Twitch API client secret for info extraction |
| `--username <name>` | Streamers username |
| `--streamid <id>` | Stream ID |
| `--starttime <unix>` | Stream start time as unix timestamp |
| `--bruteforce <min-max>` | Try playlist paths for every start time in the given unix timestamp range (example: 1784000100-1784000200) |
| `--ignoreconfig` | Prevent the config file from being loaded |

## Config
the twitch api client ID and secret can also be loaded from a config file.
flags take precedence over the config file.

| OS | Config file location |
|------|-------------|
| Windows | `%AppData%\twitch-pf\config.json` |
| Linux | `~/.config/twitch-pf/config.json` |
| macOS | `~/Library/Application Support/twitch-pf/config.json` |

```json
{
  "client_id": "f2nz3nobs6on450y...",
  "client_secret": "9cgif2max8p3o5x..."
}
```

### Dependencies

[YT-DLP](https://github.com/yt-dlp/yt-dlp) is required if no twitch api access is provided

### Caveats
 - Streamers can choose to have specific audio tracks only appear live, these won't be present in DVR playlists: https://obsproject.com/kb/twitch-vod-track-guide
 - DVR playlists can be used even if a streamer has VOD disabled, however if they have VOD enabled and then delete a vod, then that vod will have their DVR playlist deleted as well.