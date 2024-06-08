package main

import "time"

type Playlist struct {
	id   string
	name string
}

type Artist struct {
	id   string
	name string
}

type Album struct {
	id          string
	name        string
	releaseDate time.Time
}

type Track struct {
	id         string
	name       string
	artists    []Artist
	album      Album
	previewUrl string
}
