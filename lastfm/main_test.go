package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bogem/id3v2/v2"
)

func TestWriteGenrePreservesExistingMetadata(t *testing.T) {
	file := filepath.Join(t.TempDir(), "track.mp3")

	tag := id3v2.NewEmptyTag()
	tag.SetArtist("Existing Artist")
	tag.SetAlbum("Existing Album")
	tag.SetTitle("Existing Title")
	tag.SetYear("1999")
	tag.SetGenre("Old Genre")
	tag.AddCommentFrame(id3v2.CommentFrame{
		Encoding:    id3v2.EncodingUTF8,
		Language:    "eng",
		Description: "source",
		Text:        "keep me",
	})

	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tag.WriteTo(f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("fake audio payload")); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if !writeGenre(file, "New Genre") {
		t.Fatal("writeGenre returned false")
	}

	updated, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		t.Fatal(err)
	}
	defer updated.Close()

	if updated.Artist() != "Existing Artist" {
		t.Fatalf("artist not preserved: %q", updated.Artist())
	}
	if updated.Album() != "Existing Album" {
		t.Fatalf("album not preserved: %q", updated.Album())
	}
	if updated.Title() != "Existing Title" {
		t.Fatalf("title not preserved: %q", updated.Title())
	}
	if updated.Year() != "1999" {
		t.Fatalf("year not preserved: %q", updated.Year())
	}
	if updated.Genre() != "New Genre" {
		t.Fatalf("genre not updated: %q", updated.Genre())
	}
	if len(updated.GetFrames("COMM")) == 0 {
		t.Fatal("comment frame was not preserved")
	}
}

func TestReadArtistFallsBackToAlbumArtist(t *testing.T) {
	file := filepath.Join(t.TempDir(), "track.mp3")

	tag := id3v2.NewEmptyTag()
	tag.AddTextFrame("TPE2", id3v2.EncodingUTF8, "Album Artist")

	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tag.WriteTo(f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("fake audio payload")); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if artist := readArtist(file); artist != "Album Artist" {
		t.Fatalf("expected album artist fallback, got %q", artist)
	}
}

func TestReadArtistPrefersTrackArtist(t *testing.T) {
	file := filepath.Join(t.TempDir(), "track.mp3")

	tag := id3v2.NewEmptyTag()
	tag.SetArtist("Track Artist")
	tag.AddTextFrame("TPE2", id3v2.EncodingUTF8, "Album Artist")
	tag.AddTextFrame("TOPE", id3v2.EncodingUTF8, "Original Artist")

	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tag.WriteTo(f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("fake audio payload")); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if artist := readArtist(file); artist != "Track Artist" {
		t.Fatalf("expected track artist, got %q", artist)
	}
}

func TestReadArtistFallsBackToExiftoolMetadata(t *testing.T) {
	tempDir := t.TempDir()
	fakeExiftool := filepath.Join(tempDir, "exiftool")
	if err := os.WriteFile(fakeExiftool, []byte("#!/bin/sh\nprintf '%s\n' '[{\"Artist\":\"\",\"AlbumArtist\":\"Exif Album Artist\",\"Band\":\"Exif Band\",\"OriginalArtist\":\"Exif Original\",\"Album\":\"Exif Album\"}]'\n"), 0755); err != nil {
		t.Fatal(err)
	}

	previousExiftoolPath := exiftoolPath
	exiftoolPath = fakeExiftool
	t.Cleanup(func() { exiftoolPath = previousExiftoolPath })

	file := filepath.Join(tempDir, "track.mp3")
	if err := os.WriteFile(file, []byte("not an id3 tag"), 0644); err != nil {
		t.Fatal(err)
	}

	if artist := readArtist(file); artist != "Exif Album Artist" {
		t.Fatalf("expected exiftool album artist fallback, got %q", artist)
	}
	if album := readAlbum(file); album != "Exif Album" {
		t.Fatalf("expected exiftool album fallback, got %q", album)
	}
}

func TestWriteGenreWithExiftool(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "exiftool.log")
	fakeExiftool := filepath.Join(tempDir, "exiftool")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + logFile + "\"\n"
	if err := os.WriteFile(fakeExiftool, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	previousExiftoolPath := exiftoolPath
	exiftoolPath = fakeExiftool
	t.Cleanup(func() { exiftoolPath = previousExiftoolPath })

	file := filepath.Join(tempDir, "track.mp3")
	if err := os.WriteFile(file, []byte("not an id3 tag"), 0644); err != nil {
		t.Fatal(err)
	}

	if !writeGenreWithExiftool(file, "New Genre") {
		t.Fatal("writeGenreWithExiftool returned false")
	}

	log, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	args := string(log)
	if !strings.Contains(args, "-overwrite_original") || !strings.Contains(args, "-Genre=New Genre") || !strings.Contains(args, file) {
		t.Fatalf("unexpected exiftool args: %q", args)
	}
}

func TestParseLastFMResponseFiltersAndLimitsTags(t *testing.T) {
	body := []byte(`{"toptags":{"tag":[{"name":"thrash metal","count":100},{"name":"australian","count":13},{"name":"black metal","count":3},{"name":"seen live","count":1}]}}`)

	tags, err := parseLastFMResponse(body, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"thrash metal", "black metal"}
	if strings.Join(tags, ";") != strings.Join(want, ";") {
		t.Fatalf("tags = %v, want %v", tags, want)
	}
}
