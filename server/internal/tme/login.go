package tme

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (c *Client) ptloginURL() string {
	if c.loginBaseURL != "" {
		return c.loginBaseURL
	}
	return "https://ssl.ptlogin2.qq.com"
}

func (c *Client) graphURL() string {
	if c.loginBaseURL != "" {
		return c.loginBaseURL
	}
	return "https://ssl.ptlogin2.graph.qq.com"
}

const (
	qqAppID       = "716027609"
	qqMusicAppID  = "100497308"
	qqDaid        = "383"
	qqUserAgent   = "Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"
	defaultQIMEI36 = "6c9d3cd110abca9b16311cee10001e717614"
)

var ptuiCBRe = regexp.MustCompile(`ptuiCB\((.*?)\)`)
var qrsigRe = regexp.MustCompile(`qrsig=([^;]+)`)

// QRCode represents a QQ Music login QR code session.
type QRCode struct {
	QrcodeDataURL string `json:"qrcode_url"` // base64 data URL of the PNG QR code
	Key           string `json:"key"`         // qrsig used for polling
}

// QRStatus represents the current status of a QQ Music QR login.
type QRStatus struct {
	Status    string `json:"status"` // "pending" | "scanned" | "confirmed" | "expired"
	MusicID   string `json:"music_id,omitempty"`
	MusicKey  string `json:"music_key,omitempty"`
	OpenID    string `json:"openid,omitempty"`
	UnionID   string `json:"unionid,omitempty"`
	UserName  string `json:"user_name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	sigx      string // internal: ptsigx from login redirect
}

// loginSession holds state for an in-progress QQ Music QR login.
type loginSession struct {
	jar   *cookiejar.Jar
	qrsig string
	uin   string
	sigx  string
}

// GetLoginQRCode starts a QQ Music QR code login session.
// Returns a base64-encoded PNG data URL and the qrsig key for polling.
func (c *Client) GetLoginQRCode(ctx context.Context) (*QRCode, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 30 * time.Second}

	reqURL := fmt.Sprintf(
		"%s/ptqrshow?appid=%s&e=2&l=M&s=3&d=72&v=4&t=%s&daid=%s&pt_3rd_aid=%s",
		c.ptloginURL(), qqAppID, "0.1", qqDaid, qqMusicAppID,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create qr request: %w", err)
	}
	req.Header.Set("Referer", "https://xui.ptlogin2.qq.com/")
	req.Header.Set("User-Agent", qqUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch qr code: %w", err)
	}
	defer resp.Body.Close()

	// Extract qrsig from Set-Cookie
	qrsig := ""
	for _, cookie := range resp.Header["Set-Cookie"] {
		m := qrsigRe.FindStringSubmatch(cookie)
		if len(m) > 1 {
			qrsig = m[1]
			break
		}
	}
	if qrsig == "" {
		return nil, fmt.Errorf("no qrsig cookie in response")
	}

	// Read QR image
	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read qr image: %w", err)
	}

	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imgBytes)

	return &QRCode{
		QrcodeDataURL: dataURL,
		Key:           qrsig,
	}, nil
}

// CheckQRCodeStatus polls the QQ Music QR login status using the ptlogin2 flow.
func (c *Client) CheckQRCodeStatus(ctx context.Context, qrsig string) (*QRStatus, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse // don't follow redirects
	}}

	// Step 1: Poll ptqrlogin
	status, err := c.pollQRLogin(ctx, client, qrsig)
	if err != nil {
		return nil, err
	}
	if status.Status != "confirmed" {
		return status, nil
	}

	// Step 2: Check sig → get p_skey
	session := &loginSession{jar: jar, qrsig: qrsig, uin: status.MusicID, sigx: status.sigx}
	pSkey, err := c.checkSig(ctx, client, session)
	if err != nil {
		return nil, fmt.Errorf("check sig: %w", err)
	}

	// Step 3: Authorize → get code
	code, err := c.authorize(ctx, client, session, pSkey)
	if err != nil {
		return nil, fmt.Errorf("authorize: %w", err)
	}

	// Step 4: Exchange code for music credential
	return c.exchangeCredential(ctx, client, session, code)
}

// pollQRLogin polls the ptqrlogin endpoint and returns the status.
func (c *Client) pollQRLogin(ctx context.Context, client *http.Client, qrsig string) (*QRStatus, error) {
	token := hash33(qrsig)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)

	reqURL := fmt.Sprintf(
		"%s/ptqrlogin?u1=%s&ptqrtoken=%d&ptredirect=0&h=1&t=1&g=1&from_ui=1&ptlang=2052&action=0-0-%s&js_ver=20102616&js_type=1&pt_uistyle=40&aid=%s&daid=%s&pt_3rd_aid=%s&has_onekey=1",
		c.ptloginURL(), url.QueryEscape("https://graph.qq.com/oauth2.0/login_jump"), token, ts, qqAppID, qqDaid, qqMusicAppID,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create poll request: %w", err)
	}
	req.Header.Set("Referer", "https://xui.ptlogin2.qq.com/")
	req.Header.Set("Cookie", "qrsig="+url.QueryEscape(qrsig))
	req.Header.Set("User-Agent", qqUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poll login: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	text := string(body)

	m := ptuiCBRe.FindStringSubmatch(text)
	if len(m) < 2 {
		return &QRStatus{Status: "pending"}, nil
	}

	// Parse callback args: ptuiCB('code','0','url','','','','')
	parts := splitCSV(m[1])
	if len(parts) < 3 {
		return &QRStatus{Status: "pending"}, nil
	}

	code, _ := strconv.Atoi(strings.Trim(parts[0], "' "))
	switch code {
	case 0:
		// Login success, extract uin and sigx from redirect URL
		loginURL := strings.Trim(parts[2], "' ")
		uin := extractParam(loginURL, "uin")
		sigx := extractParam(loginURL, "ptsigx")
		return &QRStatus{
			Status:  "confirmed",
			MusicID: normalizeUin(uin),
			sigx:    sigx,
		}, nil
	case 65:
		return &QRStatus{Status: "expired"}, nil
	case 66:
		return &QRStatus{Status: "pending"}, nil
	case 67:
		return &QRStatus{Status: "scanned"}, nil
	default:
		return &QRStatus{Status: "pending"}, nil
	}
}

// checkSig validates the signature and returns p_skey.
func (c *Client) checkSig(ctx context.Context, client *http.Client, session *loginSession) (string, error) {
	v := url.Values{
		"uin":            {session.uin},
		"pttype":         {"1"},
		"service":        {"ptqrlogin"},
		"nodirect":       {"0"},
		"ptsigx":         {session.sigx},
		"s_url":          {"https://graph.qq.com/oauth2.0/login_jump"},
		"ptlang":         {"2052"},
		"ptredirect":     {"100"},
		"aid":            {qqAppID},
		"daid":           {qqDaid},
		"j_later":        {"0"},
		"low_login_hour": {"0"},
		"regmaster":      {"0"},
		"pt_login_type":  {"3"},
		"pt_aid":         {"0"},
		"pt_aaid":        {"16"},
		"pt_light":       {"0"},
		"pt_3rd_aid":     {qqMusicAppID},
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		c.graphURL()+"/check_sig?"+v.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("create check_sig request: %w", err)
	}
	req.Header.Set("Referer", "https://xui.ptlogin2.qq.com/")
	req.Header.Set("User-Agent", qqUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("check_sig: %w", err)
	}
	defer resp.Body.Close()

	for _, cookie := range resp.Header["Set-Cookie"] {
		if strings.HasPrefix(cookie, "p_skey=") {
			parts := strings.SplitN(cookie, ";", 2)
			return strings.TrimPrefix(parts[0], "p_skey="), nil
		}
	}

	// Try from cookie jar
	parsedURL, _ := url.Parse("https://ssl.ptlogin2.graph.qq.com")
	for _, cookie := range client.Jar.Cookies(parsedURL) {
		if cookie.Name == "p_skey" && cookie.Value != "" {
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("p_skey not found in check_sig response")
}

// authorize performs the OAuth authorization step and returns the authorization code.
func (c *Client) authorize(ctx context.Context, client *http.Client, session *loginSession, pSkey string) (string, error) {
	gTK := hash33WithSeed(pSkey, 5381)
	form := url.Values{
		"response_type": {"code"},
		"client_id":     {qqMusicAppID},
		"redirect_uri":  {"https://y.qq.com/portal/wx_redirect.html?login_type=1&surl=https%3A%252F%252Fy.qq.com%252F"},
		"scope":         {"get_user_info,get_app_friends"},
		"state":         {"state"},
		"switch":        {""},
		"from_ptlogin":  {"1"},
		"src":           {"1"},
		"update_auth":   {"1"},
		"openapi":       {"1010_1030"},
		"g_tk":          {strconv.Itoa(gTK)},
		"auth_time":     {strconv.FormatInt(time.Now().UnixMilli(), 10)},
		"ui":            {randomUI()},
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://graph.qq.com/oauth2.0/authorize",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create authorize request: %w", err)
	}
	req.Header.Set("Host", "graph.qq.com")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://y.qq.com/")
	req.Header.Set("User-Agent", qqUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("authorize: %w", err)
	}
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	code := extractParam(location, "code")
	if code == "" {
		return "", fmt.Errorf("no authorization code in redirect (status=%d)", resp.StatusCode)
	}
	return code, nil
}

// exchangeCredential exchanges the authorization code for music credentials.
func (c *Client) exchangeCredential(ctx context.Context, client *http.Client, session *loginSession, code string) (*QRStatus, error) {
	payload := map[string]any{
		"comm": map[string]any{
			"cv":           "13020508",
			"v":            "13020508",
			"QIMEI36":      defaultQIMEI36,
			"ct":           "11",
			"tmeAppID":     "qqmusic",
			"format":       "json",
			"inCharset":    "utf-8",
			"outCharset":   "utf-8",
			"uid":          session.uin,
			"tmeLoginType": "2",
		},
		"music.login.LoginServer.Login": map[string]any{
			"module": "music.login.LoginServer",
			"method": "Login",
			"param": map[string]any{
				"code": code,
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", defaultBaseURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create credential request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://y.qq.com/")
	req.Header.Set("User-Agent", qqUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange credential: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var raw map[string]any
	json.Unmarshal(respBody, &raw)

	loginResp, _ := raw["music.login.LoginServer.Login"].(map[string]any)
	if loginResp == nil {
		return nil, fmt.Errorf("unexpected credential response")
	}
	if code := getFloat(loginResp, "code"); code != 0 {
		return nil, fmt.Errorf("credential exchange failed: code=%v", code)
	}

	data, _ := loginResp["data"].(map[string]any)
	if data == nil {
		data = loginResp
	}

	return &QRStatus{
		Status:    "confirmed",
		MusicID:   fmt.Sprintf("%v", data["musicid"]),
		MusicKey:  fmt.Sprintf("%v", data["musickey"]),
		OpenID:    fmt.Sprintf("%v", data["openid"]),
		UnionID:   fmt.Sprintf("%v", data["unionid"]),
		UserName:  fmt.Sprintf("%v", data["nickname"]),
		AvatarURL: fmt.Sprintf("%v", data["headurl"]),
	}, nil
}

// hash33 computes the QQ hash33 algorithm (seed=0).
func hash33(s string) int {
	var result int
	for _, ch := range s {
		result += (result << 5) + int(ch)
	}
	return result & 0x7FFFFFFF
}

// hash33WithSeed computes the QQ hash33 algorithm with a given seed.
func hash33WithSeed(s string, seed int) int {
	result := seed
	for _, ch := range s {
		result = (result << 5) + result + int(ch)
	}
	return result & 0x7FFFFFFF
}

// splitCSV splits a QQ callback CSV string like "'0','0','url','','','',''"
func splitCSV(s string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false

	for _, ch := range s {
		switch ch {
		case '\'':
			inQuote = !inQuote
		case ',':
			if inQuote {
				current.WriteRune(ch)
			} else {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	parts = append(parts, current.String())
	return parts
}

// extractParam extracts a query parameter value from a URL.
func extractParam(rawURL, key string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get(key)
}

// normalizeUin removes leading 'o' or 'O' from a QQ uin.
func normalizeUin(uin string) string {
	if len(uin) > 0 && (uin[0] == 'o' || uin[0] == 'O') {
		return uin[1:]
	}
	return uin
}

// randomUI generates a random UI string for the authorize request.
func randomUI() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		time.Now().UnixNano()&0xFFFFFFFF,
		(time.Now().UnixNano()>>32)&0xFFFF,
		(time.Now().UnixNano()>>16)&0xFFFF,
		time.Now().UnixNano()&0xFFFF,
		time.Now().UnixNano()&0xFFFFFFFFFFFF,
	)
}
