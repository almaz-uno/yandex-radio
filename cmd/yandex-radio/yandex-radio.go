package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/zserge/lorca"
)

func main() {
	if err := doMain(); err != nil {
		log.Fatal(err)
	}
}

const songInfoFrequency = time.Second

func doMain() error {
	hd, err := homedir.Dir()
	if err != nil {
		logrus.WithError(err).Warn("Unable to acquire home dir. Using current one.")
		hd = "."
	}

	ui, err := lorca.New("https://radio.yandex.ru/", hd+"/.yandex-audio", 800, 600)
	if err != nil {
		log.Fatal(err)
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
		songInfoFile := hd + "/.current-song"
		_ = songInfoFile
		ticker := time.Tick(songInfoFrequency)
		for range ticker {
			tt := ui.Eval(`document.querySelector('.player-controls__title').title`)
			ta := ui.Eval(`document.querySelector('.player-controls__artists').title`)

			trackInfo := fmt.Sprintf("%s • %s", tt, ta)
			err := ioutil.WriteFile(songInfoFile, []byte(trackInfo), 0o664)
			if err != nil {
				logrus.WithError(err).Warn("Unable to save song info file")
			}
			logrus.WithFields(logrus.Fields{
				"tt":        tt,
				"ta":        ta,
				"trackInfo": trackInfo,
			}).Debug("Track info was got.")
		}
	}()

	<-ui.Done()
	return nil
}
