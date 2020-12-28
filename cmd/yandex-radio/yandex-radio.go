package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/TheCreeper/go-notify"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/zserge/lorca"
)

// "github.com/mqu/go-notify"
// 	"github.com/TheCreeper/go-notify"

const (
	notifyDelay       = int32(5000) // mills
	songInfoFrequency = time.Second
)

func main() {
	logrus.SetLevel(logrus.TraceLevel)
	if err := doMain(); err != nil {
		log.Fatal(err)
	}
}

func doMain() error {
	hd, err := homedir.Dir()
	if err != nil {
		logrus.WithError(err).Warn("Unable to acquire home dir. Using current one.")
		hd = "."
	}

	ui, err := lorca.New("https://radio.yandex.ru/", hd+"/.yandex-radio", 800, 600)
	if err != nil {
		logrus.WithError(err).Fatal("Fatal error")
	}
	defer ui.Close()

	// go func() {
	// 	ticker := time.Tick(5 * time.Second)
	// 	for range ticker {
	// 		logrus.Info("Tick was arrived!")
	// 		v := ui.Eval(`
	// 		let playPauseBtn = document.querySelector('.player-controls__play');
	// 		console.log(playPauseBtn);
	// 		if(playPauseBtn){
	// 			playPauseBtn.click();
	// 		}
	// 		`)
	// 		logrus.WithField("v", v).Info("Evaluated.")
	// 		// v.Err().Error()
	// 	}
	// }()

	go func() {
		var (
			playingNow   = ""
			songInfoFile = hd + "/.current-song"
			songImgFile  = hd + "/.current-song-img.png"
		)
		_ = songInfoFile
		ticker := time.Tick(songInfoFrequency)
		for range ticker {
			tt := ui.Eval(`document.querySelector('.player-controls__title').title`)
			ta := ui.Eval(`document.querySelector('.player-controls__artists').title`)
			ti := ui.Eval(`document.querySelector('.slider__item_playing div span').style.backgroundImage`)

			trackInfo := fmt.Sprintf("%s • %s", tt, ta)
			err := ioutil.WriteFile(songInfoFile, []byte(trackInfo), 0o664)
			if err != nil {
				logrus.WithError(err).Warn("Unable to save song info file")
			}
			logrus.WithFields(logrus.Fields{
				"tt":        tt,
				"ta":        ta,
				"trackInfo": trackInfo,
				"ti":        ti,
			}).Debug("Track info was got.")
			if playingNow != "" && playingNow != trackInfo {
				//https://developer.gnome.org/notification-spec/
				ntf := notify.NewNotification("Yandex Radio", trackInfo)
				ntf.Timeout = notifyDelay
				func() {
					if ti != nil && len(ti.String()) > 7 {

						tiUrl := ti.String()[5 : len(ti.String())-2]
						res, e2 := http.Get("http:" + tiUrl)
						if e2 != nil {
							logrus.WithError(e2).Warn("Unable to get song img")
						}
						defer res.Body.Close()
						bb, e2 := ioutil.ReadAll(res.Body)
						if e2 != nil {
							logrus.WithError(e2).Warn("Unable to get song img")
						}

						e2 = ioutil.WriteFile(songImgFile, bb, 0o644)
						if e2 != nil {
							logrus.WithError(e2).Warn("Unable to save song img")
						}

						logrus.Tracef("tiUrl='%s'", tiUrl)
						ntf.Hints = make(map[string]interface{})
						ntf.Hints[notify.HintImagePath] = songImgFile
					}
				}()
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
