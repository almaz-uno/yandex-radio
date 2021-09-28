package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_extractAlbumID(t *testing.T) {
	s := "url(\"//avatars.yandex.net/get-music-content/38044/be6bcf33.a.3245869-1/600x600\")"
	if assert.True(t, albumFromCover.MatchString(s)) {
		assert.Equal(t, "3245869", albumFromCover.FindStringSubmatch(s)[1])
	}
	s = `url("//avatars.yandex.net/get-music-content/95061/74858178.a.216924-2/600x600")`
	if assert.True(t, albumFromCover.MatchString(s)) {
		assert.Equal(t, "216924", albumFromCover.FindStringSubmatch(s)[1])
	}
}
