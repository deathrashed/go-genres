package main

import (
	"os"
	"path/filepath"
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
