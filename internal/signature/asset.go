package signature

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"image"
	_ "image/gif" // Register supported email image formats.
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/woodleighschool/sigweft/internal/config"
)

const (
	assetFetchTimeout = 10 * time.Second
	maxAssetBytes     = 2 << 20
	maxAssetRedirects = 3
)

func loadAsset(ctx context.Context, root *os.Root, definition config.Asset) (Asset, error) {
	source, err := url.Parse(definition.Src)
	if err != nil {
		return Asset{}, fmt.Errorf("parse source: %w", err)
	}
	var data []byte
	switch {
	case source.Scheme == "" && source.Host == "":
		data, err = root.ReadFile(definition.Src)
	case (source.Scheme == "https" || source.Scheme == "http") && source.Host != "":
		data, err = fetchAsset(ctx, source.String())
	default:
		return Asset{}, errors.New("src must be a relative path or an absolute HTTP or HTTPS URL")
	}
	if err != nil {
		return Asset{}, fmt.Errorf("read image: %w", err)
	}
	if len(data) > maxAssetBytes {
		return Asset{}, errors.New("image exceeds 2 MiB")
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Asset{}, fmt.Errorf("decode image: %w", err)
	}
	if definition.Href != "" {
		link, err := url.Parse(definition.Href)
		if err != nil || link.Host == "" || (link.Scheme != "https" && link.Scheme != "http") {
			return Asset{}, errors.New("href must be an absolute HTTP or HTTPS URL")
		}
	}
	return Asset{
		// Only bytes from a decoded raster image may bypass html/template's URL filter.
		Src: template.URL("data:image/" + format + ";base64," + base64.StdEncoding.EncodeToString(data)), //nolint:gosec // Validated image bytes, not user input.
		Alt: definition.Alt, Width: definition.Width, Height: definition.Height, Href: definition.Href,
	}, nil
}

func fetchAsset(ctx context.Context, source string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, fmt.Errorf("create image request: %w", err)
	}
	client := &http.Client{
		Timeout: assetFetchTimeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) > maxAssetRedirects {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
	response, err := client.Do(request)
	if err != nil {
		if urlError, ok := errors.AsType[*url.Error](err); ok {
			err = urlError.Err
		}
		return nil, fmt.Errorf("fetch image: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("fetch image: unexpected HTTP status %d", response.StatusCode)
	}
	if response.ContentLength > maxAssetBytes {
		return nil, errors.New("image exceeds 2 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxAssetBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return data, nil
}
