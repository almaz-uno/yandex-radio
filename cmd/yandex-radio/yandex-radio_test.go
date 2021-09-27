package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_extractAlbumID(t *testing.T) {
	assert.Equal(t, "3245869", albumFromCover.FindStringSubmatch("url(\"//avatars.yandex.net/get-music-content/38044/be6bcf33.a.3245869-1/600x600\")")[1])
}
