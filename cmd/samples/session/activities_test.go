package main

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	artificialActivityDelay = 0
	artificialActivityRandomErrors = false

	os.Exit(m.Run())
}

// TestFindRandomImageActivity is a dummy test that demos how you can write a
// simple test for an activity. In the real world the function that we're
// testing shouldn't have dependencies with third-party services.
func TestFindRandomImageActivity(t *testing.T) {
	addr, err := findRandomImageActivity(context.Background())
	if err != nil {
		t.Fatalf("findRandomImageActivity returned an error: %v", err)
	}

	if !strings.HasPrefix(addr, "https://images.unsplash.com/") {
		t.Fatalf("findRandomImageActivity returned an unexpected location: %v", addr)
	}
}

func TestDownloadImageActivity(t *testing.T) {
	const imageAddress = "https://images.unsplash.com/photo-1568821592542-3088f9e90bcf?crop=entropy&cs=tinysrgb&fit=crop&fm=jpg&h=600&ixlib=rb-1.2.1&q=80&w=800"
	path, err := downloadImageActivity(context.Background(), imageAddress)
	if err != nil {
		t.Fatalf("downloadImageActivity returned an error: %v", err)
	}

	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Fatalf("downloadImageActivity did not create the file properly: %v (%s)", err, path)
	}
	if stat.Size() == 0 {
		t.Fatalf("downloadImageActivity returned a file but it's empty: %s", path)
	}
}

func TestCalcChecksumActivity(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.Write([]byte("..."))

	// The checksum we expect.
	const want = "6eae3a5b062c6d0d79f070c26e6d62486b40cb46"

	sum, err := calcChecksumActivity(context.Background(), f.Name())
	if err != nil {
		t.Fatalf("calcChecksumActivity returned an error: %v", err)
	}
	if sum != want {
		t.Fatalf("calcChecksumActivity returned an unexpected checksum; want %s, got %s", want, sum)
	}
}
