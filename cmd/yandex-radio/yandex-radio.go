package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/TheCreeper/go-notify"
	"github.com/kr/pretty"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/zserge/lorca"
)

const (
	songInfoFrequency   = time.Second
	defaultURL          = "https://radio.yandex.ru/"
	apiURL              = "https://api.music.yandex.net/"
	trackApiURL         = apiURL + "tracks/"
	albumApiURL         = apiURL + "albums/"
	additionalInfoClass = "__additional_info"
)

var errTrackNotFound = errors.New("track not found")

var (
	logLevel    = pflag.StringP("log-level", "L", "info", "logging level (trace, debug, info, warn, error, fatal)")
	notifyDelay = pflag.Int32P("notify-delay", "d", 5000, "notification dismiss timer delay, milliseconds")
)

type (
	artist struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Cover struct {
			URI string `json:"uri"`
		} `json:"cover"`
	}

	album struct {
		ID       int      `json:"id"`
		Title    string   `json:"title"`
		Year     int      `json:"year"`
		CoverURI string   `json:"coverUri"`
		Artists  []artist `json:"artists"`
	}

	track struct {
		ID      string   `json:"id"`
		Title   string   `json:"title"`
		Artists []artist `json:"artists"`
		Albums  []album  `json:"albums"`
	}
)

func (t *track) String() string {
	if t == nil {
		return ""
	}

	return pretty.Sprint(t)
}

var albumFromCover = regexp.MustCompile(`^url\("//avatars.yandex.net/get-music-content/\d+/.+\.a\.(\d+)-\d+/600x600"\)$`)

func main() {
	pflag.Parse()
	if lvl, err := logrus.ParseLevel(*logLevel); err == nil {
		logrus.SetLevel(lvl)
	} else {
		logrus.WithError(err).Warnf("Unable to parse log level")
	}

	if err := doMain(pflag.Args()); err != nil {
		log.Fatal(err)
	}
}

func doMain(args []string) error {
	hd, err := homedir.Dir()
	if err != nil {
		logrus.WithError(err).Warn("Unable to acquire home dir. Using current one.")
		hd = "."
	}

	url := defaultURL
	if len(args) > 0 {
		url = args[0]
	}

	ui, err := lorca.New(url, hd+"/.yandex-radio", 800, 600)
	if err != nil {
		logrus.WithError(err).Fatal("Fatal error")
	}
	defer ui.Close()

	// https://api.music.yandex.net/albums/12983383/with-tracks
	// https://api.music.yandex.net/albums/12983383/with-tracks
	// https://api.music.yandex.net/tracks/13087591

	go func() {
		var (
			playingNow   = ""
			songInfoFile = hd + "/.current-song"
			songImgFile  = hd + "/.current-song-img.png"
		)
		ticker := time.Tick(songInfoFrequency)
		for range ticker {
			trackIDVal := ui.Eval(`document.querySelector('.player-controls__bar button.like').attributes['data-idx'].value`)

			tt := ui.Eval(`document.querySelector('.player-controls__title').title`)
			ta := ui.Eval(`document.querySelector('.player-controls__artists').title`)
			ti := ui.Eval(`document.querySelector('.slider__item_playing div span').style.backgroundImage`)

			albumID := ""
			if albumFromCover.MatchString(ti.String()) {
				albumID = albumFromCover.FindStringSubmatch(ti.String())[1]
			}

			trackInfo := fmt.Sprintf("%s • %s", tt, ta)
			err := os.WriteFile(songInfoFile, []byte(trackInfo), 0o664)
			if err != nil {
				logrus.WithError(err).Warn("Unable to save song info file")
			}
			logrus.WithFields(logrus.Fields{
				"trackID":   trackIDVal,
				"albumID":   albumID,
				"tt":        tt,
				"ta":        ta,
				"trackInfo": trackInfo,
				"ti":        ti,
			}).Debug("Track info was got.")

			var (
				albumInfo       = ""
				existsAlbumInfo = ui.Eval(`document.querySelector('.slider__item_playing div span').title`).String()
				existsArtist    = ui.Eval(`document.querySelector('.player-controls__artists').innerText`).String()
			)

			if existsAlbumInfo == "" || existsArtist == ta.String() {
				if len(albumID) > 0 {

					al, err := getAlbumInfo(albumID)
					if err != nil {
						logrus.WithError(err).Warnf("Unable to acquire album info for album %s", albumID)
					} else {
						logrus.WithField("tr", al).Info("Album playing now")

						albumInfo = fmt.Sprintf("%s (%d)", al.Title, al.Year)

						ui.Eval(`document.querySelector('.slider__item_playing div span').title="` + albumInfo + `"`)
						ui.Eval(`document.querySelector('.player-controls__artists').innerText='` + ta.String() + `\n\n` + albumInfo + `'`)
					}
				}
			}

			if playingNow != "" && playingNow != trackInfo {

				// https://developer.gnome.org/notification-spec/

				ntf := notify.NewNotification("Yandex Radio", fmt.Sprintf("<b>%s</b>\r%s\r%s",
					html.EscapeString(tt.String()),
					html.EscapeString(ta.String()),
					html.EscapeString(albumInfo)),
				)
				ntf.Timeout = *notifyDelay
				ntf.Hints = make(map[string]interface{})
				ntf.Actions = []string{"dismiss", "Dismiss"}

				if ti != nil && len(ti.String()) > 7 {
					tiUrl := ti.String()[5 : len(ti.String())-2]
					logrus.Tracef("tiUrl='%s'", tiUrl)
					err = updateImageFile(tiUrl, songImgFile)
					if err != nil {
						logrus.WithError(err).Warn("Unable to get album img")
					} else {
						ntf.Hints[notify.HintImagePath] = songImgFile
					}
				}

				_, err = ntf.Show()
				if err != nil {
					logrus.WithError(err).Warn("Unable to show notification")
				}

			}
			playingNow = trackInfo
		}
	}()

	<-ui.Done()
	return nil
}

// download album img
func updateImageFile(url, imgFile string) error {
	res, err := http.Get("http:" + url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	bb, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = os.WriteFile(imgFile, bb, 0o644)
	if err != nil {
		return err
	}

	return nil
}

func getAlbumInfo(albumID string) (*album, error) {
	albumURL := albumApiURL + albumID
	res, err := http.Get(albumURL)
	if err != nil {
		return nil, err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	alinf := new(struct {
		InvInfo json.RawMessage `json:"invocationInfo"`
		Result  *album          `json:"result"`
	})

	bb, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	logrus.WithFields(logrus.Fields{
		"albumURL": albumURL,
		"json":     string(bb),
	}).Trace("Got album info")

	err = json.Unmarshal(bb, alinf)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal response for album %s: %v", albumID, err)
	}

	return alinf.Result, nil
}

func getTrackInfo(trackID string) (*track, error) {
	trackURL := trackApiURL + trackID
	res, err := http.Get(trackURL)
	if res.Body != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	trinf := new(struct {
		Result []*track `json:"result"`
	})

	bb, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	logrus.WithFields(logrus.Fields{
		"trackURL": trackURL,
		"json":     string(bb),
	}).Trace("Got track info")

	err = json.Unmarshal(bb, trinf)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal response for track %s: %v", trackID, err)
	}

	if len(trinf.Result) == 0 {
		return nil, fmt.Errorf("unable to get info for track %s: %v", trackID, errTrackNotFound)
	}

	return trinf.Result[0], nil
}

// func commitNotifyAction(ui lorca.UI, a string) {
// 	switch a {
// 	case "default":

// 	default:
// 		logrus.Warnf("Invalid action %s", a)
// 	}
// }
