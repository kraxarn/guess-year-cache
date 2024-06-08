create table tracks
(
    id          text primary key,
    name        text not null,
    preview_url text not null
);

create table playlists
(
    id   text primary key,
    name text not null
);

create table artists
(
    id   text primary key,
    name text not null
);

create table albums
(
    id           text primary key,
    name         text not null,
    release_date date not null
);

create table playlist_tracks
(
    playlist_id text not null references playlists (id),
    track_id    text not null references tracks (id)
);

create table track_artists
(
    track_id  text not null references tracks (id),
    artist_id text not null references artists (id)
);

create table album_tracks
(
    album_id text not null references albums (id),
    track_id text not null references tracks (id)
);

create unique index idx_playlist_track on playlist_tracks (playlist_id, track_id);
create unique index idx_track_artist on track_artists (track_id, artist_id);
create unique index idx_album_track on album_tracks (album_id, track_id);