package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jellydn/knulli-app-store/internal/manifest"
)

const maximumReleaseBytes int64 = 512 << 20

func download(ctx context.Context, client *http.Client, release manifest.Release, destination string) error {
	if release.Size > maximumReleaseBytes {
		return fmt.Errorf("release size %d exceeds the %d-byte limit", release.Size, maximumReleaseBytes)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, release.URL, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", response.Status)
	}
	if response.Request == nil || response.Request.URL.Scheme != "https" {
		return fmt.Errorf("download ended at a non-HTTPS URL")
	}
	if response.ContentLength >= 0 && response.ContentLength != release.Size {
		return fmt.Errorf("download size is %d, expected %d", response.ContentLength, release.Size)
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(output, hash), io.LimitReader(response.Body, release.Size+1))
	syncErr := output.Sync()
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written != release.Size {
		return fmt.Errorf("download size is %d, expected %d", written, release.Size)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != release.SHA256 {
		return fmt.Errorf("SHA-256 mismatch: got %s", actual)
	}
	return nil
}

func defaultHTTPClient() *http.Client {
	client := &http.Client{Timeout: 10 * time.Minute}
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		if request.URL.Scheme != "https" {
			return fmt.Errorf("redirect to non-HTTPS URL")
		}
		return nil
	}
	return client
}

func validateDownloadURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("release URL must use HTTPS")
	}
	if strings.Contains(strings.ToLower(parsed.Path), "/latest/") {
		return fmt.Errorf("release URL must not use a latest-release alias")
	}
	return nil
}
