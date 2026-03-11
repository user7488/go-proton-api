package proton

import (
	"context"

	"github.com/go-resty/resty/v2"
)

type VerificationData struct {
	VerificationCode string
	ContentKeyPacket string
}

func (c *Client) GetVerificationData(ctx context.Context, volumeID, linkID, revisionID string) (VerificationData, error) {
	var res struct {
		VerificationCode string
		ContentKeyPacket string
	}

	if err := c.do(ctx, func(r *resty.Request) (*resty.Response, error) {
		return r.SetResult(&res).Get("/drive/v2/volumes/" + volumeID + "/links/" + linkID + "/revisions/" + revisionID + "/verification")
	}); err != nil {
		return VerificationData{}, err
	}

	return VerificationData{
		VerificationCode: res.VerificationCode,
		ContentKeyPacket: res.ContentKeyPacket,
	}, nil
}
