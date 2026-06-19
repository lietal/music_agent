package tme

import (
	"context"
	"fmt"
)

// QRCode represents a QQ Music login QR code.
type QRCode struct {
	QrcodeURL string `json:"qrcode_url"`
	Key       string `json:"key"`
}

// QRStatus represents the current status of a QQ Music QR login.
type QRStatus struct {
	Status    string `json:"status"`
	MusicID   string `json:"music_id,omitempty"`
	MusicKey  string `json:"music_key,omitempty"`
	UserName  string `json:"user_name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// GetLoginQRCode requests a new QQ Music login QR code.
func (c *Client) GetLoginQRCode(ctx context.Context) (*QRCode, error) {
	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.login.Qrcode",
			Method: "GetQrcode",
			Param:  map[string]any{},
		},
	})
	if err != nil {
		return nil, err
	}
	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("get qrcode failed: code=%d", sub.Code)
	}
	return &QRCode{
		QrcodeURL: getString(sub.Data, "qrcode_url"),
		Key:       getString(sub.Data, "qrcode_key"),
	}, nil
}

// CheckQRCodeStatus polls the QR code login status.
// Returns status: "pending", "scanned", "confirmed", or "expired".
// When confirmed, MusicID, MusicKey, and UserName are populated.
func (c *Client) CheckQRCodeStatus(ctx context.Context, key string) (*QRStatus, error) {
	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.login.Qrcode",
			Method: "CheckQrcode",
			Param: map[string]any{
				"qrcode_key": key,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("check qrcode failed: code=%d", sub.Code)
	}

	statusInt := getInt(sub.Data, "status")
	statusMap := map[int]string{1: "pending", 2: "scanned", 3: "confirmed", 4: "expired"}
	status := statusMap[statusInt]
	if status == "" {
		status = "pending"
	}

	qr := &QRStatus{Status: status}
	if status == "confirmed" {
		qr.MusicID = fmt.Sprintf("%d", getInt(sub.Data, "musicid"))
		qr.MusicKey = getString(sub.Data, "musickey")
		qr.UserName = getString(sub.Data, "nickname")
		qr.AvatarURL = getString(sub.Data, "headurl")
	}
	return qr, nil
}
