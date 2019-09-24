package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"go.uber.org/cadence/activity"
)

const (
	findRandomImageActivityName = "findRandomImageActivity"
	downloadImageActivityName   = "downloadImageActivity"
	calcChecksumActivityName    = "calcChecksumActivity"
)

func init() {
	activity.RegisterWithOptions(
		findRandomImageActivity,
		activity.RegisterOptions{Name: findRandomImageActivityName},
	)
	activity.RegisterWithOptions(
		downloadImageActivity,
		activity.RegisterOptions{Name: downloadImageActivityName},
	)
	activity.RegisterWithOptions(
		calcChecksumActivity,
		activity.RegisterOptions{Name: calcChecksumActivityName},
	)
}

// This is our HTTP client with sane timeouts.
//
// It has been set up so it does not follow redirects. We do this so
// findRandomImageActivity can hit a remote HTTP server and capture the
// Location header of a 302 response.
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

var errRandom = errors.New("random error")

func findRandomImageActivity(ctx context.Context) (string, error) {
	if shouldFail() {
		return "", errRandom
	}

	const source = "https://source.unsplash.com/random/800x600"
	req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}

	return resp.Header.Get("Location"), nil
}

func downloadImageActivity(ctx context.Context, imageAddress string) (string, error) {
	if shouldFail() {
		return "", errRandom
	}

	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	defer f.Close()

	// TODO: fetch the actual blob in imageAddress.
	f.Write([]byte("foobar"))

	return f.Name(), nil
}

func calcChecksumActivity(ctx context.Context, path string) (string, error) {
	if shouldFail() {
		return "", errRandom
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha1.New()
	if _, err := io.Copy(hasher, f); err != nil {
		log.Fatal(err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func shouldFail() bool {
	if artificialActivityDelay > 0 {
		time.Sleep(time.Second * time.Duration(2))
	}
	if !artificialActivityRandomErrors {
		return false
	}
	src := rand.NewSource(time.Now().UnixNano())
	return src.Int63()&0x01 == 1
}
