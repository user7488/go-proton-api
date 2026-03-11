package proton

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"github.com/go-resty/resty/v2"
)

func (c *Client) GetBlock(ctx context.Context, bareURL, token string) (io.ReadCloser, error) {
	res, err := c.doRes(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetHeader("pm-storage-token", token).SetDoNotParseResponse(true).Get(bareURL)
	})
	if err != nil {
		return nil, err
	}

	return res.RawBody(), nil
}

func (c *Client) RequestBlockUpload(ctx context.Context, req BlockUploadReq) ([]BlockUploadLink, error) {
	var res struct {
		UploadLinks []BlockUploadLink
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).SetBody(req).Post("/drive/blocks")
	}); err != nil {
		return nil, err
	}

	return res.UploadLinks, nil
}

func (c *Client) UploadBlock(ctx context.Context, bareURL, token string, block io.Reader) error {
	// Build multipart body — the part has no Content-Type header,
	// matching the official Proton Drive SDK behavior.
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="Block"; filename="blob"`)
	pw, err := w.CreatePart(partHeader)
	if err != nil {
		return err
	}
	if _, err := io.Copy(pw, block); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	// Build a plain HTTP request — do NOT send Proton API auth headers
	// (x-pm-uid, Authorization) to the storage server.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, bareURL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("pm-storage-token", token)
	req.Header.Set("Content-Type", w.FormDataContentType())

	transport := c.m.rc.GetClient().Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil {
			apiErr.Status = resp.StatusCode
			return fmt.Errorf("%v %s %s: %w", resp.StatusCode, http.MethodPost, bareURL, &apiErr)
		}
		return fmt.Errorf("%v %s %s: %s", resp.StatusCode, http.MethodPost, bareURL, resp.Status)
	}

	return nil
}
