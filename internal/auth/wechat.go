package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type WeChatProvider struct {
	appID       string
	appSecret   string
	redirectURI string
	httpClient  *http.Client
}

func NewWeChatProvider(appID, appSecret, redirectURI string) *WeChatProvider {
	return &WeChatProvider{
		appID:       appID,
		appSecret:   appSecret,
		redirectURI: redirectURI,
		httpClient:  &http.Client{},
	}
}

func (p *WeChatProvider) Name() string {
	return "wechat"
}

func (p *WeChatProvider) AuthURL(state string) string {
	params := url.Values{}
	params.Set("appid", p.appID)
	params.Set("redirect_uri", p.redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "snsapi_userinfo")
	params.Set("state", state)

	return "https://open.weixin.qq.com/connect/oauth2/authorize?" + params.Encode() + "#wechat_redirect"
}

type wechatAccessTokenResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type wechatUserInfoResp struct {
	OpenID     string `json:"openid"`
	Nickname   string `json:"nickname"`
	Sex        int    `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
	HeadImgURL string `json:"headimgurl"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func (p *WeChatProvider) Exchange(ctx context.Context, code string) (*UserInfo, error) {
	accessToken, openID, err := p.exchangeCode(ctx, code)
	if err != nil {
		return nil, err
	}

	userInfo, err := p.getUserInfo(ctx, accessToken, openID)
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

func (p *WeChatProvider) exchangeCode(ctx context.Context, code string) (string, string, error) {
	params := url.Values{}
	params.Set("appid", p.appID)
	params.Set("secret", p.appSecret)
	params.Set("code", code)
	params.Set("grant_type", "authorization_code")

	reqURL := "https://api.weixin.qq.com/sns/oauth2/access_token?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp wechatAccessTokenResp
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.ErrCode != 0 {
		return "", "", fmt.Errorf("wechat token error: %s (code: %d)", tokenResp.ErrMsg, tokenResp.ErrCode)
	}

	return tokenResp.AccessToken, tokenResp.OpenID, nil
}

func (p *WeChatProvider) getUserInfo(ctx context.Context, accessToken, openID string) (*UserInfo, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("openid", openID)
	params.Set("lang", "zh_CN")

	reqURL := "https://api.weixin.qq.com/sns/userinfo?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get userinfo: %w", err)
	}
	defer resp.Body.Close()

	var userResp wechatUserInfoResp
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}

	if userResp.ErrCode != 0 {
		return nil, fmt.Errorf("wechat userinfo error: %s (code: %d)", userResp.ErrMsg, userResp.ErrCode)
	}

	return &UserInfo{
		UserID:      userResp.OpenID,
		Provider:    "wechat",
		ProviderID:  userResp.OpenID,
		DisplayName: userResp.Nickname,
		AvatarURL:   userResp.HeadImgURL,
	}, nil
}
