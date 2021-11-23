package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	s = `url("//avatars.yandex.net/get-music-content/163479/d74a7f44.a.7998579-1/600x600")`
	if assert.True(t, albumFromCover.MatchString(s)) {
		assert.Equal(t, "7998579", albumFromCover.FindStringSubmatch(s)[1])
	}
}

func Test_getInfo(t *testing.T) {
	al, err := getAlbumInfo("4029708")
	require.NoError(t, err)

	fmt.Printf("%+v \n", al)
}
