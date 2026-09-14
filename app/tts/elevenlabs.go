package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultElevenLabsOrigin = "https://api.elevenlabs.io"

// Voice is one ElevenLabs voice the user may pick.
type Voice struct {
	ID       string `json:"voice_id"`
	Name     string `json:"name"`
	Category string `json:"category,omitempty"`
}

// Synthesizer is the ElevenLabs surface the service uses. Tests replace it.
type Synthesizer interface {
	Synthesize(ctx context.Context, req SynthesisRequest, apiKey string) ([]byte, error)
	ListVoices(ctx context.Context, apiKey string) ([]Voice, error)
}

// Client talks to ElevenLabs with an ephemeral HTTP client: no cookie jar,
// no URL cache, and credential-bearing redirects only to the same HTTPS origin.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (c *Client) base() string {
	if strings.TrimSpace(c.BaseURL) != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return defaultElevenLabsOrigin
}

func (c *Client) http() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{
		Timeout:       90 * time.Second,
		CheckRedirect: sameHTTPSOriginRedirect,
	}
}

func sameHTTPSOriginRedirect(req *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		return http.ErrUseLastResponse
	}
	from := via[len(via)-1].URL
	if !sameHTTPSOrigin(from, req.URL) {
		return http.ErrUseLastResponse
	}
	return nil
}

func sameHTTPSOrigin(from, to *url.URL) bool {
	if from == nil || to == nil {
		return false
	}
	if !strings.EqualFold(from.Scheme, "https") || !strings.EqualFold(to.Scheme, "https") {
		return false
	}
	if strings.ToLower(from.Hostname()) != strings.ToLower(to.Hostname()) {
		return false
	}
	return httpsPort(from) == httpsPort(to)
}

func httpsPort(u *url.URL) string {
	if u.Port() == "" {
		return "443"
	}
	return u.Port()
}

func (c *Client) Synthesize(ctx context.Context, req SynthesisRequest, apiKey string) ([]byte, error) {
	voiceID := strings.TrimSpace(req.VoiceID)
	if voiceID == "" {
		return nil, httpErrorf(http.StatusBadRequest, "Choose an ElevenLabs voice in Settings.")
	}
	if !validVoiceID(voiceID) {
		return nil, httpErrorf(http.StatusBadRequest, "Choose an ElevenLabs voice in Settings.")
	}
	if !req.SpeedValid() {
		return nil, httpErrorf(http.StatusBadRequest, "Speech speed must be between 0.70 and 1.20.")
	}

	u, err := url.Parse(c.base() + "/v1/text-to-speech/" + url.PathEscape(voiceID))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("output_format", OutputFormat)
	u.RawQuery = q.Encode()

	payload, err := json.Marshal(speechRequest{
		Text:    req.Text,
		ModelID: req.ModelID,
		VoiceSettings: voiceSettings{
			Stability:       Stability,
			SimilarityBoost: SimilarityBoost,
			Style:           Style,
			UseSpeakerBoost: UseSpeakerBoost,
			Speed:           req.Speed,
		},
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("xi-api-key", apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "audio/mpeg")
	httpReq.Header.Set("Cache-Control", "no-store")

	resp, err := c.http().Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, mapProviderError(resp.StatusCode, body)
	}
	if len(body) == 0 {
		return nil, httpErrorf(http.StatusBadGateway, "ElevenLabs returned no audio.")
	}
	mime := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	if mime != "audio/mpeg" && mime != "audio/mp3" && mime != "application/octet-stream" {
		return nil, httpErrorf(http.StatusBadGateway, "ElevenLabs returned data that was not audio.")
	}
	return body, nil
}

func (c *Client) ListVoices(ctx context.Context, apiKey string) ([]Voice, error) {
	voicesByID := map[string]Voice{}
	var next *string
	seen := map[string]struct{}{}
	for {
		u, err := url.Parse(c.base() + "/v2/voices")
		if err != nil {
			return nil, err
		}
		q := u.Query()
		q.Set("page_size", "100")
		q.Set("sort", "name")
		q.Set("sort_direction", "asc")
		q.Set("include_total_count", "false")
		if next != nil {
			q.Set("next_page_token", *next)
		}
		u.RawQuery = q.Encode()

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("xi-api-key", apiKey)
		httpReq.Header.Set("Cache-Control", "no-store")

		resp, err := c.http().Do(httpReq)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return nil, mapProviderError(resp.StatusCode, body)
		}
		var page voicePage
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, httpErrorf(http.StatusBadGateway, "ElevenLabs returned an invalid response.")
		}
		for _, v := range page.Voices {
			voicesByID[v.ID] = v
		}
		if !page.HasMore {
			break
		}
		token := strings.TrimSpace(page.NextPageToken)
		if token == "" {
			return nil, httpErrorf(http.StatusBadGateway, "ElevenLabs returned an invalid response.")
		}
		if _, dup := seen[token]; dup {
			return nil, httpErrorf(http.StatusBadGateway, "ElevenLabs returned an invalid response.")
		}
		seen[token] = struct{}{}
		next = &token
	}
	out := make([]Voice, 0, len(voicesByID))
	for _, v := range voicesByID {
		out = append(out, v)
	}
	return out, nil
}

func (r SynthesisRequest) SpeedValid() bool {
	if r.Speed != r.Speed { // NaN
		return false
	}
	return r.Speed >= MinSpeed && r.Speed <= MaxSpeed
}

func validVoiceID(id string) bool {
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

type speechRequest struct {
	Text          string        `json:"text"`
	ModelID       string        `json:"model_id"`
	VoiceSettings voiceSettings `json:"voice_settings"`
}

type voiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
	Style           float64 `json:"style"`
	UseSpeakerBoost bool    `json:"use_speaker_boost"`
	Speed           float64 `json:"speed"`
}

type voicePage struct {
	Voices        []Voice `json:"voices"`
	HasMore       bool    `json:"has_more"`
	NextPageToken string  `json:"next_page_token"`
}

func mapProviderError(status int, body []byte) error {
	code, typ := providerCodes(body)
	switch code {
	case "quota_exceeded", "insufficient_credits":
		return httpErrorf(http.StatusPaymentRequired, "The ElevenLabs credit limit was reached. Increase this API key's usage cap, add account credits, or wait for the next refresh.")
	case "invalid_api_key", "missing_api_key", "invalid_authorization_header", "unauthorized", "sign_in_required":
		return httpErrorf(http.StatusUnauthorized, "The ElevenLabs API key was rejected. Replace it in Settings.")
	case "insufficient_permissions":
		return httpErrorf(http.StatusForbidden, "The ElevenLabs API key lacks the required permission. Enable Text to Speech and Voices access.")
	case "concurrent_limit_exceeded", "too_many_concurrent_requests":
		return httpErrorf(http.StatusTooManyRequests, "Another ElevenLabs request is still running. Try again shortly.")
	case "system_busy":
		return httpErrorf(http.StatusServiceUnavailable, "ElevenLabs is busy. Try again shortly.")
	case "rate_limit_exceeded":
		return httpErrorf(http.StatusTooManyRequests, "The ElevenLabs rate or account limit was reached. Try again later.")
	}
	switch typ {
	case "payment_required":
		return httpErrorf(http.StatusPaymentRequired, "The ElevenLabs credit limit was reached. Increase this API key's usage cap, add account credits, or wait for the next refresh.")
	case "authentication_error":
		return httpErrorf(http.StatusUnauthorized, "The ElevenLabs API key was rejected. Replace it in Settings.")
	case "authorization_error":
		return httpErrorf(http.StatusForbidden, "ElevenLabs denied this request. Check the API key's permissions and IP restrictions.")
	case "rate_limit_error":
		return httpErrorf(http.StatusTooManyRequests, "The ElevenLabs rate or account limit was reached. Try again later.")
	}
	switch status {
	case 401:
		return httpErrorf(http.StatusUnauthorized, "The ElevenLabs API key was rejected. Replace it in Settings.")
	case 402:
		return httpErrorf(http.StatusPaymentRequired, "The ElevenLabs credit limit was reached. Increase this API key's usage cap, add account credits, or wait for the next refresh.")
	case 403:
		return httpErrorf(http.StatusForbidden, "ElevenLabs denied this request. Check the API key's permissions and IP restrictions.")
	case 429:
		return httpErrorf(http.StatusTooManyRequests, "The ElevenLabs rate or account limit was reached. Try again later.")
	case 422:
		return httpErrorf(http.StatusBadRequest, "ElevenLabs rejected the selected voice or speech settings.")
	case 400:
		return httpErrorf(http.StatusBadRequest, "ElevenLabs rejected the speech request.")
	default:
		if status >= 500 {
			return httpErrorf(http.StatusBadGateway, "ElevenLabs is temporarily unavailable.")
		}
		return httpErrorf(http.StatusBadGateway, "ElevenLabs error %s.", strconv.Itoa(status))
	}
}

func providerCodes(body []byte) (code, typ string) {
	var object map[string]any
	if json.Unmarshal(body, &object) != nil {
		return "", ""
	}
	detail, _ := object["detail"].(map[string]any)
	if detail == nil {
		return "", ""
	}
	code, _ = detail["code"].(string)
	if code == "" {
		code, _ = detail["status"].(string)
	}
	typ, _ = detail["type"].(string)
	return code, typ
}
