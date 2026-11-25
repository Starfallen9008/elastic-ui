package update

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
)

type fixtureTransport func(*http.Request) (*http.Response, error)

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestVersionsAndSources(t *testing.T) {
	patch := "v" + bumpVersion(Version, 2)
	for _, version := range []string{patch, bumpVersion(Version, 1), bumpVersion(Version, 0)} {
		if !newer(version, Version) {
			t.Fatalf("rejected newer version %s", version)
		}
	}
	for _, version := range []string{"v" + Version, "0.0.1", "v" + Version + "-beta", "latest", "1.0", "1.-1.0"} {
		if newer(version, Version) {
			t.Fatalf("accepted version %s", version)
		}
	}
	for _, address := range []string{
		"http://github.com/" + Repository + "/releases/download/" + patch + "/setup.exe",
		"https://github.com/other/repo/releases/download/" + patch + "/setup.exe",
		"https://github.com.evil.invalid/" + Repository + "/releases/download/" + patch + "/setup.exe",
	} {
		if trustedAsset(address) {
			t.Fatalf("accepted source %s", address)
		}
	}
}

func bumpVersion(version string, index int) string {
	parts := strings.Split(version, ".")
	number, _ := strconv.Atoi(parts[index])
	parts[index] = strconv.Itoa(number + 1)
	return strings.Join(parts, ".")
}

func TestCheckAndDownloadRejectsTampering(t *testing.T) {
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	data := []byte("inert installer fixture")
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	tag := "v" + bumpVersion(Version, 2)
	tampered := false
	http.DefaultTransport = fixtureTransport(func(r *http.Request) (*http.Response, error) {
		body := ""
		switch {
		case r.URL.Host == "api.github.com":
			body = `{"tag_name":"` + tag + `","assets":[{"name":"` + InstallerName + `","size":23,"browser_download_url":"https://github.com/` + Repository + `/releases/download/` + tag + `/` + InstallerName + `"},{"name":"SHA256SUMS","browser_download_url":"https://github.com/` + Repository + `/releases/download/` + tag + `/SHA256SUMS"}]}`
		case strings.HasSuffix(r.URL.Path, "SHA256SUMS"):
			body = digest + "  " + InstallerName
		default:
			body = string(data)
			if tampered {
				body = "tampered installer file!"
			}
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})
	release, err := Check(context.Background())
	if err != nil || !release.Available {
		t.Fatalf("check: %+v %v", release, err)
	}
	release.Size = int64(len(data))
	directory := t.TempDir()
	path, hash, err := Download(context.Background(), *release, directory)
	if err != nil || hash != digest {
		t.Fatalf("download: %s %v", hash, err)
	}
	if actual, _ := os.ReadFile(path); string(actual) != string(data) {
		t.Fatal("download changed")
	}
	tampered = true
	if _, _, err := Download(context.Background(), *release, directory); err == nil {
		t.Fatal("accepted tampered installer")
	}
	entries, _ := os.ReadDir(directory)
	if len(entries) != 1 {
		t.Fatal("failed download left partial executable")
	}
}
