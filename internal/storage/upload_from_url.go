package storage

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxUploadFromURLSize = 10 << 20 // 10MB

var uploadFromURLClient = &http.Client{Timeout: 45 * time.Second}

// UploadFromURL 下载远程图片并写入报价中心存储。
func UploadFromURL(store Storage, rawURL, subdir string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("empty url")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported url scheme")
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "QuoteCore/1.0")

	resp, err := uploadFromURLClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	body := io.LimitReader(resp.Body, maxUploadFromURLSize+1)
	tmp, err := os.CreateTemp("", "quotecore-url-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	n, err := io.Copy(tmp, body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if n > maxUploadFromURLSize {
		return "", fmt.Errorf("file too large (max 10MB)")
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}

	filename := filenameFromURL(parsed, resp.Header.Get("Content-Type"))
	if !isAllowedImageName(filename, resp.Header.Get("Content-Type")) {
		return "", fmt.Errorf("unsupported image type (jpg/jpeg/png/webp/gif)")
	}
	subdir = strings.Trim(subdir, "/")
	if subdir == "" {
		subdir = "items"
	}
	return store.UploadPath(tmpPath, filename, subdir)
}

func filenameFromURL(parsed *url.URL, contentType string) string {
	name := filepath.Base(parsed.Path)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == "/" {
		name = "image"
	}
	// strip query-like junk from basename
	if i := strings.IndexAny(name, "?#"); i >= 0 {
		name = name[:i]
	}
	ext := filepath.Ext(name)
	if ext == "" {
		switch strings.ToLower(strings.Split(contentType, ";")[0]) {
		case "image/jpeg", "image/jpg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		case "image/gif":
			ext = ".gif"
		default:
			ext = ".jpg"
		}
		name += ext
	}
	return safeFilename(name)
}

func isAllowedImageName(filename, contentType string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return true
	}
	ct := strings.ToLower(strings.Split(contentType, ";")[0])
	switch ct {
	case "image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif":
		return true
	}
	return false
}
