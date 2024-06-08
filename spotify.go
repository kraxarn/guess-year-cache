package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

var playlists = []Playlist{
	{"1y8GwyganCtgF0XqsCHkaw", "HITSTER (Eng)"},
	{"5mQVbkcILLiU2aqVOplsMy", "HITSTER (Swe)"},
	{"2fiSqo8purGffbAeTAKVwF", "HITSTER Expansion (Swe)"},
	{"79RL8j33YuRiJ7j2ywzP9L", "HITSTER (Den)"},
	{"4uWJqAGuzZ7gTgU4F1ZmAE", "HITSTER (Nor)"},
	{"6Nn768rDkJXxIrGg8CjyKL", "HITSTER (Fin)"},
}

const baseUrl string = "https://api.spotify.com/v1"

func fetch(token, path string) map[string]interface{} {
	request, err := http.NewRequest("GET", baseUrl+path, nil)
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(response.Body)

	var result map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&result)

	if response.StatusCode != http.StatusOK {
		log.Fatalf("error %d: %s", response.StatusCode, result)
	}

	if err != nil {
		log.Fatal(err)
	}

	return result
}

func GetTrackCount(token, playlistId string) int {
	url := fmt.Sprintf("/playlists/%s?fields=tracks(total)", playlistId)
	result := fetch(token, url)

	tracks := result["tracks"].(map[string]interface{})
	return int(tracks["total"].(float64))
}

func parseDate(date string) time.Time {
	var layout string

	switch len(date) {
	case 4:
		layout = "2006"
		break

	case 7:
		layout = "2006-01"
		break

	case 10:
		layout = "2006-01-02"
		break

	default:
		log.Fatalf("unknown length: %d", len(date))
	}

	value, err := time.Parse(layout, date)
	if err != nil {
		log.Fatal(err)
	}

	return value
}

func GetTracks(token, playlistId string, offset int, tracks chan Track) {
	url := fmt.Sprintf(
		"/playlists/%s/tracks?limit=50&offset=%d"+
			"&fields=items(track(id,name,preview_url,artists(id,name),album(id,name,release_date))",
		playlistId, offset,
	)
	result := fetch(token, url)

	items := result["items"].([]interface{})
	for _, item := range items {
		itemData := item.(map[string]interface{})
		trackData := itemData["track"].(map[string]interface{})

		track := Track{
			id:         trackData["id"].(string),
			name:       trackData["name"].(string),
			artists:    nil,
			album:      Album{},
			previewUrl: "",
		}

		if previewUrl, ok := trackData["preview_url"].(string); ok {
			track.previewUrl = previewUrl
		}

		artistsData := trackData["artists"].([]interface{})
		for _, artistData := range artistsData {
			artistData := artistData.(map[string]interface{})
			artist := Artist{
				id:   artistData["id"].(string),
				name: artistData["name"].(string),
			}
			track.artists = append(track.artists, artist)
		}

		albumData := trackData["album"].(map[string]interface{})
		track.album = Album{
			id:          albumData["id"].(string),
			name:        albumData["name"].(string),
			releaseDate: parseDate(albumData["release_date"].(string)),
		}

		tracks <- track
	}

	close(tracks)
}

func getDbConnectionString() string {
	return os.Getenv("DB_CONNECTION_STRING")
}

func UpdateCache(token string) {
	ctx := context.Background()
	db, err := pgx.Connect(ctx, getDbConnectionString())
	if err != nil {
		log.Fatal(err)
	}

	defer func(db *pgx.Conn, ctx context.Context) {
		err = db.Close(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}(db, ctx)

	exec := func(sql string, arguments ...any) {
		_, err = db.Exec(ctx, sql, arguments...)
		if err != nil {
			log.Fatal(err)
		}
	}

	for index, playlist := range playlists {
		fmt.Printf("[%3d/%3d] updating playlist: %s\n",
			index, len(playlists), playlist.name)

		exec("insert into playlists (id, name) values ($1, $2) on conflict (id) do nothing",
			playlist.id, playlist.name)

		trackCount := GetTrackCount(token, playlist.id)

		pageCount := int(math.Ceil(float64(trackCount) / 50.0))
		for i := 0; i < pageCount; i++ {
			fmt.Printf("[%3d/%3d] updating tracks\n", i*50, trackCount)

			tracks := make(chan Track)
			go GetTracks(token, playlist.id, i*50, tracks)

			for track := range tracks {
				exec("insert into tracks (id, name, preview_url) values ($1, $2, $3) on conflict (id) do nothing",
					track.id, track.name, track.previewUrl)

				exec("insert into playlist_tracks (playlist_id, track_id) values ($1, $2) on conflict (playlist_id, track_id) do nothing",
					playlist.id, track.id)

				for _, artist := range track.artists {
					exec("insert into artists (id, name) values ($1, $2) on conflict (id) do nothing",
						artist.id, artist.name)

					exec("insert into track_artists (track_id, artist_id) values ($1, $2) on conflict (track_id, artist_id) do nothing",
						track.id, artist.id)
				}

				exec("insert into albums (id, name, release_date) values ($1, $2, $3) on conflict (id) do nothing",
					track.album.id, track.album.name, track.album.releaseDate)

				exec("insert into album_tracks (album_id, track_id) values ($1, $2) on conflict (album_id, track_id) do nothing",
					track.album.id, track.id)
			}
		}
	}
}
