package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const Version = "1.0.1"
const Repository = "Starfallen9008/elastic-ui"
const InstallerName = "elastic-ui-windows-amd64-setup.exe"

type Release struct {
	Version     string `json:"version"`
	Available   bool   `json:"available"`
	URL         string `json:"url"`
	DownloadURL string `json:"downloadUrl"`
	ChecksumURL string `json:"checksumUrl"`
	Notes       string `json:"notes"`
	Size        int64  `json:"size"`
}

func newer(candidate, current string) bool {
	parse := func(version string) ([3]int, bool) {
		var numbers [3]int
		parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
		if len(parts) != 3 {
			return numbers, false
		}
		for i, part := range parts {
			number, err := strconv.Atoi(part)
			if err != nil || number < 0 {
				return numbers, false
			}
			numbers[i] = number
		}
		return numbers, true
	}
	a, okA := parse(candidate)
	b, okB := parse(current)
	if !okA || !okB {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

func trustedAsset(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == "github.com" && u.User == nil &&
		strings.HasPrefix(u.Path, "/"+Repository+"/releases/download/")
}

func request(ctx context.Context, address string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ElasticUI/"+Version)
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" || len(via) > 5 {
			return errors.New("更新下载重定向无效")
		}
		return nil
	}}
	return client.Do(req)
}

func Check(ctx context.Context) (*Release, error) {
	response, err := request(ctx, "https://api.github.com/repos/"+Repository+"/releases/latest")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, errors.New("GitHub 仓库或正式发布版本不存在")
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub 更新检查失败：HTTP %d", response.StatusCode)
	}
	var raw struct {
		Tag        string `json:"tag_name"`
		URL        string `json:"html_url"`
		Body       string `json:"body"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
			Size int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&raw); err != nil {
		return nil, err
	}
	release := &Release{Version: strings.TrimPrefix(raw.Tag, "v"), URL: raw.URL, Notes: raw.Body}
	for _, asset := range raw.Assets {
		if asset.Name == InstallerName {
			release.DownloadURL = asset.URL
			release.Size = asset.Size
		}
		if asset.Name == "SHA256SUMS" {
			release.ChecksumURL = asset.URL
		}
	}
	release.Available = !raw.Draft && !raw.Prerelease && newer(raw.Tag, Version)
	if release.Available && (!trustedAsset(release.DownloadURL) || !trustedAsset(release.ChecksumURL)) {
		return nil, errors.New("发布版本缺少有效安装包或校验文件")
	}
	return release, nil
}

func Download(ctx context.Context, release Release, directory string) (string, string, error) {
	if !release.Available || !trustedAsset(release.DownloadURL) || !trustedAsset(release.ChecksumURL) {
		return "", "", errors.New("更新来源无效")
	}
	response, err := request(ctx, release.ChecksumURL)
	if err != nil {
		return "", "", err
	}
	data, readErr := io.ReadAll(io.LimitReader(response.Body, 65536))
	response.Body.Close()
	if readErr != nil {
		return "", "", readErr
	}
	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("校验文件下载失败：HTTP %d", response.StatusCode)
	}
	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == InstallerName {
			expected = strings.ToLower(fields[0])
		}
	}
	decoded, err := hex.DecodeString(expected)
	if err != nil || len(decoded) != sha256.Size {
		return "", "", errors.New("安装包 SHA256 缺失或无效")
	}
	response, err = request(ctx, release.DownloadURL)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("安装包下载失败：HTTP %d", response.StatusCode)
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", "", err
	}
	file, err := os.CreateTemp(directory, "update-*.exe")
	if err != nil {
		return "", "", err
	}
	path := file.Name()
	success := false
	defer func() {
		file.Close()
		if !success {
			os.Remove(path)
		}
	}()
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, 201<<20))
	if err != nil {
		return "", "", err
	}
	if size > 200<<20 || (release.Size > 0 && size != release.Size) {
		return "", "", errors.New("安装包大小校验失败")
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected {
		return "", "", errors.New("安装包 SHA256 校验失败")
	}
	if err := file.Close(); err != nil {
		return "", "", err
	}
	if filepath.Ext(path) != ".exe" {
		return "", "", errors.New("安装包格式错误")
	}
	success = true
	return path, expected, nil
}
