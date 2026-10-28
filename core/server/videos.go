package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/suakitsu/picpilot-platform/core/config"
	"github.com/suakitsu/picpilot-platform/core/creative"
	"github.com/suakitsu/picpilot-platform/core/jobs"
	"github.com/suakitsu/picpilot-platform/core/security"
	"github.com/suakitsu/picpilot-platform/core/storage"
	"github.com/suakitsu/picpilot-platform/core/store"
	"github.com/suakitsu/picpilot-platform/core/video"
)

func (s *Server) registerVideoRoutes() {
	probe, _ := http.NewRequest(http.MethodGet, "/api/videos/config", nil)
	if _, pattern := s.mux.Handler(probe); pattern == "GET /api/videos/config" {
		return
	}
	s.mux.HandleFunc("GET /api/videos/config", s.auth(s.getVideoConfig))
	s.mux.HandleFunc("GET /api/videos/tasks", s.auth(s.listVideos))
	s.mux.HandleFunc("GET /api/videos/tasks/{id}", s.auth(s.getVideo))
	s.mux.HandleFunc("POST /api/videos/tasks", s.auth(s.createVideo))
	s.mux.HandleFunc("DELETE /api/videos/tasks/{id}", s.auth(s.deleteVideo))
	s.mux.HandleFunc("POST /api/videos/tasks/{id}/cancel", s.auth(s.cancelVideo))
	s.mux.HandleFunc("POST /api/videos/script-preview", s.auth(s.previewVideoScript))
	s.mux.HandleFunc("GET /api/admin/video-config", s.auth(s.admin(s.getVideoAdminConfig)))
	s.mux.HandleFunc("PATCH /api/admin/video-config", s.auth(s.admin(s.patchVideoConfig)))
	s.mux.HandleFunc("GET /api/admin/video-tasks", s.auth(s.admin(s.listAdminVideos)))
	s.mux.HandleFunc("DELETE /api/admin/video-tasks/{id}", s.auth(s.admin(s.deleteAdminVideo)))
	s.mux.HandleFunc("POST /api/public/video-webhooks/apimart", s.receiveVideoWebhook)
	s.mux.HandleFunc("POST /api/public/video-webhooks/apimart/{secret}/callback", s.receiveVideoWebhook)
	// The suffix preserves Gemini's JPG/PNG URL contract for signed local assets.
	s.mux.HandleFunc("GET /api/videos/references/{id}/{filename}", s.serveVideoReference)
}

type videoVariant struct {
	Resolution string `json:"resolution"`
	Duration   int    `json:"duration_seconds"`
	Price      int64  `json:"price"`
	Enabled    bool   `json:"enabled"`
}
type videoModel struct {
	Code               string         `json:"code"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Provider           video.Provider `json:"provider"`
	Model              string         `json:"model"`
	Enabled            bool           `json:"enabled"`
	DefaultResolution  string         `json:"default_resolution"`
	DefaultDuration    int            `json:"default_duration_seconds"`
	DurationSelectable bool           `json:"duration_selectable"`
	AudioToggle        bool           `json:"audio_toggle"`
	MaxReferences      int            `json:"max_reference_images"`
	Ratios             []string       `json:"aspect_ratios"`
	Resolutions        []string       `json:"resolutions"`
	Durations          []int          `json:"durations"`
	Variants           []videoVariant `json:"variants"`
}
type videoModelUpdate struct {
	Code     string         `json:"code"`
	Enabled  bool           `json:"enabled"`
	Variants []videoVariant `json:"variants"`
}
type videoKeySetting struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Enabled     bool   `json:"enabled"`
	Concurrency int    `json:"concurrency"`
}
type videoRuntimeConfig struct {
	Enabled              bool               `json:"enabled"`
	Paid                 bool               `json:"paid_provider_requested"`
	Gemini               bool               `json:"gemini_enabled"`
	Seedance             bool               `json:"seedance_enabled"`
	HFSY                 bool               `json:"hfsy_enabled"`
	Agnes                bool               `json:"agnes_route_enabled"`
	StandardEnabled      bool               `json:"standard_enabled"`
	StandardDuration     int                `json:"standard_duration_seconds"`
	StandardResolution   string             `json:"standard_resolution"`
	StandardPrice        int64              `json:"standard_price"`
	QualityEnabled       bool               `json:"quality_enabled"`
	QualityDuration      int                `json:"quality_duration_seconds"`
	QualityResolution    string             `json:"quality_resolution"`
	QualityPrice         int64              `json:"quality_price"`
	AgnesPercent         int                `json:"agnes_route_percent"`
	AgnesDailyLimit      int                `json:"agnes_daily_limit"`
	AgnesConcurrency     int                `json:"agnes_concurrency"`
	AgnesCircuitFailures int                `json:"agnes_circuit_failures"`
	PollSeconds          int                `json:"poll_seconds"`
	TimeoutMinutes       int                `json:"timeout_minutes"`
	AgnesModel           string             `json:"agnes_model"`
	AgnesKeys            []videoKeySetting  `json:"agnes_keys"`
	Models               []videoModelUpdate `json:"model_configs"`
}

func defaultVideoConfig() videoRuntimeConfig {
	return videoRuntimeConfig{StandardEnabled: true, StandardDuration: 10, StandardResolution: "720p", StandardPrice: 900, QualityEnabled: true, QualityDuration: 10, QualityResolution: "720p", QualityPrice: 1200, AgnesDailyLimit: 40, AgnesConcurrency: 1, AgnesCircuitFailures: 3, PollSeconds: 5, TimeoutMinutes: 30, AgnesModel: "agnes-video-v2.0", Models: []videoModelUpdate{}}
}
func (s *Server) videoConfig(ctx context.Context) (videoRuntimeConfig, error) {
	c := defaultVideoConfig()
	if s.Config.Edition == config.Personal {
		c.Enabled, c.Paid, c.Gemini, c.Seedance, c.HFSY, c.Agnes = true, true, true, true, true, true
		return c, nil
	}
	data, err := s.Store.Setting(ctx, "video_runtime_config")
	if err != nil {
		return c, err
	}
	if err = json.Unmarshal(data, &c); err != nil {
		return c, store.ErrConflict
	}
	return c, nil
}

func videoCatalog(c videoRuntimeConfig) []videoModel {
	ratios := []string{"9:16", "16:9", "1:1", "3:4", "4:3", "adaptive"}
	models := []videoModel{
		{Code: "gemini-omni", Name: "Gemini Omni", Provider: video.Gemini, Model: "gemini-omni-flash-preview", MaxReferences: 3, Ratios: ratios[:2], Resolutions: []string{"720p"}, Durations: []int{10}},
		{Code: "agnes-v2", Name: "Agnes V2.0", Provider: video.Agnes, Model: c.AgnesModel, MaxReferences: 1, Ratios: ratios[:5], Resolutions: []string{"720p"}, Durations: []int{10}},
		{Code: "seedance-mini", Name: "Seedance Mini", Provider: video.Seedance, Model: "doubao-seedance-2.0-mini", MaxReferences: 4, Ratios: ratios, Resolutions: []string{"480p", "720p"}, Durations: []int{5, 8, 10, 15}, AudioToggle: true, DurationSelectable: true},
		{Code: "seedance-fast", Name: "Seedance 2.0 Fast", Provider: video.Seedance, Model: "doubao-seedance-2.0-fast", MaxReferences: 4, Ratios: ratios, Resolutions: []string{"480p", "720p"}, Durations: []int{5, 8, 10, 15}, AudioToggle: true, DurationSelectable: true},
		{Code: "seedance-2", Name: "Seedance 2.0", Provider: video.Seedance, Model: "doubao-seedance-2.0", MaxReferences: 4, Ratios: ratios, Resolutions: []string{"480p", "720p", "1080p", "4k"}, Durations: []int{5, 8, 10, 15}, AudioToggle: true, DurationSelectable: true},
	}
	prices := [][]int64{{950}, {100}, {150, 230, 290, 430, 320, 490, 610, 920}, {290, 460, 570, 850, 610, 980, 1220, 1830}, {360, 570, 710, 1060, 760, 1220, 1520, 2280, 1890, 3030, 3780, 5670, 3850, 6160, 7700, 11550}}
	for i := range models {
		m := &models[i]
		m.Enabled = true
		m.DefaultResolution = "720p"
		m.DefaultDuration = 10
		m.Description = m.Name
		for _, resolution := range m.Resolutions {
			for _, duration := range m.Durations {
				m.Variants = append(m.Variants, videoVariant{resolution, duration, prices[i][len(m.Variants)], true})
			}
		}
		for _, update := range c.Models {
			if update.Code == m.Code {
				m.Enabled = update.Enabled
				for j, v := range m.Variants {
					for _, configured := range update.Variants {
						if v.Resolution == configured.Resolution && v.Duration == configured.Duration {
							m.Variants[j] = configured
						}
					}
				}
			}
		}
	}
	return models
}

type videoCreateInput struct {
	Title          string   `json:"title"`
	Prompt         string   `json:"prompt"`
	ModelCode      string   `json:"model_code"`
	AspectRatio    string   `json:"aspect_ratio"`
	Mode           string   `json:"mode"`
	Duration       int      `json:"duration_seconds"`
	Resolution     string   `json:"resolution"`
	GenerateAudio  *bool    `json:"generate_audio"`
	Seed           *int64   `json:"seed"`
	OriginalPrompt string   `json:"original_prompt"`
	ImageURLs      []string `json:"image_urls"`
	ProjectID      string   `json:"project_id,omitempty"`
}
type videoReference struct {
	URL     string `json:"url,omitempty"`
	AssetID string `json:"asset_id,omitempty"`
}

// This DTO is private durable input, not a public response and not video.Request's
// intentionally redacted JSON representation. No credentials or image bytes occur.
type videoTaskDocument struct {
	Prompt          string           `json:"prompt"`
	OriginalPrompt  string           `json:"original_prompt,omitempty"`
	ModelCode       string           `json:"model_code"`
	Mode            string           `json:"mode"`
	Model           string           `json:"model"`
	Provider        video.Provider   `json:"provider"`
	ProviderID      string           `json:"provider_id"`
	ProviderScope   string           `json:"provider_scope"`
	ProviderVersion int64            `json:"provider_version"`
	AspectRatio     string           `json:"aspect_ratio"`
	Duration        int              `json:"duration_seconds"`
	Resolution      string           `json:"resolution"`
	GenerateAudio   *bool            `json:"generate_audio,omitempty"`
	Seed            *int64           `json:"seed,omitempty"`
	References      []videoReference `json:"references"`
	Fingerprint     string           `json:"fingerprint"`
	ListedPrice     int64            `json:"listed_price"`
	MockMode        bool             `json:"mock_mode"`
	RoutingReason   string           `json:"routing_reason"`
}

func decodeVideoTask(t store.Task) (videoTaskDocument, error) {
	var p jobs.Payload
	var d videoTaskDocument
	if t.Capability != "video" || json.Unmarshal(t.Request, &p) != nil || p.Validate("video") != nil || json.Unmarshal(p.Video, &d) != nil || d.ProviderID == "" || d.ProviderScope == "" {
		return d, store.ErrConflict
	}
	return d, nil
}

type videoSelection struct {
	Model          videoModel
	Variant        videoVariant
	Provider       store.Provider
	Family         video.Provider
	Native, Reason string
}

func (s *Server) selectVideo(c videoRuntimeConfig, providers []store.Provider, input videoCreateInput) (videoSelection, error) {
	var result videoSelection
	code := input.ModelCode
	legacy := code == ""
	if legacy {
		if input.Mode == "quality" || input.Mode == "fast" {
			code = "seedance-fast"
		} else {
			code = "gemini-omni"
		}
	}
	for _, m := range videoCatalog(c) {
		if m.Code == code {
			result.Model = m
		}
	}
	if result.Model.Code == "" || !result.Model.Enabled {
		return result, store.ErrConflict
	}
	resolution, duration := input.Resolution, input.Duration
	if legacy {
		if code == "gemini-omni" {
			if !c.StandardEnabled {
				return result, store.ErrConflict
			}
			resolution, duration = c.StandardResolution, c.StandardDuration
			result.Variant = videoVariant{resolution, duration, c.StandardPrice, true}
		} else {
			if !c.QualityEnabled {
				return result, store.ErrConflict
			}
			resolution, duration = c.QualityResolution, c.QualityDuration
			result.Variant = videoVariant{resolution, duration, c.QualityPrice, true}
		}
	}
	if resolution == "" {
		resolution = result.Model.DefaultResolution
	}
	resolution = strings.ToLower(resolution)
	if duration == 0 {
		duration = result.Model.DefaultDuration
	}
	if !legacy {
		for _, v := range result.Model.Variants {
			if v.Resolution == resolution && v.Duration == duration {
				result.Variant = v
			}
		}
	}
	if !result.Variant.Enabled {
		return result, store.ErrConflict
	}
	ratio := input.AspectRatio
	if ratio == "" {
		ratio = "9:16"
	}
	if !oneOf(ratio, result.Model.Ratios...) {
		return result, store.ErrConflict
	}
	families := []video.Provider{result.Model.Provider}
	if result.Model.Provider == video.Seedance && c.HFSY && oneOf(ratio, "9:16", "16:9", "1:1") && oneOf(resolution, "480p", "720p") {
		families = []video.Provider{video.HFSY, video.Seedance}
	}
	// Legacy standard can use Agnes if no Gemini route exists. New explicit model
	// selections never change families; post-submission fallback never occurs.
	if legacy && code == "gemini-omni" {
		families = append(families, video.Agnes)
	}
	for _, family := range families {
		if (family == video.Gemini && (!c.Paid || !c.Gemini)) || (family == video.Seedance && (!c.Paid || !c.Seedance)) || (family == video.HFSY && !c.HFSY) || (family == video.Agnes && !c.Agnes) {
			continue
		}
		if family == video.HFSY && duration < 4 {
			continue
		}
		native, err := video.ResolveModel(family, code, resolution)
		if family == video.Agnes {
			native, err = c.AgnesModel, nil
		}
		if err != nil {
			continue
		}
		for _, p := range providers {
			if !p.Enabled || p.Capability != "video" || p.ExecutionMode != "http" || (!p.HasAPIKey && !s.Config.DemoMode) || p.Model != native {
				continue
			}
			protocol := map[video.Provider]string{video.Gemini: "apimart_video", video.Seedance: "apimart_video", video.HFSY: "hfsy_video", video.Agnes: "agnes_video"}[family]
			if p.Protocol != protocol {
				continue
			}
			result.Provider, result.Family, result.Native = p, family, native
			result.Reason = "fixed_" + string(family)
			if family == video.HFSY {
				result.Reason = "fixed_hfsy_exact_match"
				minimum := map[string]int64{"sd-2.0-mini": 128, "sd-2.0-fast-480": 213, "sd-2.0-fast": 319, "sd-2.0-480": 319, "sd-2.0": 426}[native]
				result.Variant.Price = max(result.Variant.Price, minimum)
			}
			if family == video.Agnes {
				result.Model.MaxReferences = 1
			} else if legacy {
				result.Model.MaxReferences = 3
			}
			if s.Config.Edition == config.Personal {
				result.Variant.Price = 0
			}
			if s.Config.Edition == config.Platform && !s.Config.DemoMode && result.Variant.Price < 1 {
				return result, store.ErrConflict
			}
			return result, nil
		}
	}
	return result, store.ErrNotFound
}

func (s *Server) videoPublicConfig(c videoRuntimeConfig, providers []store.Provider) map[string]any {
	models := videoCatalog(c)
	for i := range models {
		m := &models[i]
		available := false
		ratios := []string{}
		for _, ratio := range m.Ratios {
			found := false
			for _, v := range m.Variants {
				_, err := s.selectVideo(c, providers, videoCreateInput{ModelCode: m.Code, Resolution: v.Resolution, Duration: v.Duration, AspectRatio: ratio})
				if err == nil {
					found = true
				}
			}
			if found {
				ratios = append(ratios, ratio)
			}
		}
		for j := range m.Variants {
			v := &m.Variants[j]
			enabled := false
			for _, ratio := range m.Ratios {
				selection, err := s.selectVideo(c, providers, videoCreateInput{ModelCode: m.Code, Resolution: v.Resolution, Duration: v.Duration, AspectRatio: ratio})
				if err == nil {
					enabled = true
					v.Price = max(v.Price, selection.Variant.Price)
				}
			}
			v.Enabled = v.Enabled && enabled
			available = available || v.Enabled
			if s.Config.Edition == config.Personal {
				v.Price = 0
			}
		}
		m.Enabled = m.Enabled && available
		m.Ratios = ratios
	}
	standard, standardErr := s.selectVideo(c, providers, videoCreateInput{Mode: "standard", AspectRatio: "9:16"})
	quality, qualityErr := s.selectVideo(c, providers, videoCreateInput{Mode: "quality", AspectRatio: "9:16"})
	return map[string]any{"enabled": c.Enabled, "mock_mode": s.Config.DemoMode, "aspect_ratios": []string{"9:16", "16:9"}, "default_model": "gemini-omni", "model_configs": models,
		"standard": map[string]any{"enabled": c.StandardEnabled && standardErr == nil, "duration_seconds": c.StandardDuration, "resolution": c.StandardResolution, "price": standard.Variant.Price},
		"quality":  map[string]any{"enabled": c.QualityEnabled && qualityErr == nil, "duration_seconds": c.QualityDuration, "resolution": c.QualityResolution, "price": quality.Variant.Price}}
}
func (s *Server) getVideoConfig(w http.ResponseWriter, r *http.Request) {
	c, err := s.videoConfig(r.Context())
	if err != nil {
		s.videoError(w, r, err)
		return
	}
	providers, err := s.Store.Providers(r.Context(), s.providerScope(r))
	if err != nil {
		s.videoError(w, r, err)
		return
	}
	ok(w, r, s.videoPublicConfig(c, providers))
}

var videoRequestKey = regexp.MustCompile(`^[A-Za-z0-9._:-]{16,128}$`)

func validVideoPrompt(prompt string) bool {
	return utf8.ValidString(prompt) && len(prompt) <= 16000 && len(utf16.Encode([]rune(prompt))) >= 4 && len(utf16.Encode([]rune(prompt))) <= 4000
}

type videoUpload struct {
	Data []byte
	Name string
}

func readVideoCreate(w http.ResponseWriter, r *http.Request) (videoCreateInput, []videoUpload, bool) {
	var in videoCreateInput
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		valid := decode(w, r, &in)
		return in, nil, valid
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4*storage.MaxImageBytes+(1<<20))
	mr, err := r.MultipartReader()
	if err != nil {
		fail(w, r, 415, "CONTENT_TYPE_INVALID", "Expected multipart/form-data or application/json")
		return in, nil, false
	}
	fields := map[string]string{}
	uploads := []videoUpload{}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			fail(w, r, 400, "INVALID_MULTIPART", "Invalid or oversized video upload")
			return in, nil, false
		}
		name := part.FormName()
		if name == "images" {
			if len(uploads) >= 4 {
				part.Close()
				fail(w, r, 400, "INVALID_REQUEST", "Too many reference images")
				return in, nil, false
			}
			data, readErr := io.ReadAll(io.LimitReader(part, storage.MaxImageBytes+1))
			filename := part.FileName()
			part.Close()
			if readErr != nil || int64(len(data)) > storage.MaxImageBytes {
				fail(w, r, 413, "IMAGE_TOO_LARGE", "Reference image exceeds the limit")
				return in, nil, false
			}
			if len(data) > 0 {
				uploads = append(uploads, videoUpload{data, filename})
			}
			continue
		}
		value, err := io.ReadAll(io.LimitReader(part, 20001))
		part.Close()
		if err != nil || len(value) > 20000 {
			fail(w, r, 400, "INVALID_REQUEST", "Invalid video field")
			return in, nil, false
		}
		if name == "image_urls" {
			if len(in.ImageURLs) >= 4 {
				fail(w, r, 400, "INVALID_REQUEST", "Too many reference images")
				return in, nil, false
			}
			in.ImageURLs = append(in.ImageURLs, string(value))
			continue
		}
		if !oneOf(name, "title", "prompt", "aspect_ratio", "mode", "model_code", "duration_seconds", "resolution", "generate_audio", "seed", "original_prompt", "project_id") {
			fail(w, r, 400, "INVALID_REQUEST", "Unexpected video field")
			return in, nil, false
		}
		if _, exists := fields[name]; exists {
			fail(w, r, 400, "INVALID_REQUEST", "Duplicate video field")
			return in, nil, false
		}
		fields[name] = string(value)
	}
	in.Title, in.Prompt, in.AspectRatio, in.Mode, in.ModelCode, in.Resolution, in.OriginalPrompt, in.ProjectID = fields["title"], fields["prompt"], fields["aspect_ratio"], fields["mode"], fields["model_code"], fields["resolution"], fields["original_prompt"], fields["project_id"]
	if v := fields["duration_seconds"]; v != "" {
		in.Duration, err = strconv.Atoi(v)
		if err != nil {
			fail(w, r, 400, "INVALID_REQUEST", "Invalid duration")
			return in, nil, false
		}
	}
	if v := fields["generate_audio"]; v != "" {
		var b bool
		b, err = strconv.ParseBool(v)
		if err != nil {
			fail(w, r, 400, "INVALID_REQUEST", "Invalid audio option")
			return in, nil, false
		}
		in.GenerateAudio = &b
	}
	if v := fields["seed"]; v != "" {
		var seed int64
		seed, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			fail(w, r, 400, "INVALID_REQUEST", "Invalid seed")
			return in, nil, false
		}
		in.Seed = &seed
	}
	return in, uploads, true
}

func (s *Server) createVideo(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if !videoRequestKey.MatchString(key) {
		fail(w, r, 400, "IDEMPOTENCY_KEY_INVALID", "Use a 16-128 character Idempotency-Key")
		return
	}
	in, uploads, valid := readVideoCreate(w, r)
	if !valid {
		return
	}
	in.Prompt = strings.TrimSpace(in.Prompt)
	in.Title = strings.TrimSpace(in.Title)
	in.ModelCode = strings.TrimSpace(in.ModelCode)
	in.Resolution = strings.ToLower(strings.TrimSpace(in.Resolution))
	if in.AspectRatio == "" {
		in.AspectRatio = "9:16"
	}
	if in.Mode == "" {
		in.Mode = "standard"
	}
	if in.Mode == "fast" {
		in.Mode = "quality"
	}
	if !validVideoPrompt(in.Prompt) || len(in.OriginalPrompt) > 16000 || len(in.Title) > 1000 || len(in.ProjectID) > 128 || in.Duration < 0 || (in.Seed != nil && (*in.Seed < 0 || *in.Seed > 2147483647)) {
		fail(w, r, 400, "INVALID_REQUEST", "Invalid video description or parameters")
		return
	}
	if in.Title == "" {
		runes := []rune(in.Prompt)
		in.Title = string(runes[:min(len(runes), 32)])
	}
	if len(in.ImageURLs)+len(uploads) < 1 || len(in.ImageURLs)+len(uploads) > 4 {
		fail(w, r, 400, "INVALID_REQUEST", "Select one to four reference images")
		return
	}
	hashes := make([]string, len(uploads))
	for i, upload := range uploads {
		sum := sha256.Sum256(upload.Data)
		hashes[i] = hex.EncodeToString(sum[:])
	}
	fingerprintData, _ := json.Marshal(struct {
		Input   videoCreateInput
		Uploads []string
	}{in, hashes})
	fingerprintHash := sha256.Sum256(fingerprintData)
	fingerprint := hex.EncodeToString(fingerprintHash[:])
	u := currentUser(r)
	if existing, err := s.Store.VideoTaskByKey(r.Context(), u.ID, key); err == nil {
		d, err := decodeVideoTask(existing.Task)
		if err != nil || d.Fingerprint != fingerprint {
			fail(w, r, 409, "CONFLICT", "Idempotency-Key already identifies different input")
			return
		}
		ok(w, r, s.videoTaskView(r.Context(), existing, false))
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		s.videoError(w, r, err)
		return
	}
	c, err := s.videoConfig(r.Context())
	if err != nil {
		s.videoError(w, r, err)
		return
	}
	if !c.Enabled {
		fail(w, r, 403, "VIDEO_DISABLED", "Video generation is disabled")
		return
	}
	providers, err := s.Store.Providers(r.Context(), s.providerScope(r))
	if err != nil {
		s.videoError(w, r, err)
		return
	}
	selected, err := s.selectVideo(c, providers, in)
	if err != nil {
		fail(w, r, 400, "VIDEO_SPEC_UNAVAILABLE", "The selected video specification has no enabled provider")
		return
	}
	if len(in.ImageURLs)+len(uploads) > selected.Model.MaxReferences {
		fail(w, r, 400, "INVALID_REQUEST", "Too many references for this model")
		return
	}
	refs := []videoReference{}
	for _, raw := range in.ImageURLs {
		ref, err := s.videoReference(r.Context(), u.ID, strings.TrimSpace(raw), selected.Family)
		if err != nil {
			fail(w, r, 400, "INVALID_REFERENCE", "Use an owned reference or a valid public image URL")
			return
		}
		refs = append(refs, ref)
	}
	// Validate all uploaded bytes before persisting any asset or charging a task.
	for _, upload := range uploads {
		if _, err := storage.NormalizeImage(bytes.NewReader(upload.Data)); err != nil {
			fail(w, r, 400, "INVALID_REFERENCE", "Use valid JPEG, PNG or WebP reference images")
			return
		}
	}
	// Both form fields are caller-authored. OriginalPrompt must not let a caller
	// bypass screening of an edited script; server-composed instructions stay out.
	userText := in.Prompt
	if original := strings.TrimSpace(in.OriginalPrompt); original != "" && original != in.Prompt {
		userText = original + "\n" + in.Prompt
	}
	if err := s.screenUserInput(r.Context(), u.ID, in.ProjectID, "product_video", key, userText); err != nil {
		s.databaseError(w, r, err)
		return
	}
	var uploadedIDs []string
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		keys, err := s.Store.DiscardVideoReferences(ctx, u.ID, uploadedIDs)
		if err != nil {
			s.Logger.Error("unused video reference cleanup failed", "sqlstate", store.SQLState(err))
			return
		}
		for _, key := range keys {
			_ = s.Files.Remove(key)
		}
	}()
	for _, upload := range uploads {
		a, err := s.saveImage(r.Context(), u.ID, in.ProjectID, "", "video_reference", upload.Name, bytes.NewReader(upload.Data))
		if err != nil {
			s.videoError(w, r, err)
			return
		}
		uploadedIDs = append(uploadedIDs, a.ID)
		refs = append(refs, videoReference{AssetID: a.ID})
	}
	d := videoTaskDocument{Prompt: in.Prompt, OriginalPrompt: in.OriginalPrompt, ModelCode: selected.Model.Code, Mode: in.ModelCode, Model: selected.Native, Provider: selected.Family, ProviderID: selected.Provider.ID, ProviderScope: selected.Provider.OwnerScope, ProviderVersion: selected.Provider.Version, AspectRatio: in.AspectRatio, Duration: selected.Variant.Duration, Resolution: selected.Variant.Resolution, References: refs, Fingerprint: fingerprint, ListedPrice: selected.Variant.Price, MockMode: s.Config.DemoMode, RoutingReason: selected.Reason}
	if d.Mode == "" {
		d.Mode = in.Mode
	}
	if selected.Family == video.Seedance {
		audio := in.GenerateAudio == nil || *in.GenerateAudio
		d.GenerateAudio = &audio
		d.Seed = in.Seed
	}
	videoJSON, _ := json.Marshal(d)
	payload, _ := json.Marshal(jobs.Payload{Version: 1, Video: videoJSON})
	cost := selected.Variant.Price
	if s.Config.DemoMode || s.Config.Edition == config.Personal {
		cost = 0
	}
	t, _, err := s.Store.SubmitTask(r.Context(), store.SubmitTask{UserID: u.ID, ProjectID: in.ProjectID, Capability: "video", Kind: "product_video", Title: in.Title, Request: payload, RequestIdentity: fingerprintData, IdempotencyKey: key, CreditsCost: cost})
	if errors.Is(err, store.ErrConflict) {
		if existing, e := s.Store.VideoTaskByKey(r.Context(), u.ID, key); e == nil {
			old, e := decodeVideoTask(existing.Task)
			if e == nil && old.Fingerprint == fingerprint {
				ok(w, r, s.videoTaskView(r.Context(), existing, false))
				return
			}
		}
	}
	if err != nil {
		s.videoError(w, r, err)
		return
	}
	v, err := s.Store.VideoTask(r.Context(), u.ID, t.ID)
	if err != nil {
		s.videoError(w, r, err)
		return
	}
	ok(w, r, s.videoTaskView(r.Context(), v, false))
}

func (s *Server) videoReference(ctx context.Context, userID, raw string, family video.Provider) (videoReference, error) {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 8192 || u.User != nil || u.Fragment != "" || u.Hostname() == "" || !oneOf(u.Scheme, "https", "http") {
		return videoReference{}, store.ErrConflict
	}
	base, _ := url.Parse(s.Config.PublicBaseURL)
	if u.Host == base.Host && u.Scheme == base.Scheme && (strings.HasPrefix(u.Path, "/api/files/assets/") || strings.HasPrefix(u.Path, "/api/videos/references/")) {
		id := strings.TrimPrefix(u.Path, "/api/files/assets/")
		if strings.HasPrefix(u.Path, "/api/videos/references/") {
			id = strings.Split(strings.TrimPrefix(u.Path, "/api/videos/references/"), "/")[0]
		}
		a, err := s.Store.OwnedAsset(ctx, userID, id)
		if err != nil || !oneOf(a.ContentType, "image/png", "image/jpeg", "image/webp") || (family == video.Gemini && a.ContentType == "image/webp") {
			return videoReference{}, store.ErrNotFound
		}
		return videoReference{AssetID: id}, nil
	}
	if u.Scheme != "https" && !s.Config.AllowPrivateProviders {
		return videoReference{}, store.ErrConflict
	}
	if family == video.Gemini && !oneOf(strings.ToLower(path.Ext(u.Path)), ".jpg", ".jpeg", ".png") {
		return videoReference{}, store.ErrConflict
	}
	return videoReference{URL: raw}, nil
}
func (s *Server) videoAssetURL(a store.Asset) string {
	u, _ := url.Parse(storage.AssetURL(s.Config.PublicBaseURL, s.Config.WorkerSigningKey, a.ID, a.UserID, time.Now()))
	ext := "png"
	if a.ContentType == "image/jpeg" {
		ext = "jpg"
	}
	if a.ContentType == "image/webp" {
		ext = "webp"
	}
	u.Path = "/api/videos/references/" + a.ID + "/reference." + ext
	return u.String()
}
func (s *Server) serveVideoReference(w http.ResponseWriter, r *http.Request) {
	if !oneOf(r.PathValue("filename"), "reference.png", "reference.jpg", "reference.webp") {
		fail(w, r, 404, "NOT_FOUND", "File not found")
		return
	}
	s.serveAsset(w, r)
}

// CompileVideoTask reconstructs the adapter input from jobs.Payload, resolves
// only owned assets, refreshes their signed URLs, and enforces the pinned route.
// It does not decrypt credentials or contact a model. Parent execution does both.
func (s *Server) CompileVideoTask(ctx context.Context, t store.Task) (video.Request, store.Provider, error) {
	d, err := decodeVideoTask(t)
	if err != nil {
		return video.Request{}, store.Provider{}, err
	}
	expected := "platform"
	if s.Config.Edition == config.Personal {
		expected = t.UserID
	}
	if d.ProviderScope != expected {
		return video.Request{}, store.Provider{}, store.ErrNotFound
	}
	p, err := s.Store.Provider(ctx, expected, d.ProviderID)
	if err != nil {
		return video.Request{}, p, err
	}
	return s.compileVideoWithProvider(ctx, t, d, p)
}

// CompileVideoAttempt uses the current provider snapshot captured atomically by
// ReserveAttempt. A pre-reservation provider lookup is not an execution snapshot.
func (s *Server) CompileVideoAttempt(ctx context.Context, t store.Task, a store.Attempt) (video.Request, error) {
	d, err := decodeVideoTask(t)
	if err != nil || a.TaskID != t.ID || a.ProviderID != a.Provider.ID || a.ProviderScope != a.Provider.OwnerScope || a.Model != a.Provider.Model || a.Protocol != a.Provider.Protocol {
		return video.Request{}, store.ErrConflict
	}
	r, _, err := s.compileVideoWithProvider(ctx, t, d, a.Provider)
	return r, err
}

func (s *Server) compileVideoWithProvider(ctx context.Context, t store.Task, d videoTaskDocument, p store.Provider) (video.Request, store.Provider, error) {
	expected := "platform"
	if s.Config.Edition == config.Personal {
		expected = t.UserID
	}
	protoco€ç˚ﬁ⁄$z{-ÆÈ‹j◊ùCH", "/api/admin/video-config", token, map[string]any{"hfsy_enabled": true, "hfsy_api_key": "fixture-hfsy-runtime"}); status != 200 {
					t.Fatal("HFSY setup")
				}
			}
			var creates, polls, downloads atomic.Int32
			media := videoMockMP4()
			cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				downloads.Add(1)
				if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
					t.Error("credentials forwarded to result origin")
				}
				w.Header().Set("Content-Type", "video/mp4")
				_, _ = w.Write(media)
			}))
			defer cdn.Close()
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer fixture-") {
					t.Error("mock API credential not applied")
				}
				if r.Method == "POST" {
					creates.Add(1)
					_, _ = io.Copy(io.Discard, r.Body)
					switch tc.family {
					case video.Agnes:
						_, _ = w.Write([]byte(`{"video_id":"runtime-id"}`))
					case video.HFSY:
						_, _ = w.Write([]byte(`{"data":{"task_id":"runtime-id"}}`))
					default:
						_, _ = w.Write([]byte(`{"code":200,"data":[{"task_id":"runtime-id"}]}`))
					}
					return
				}
				polls.Add(1)
				url := cdn.URL + "/result.mp4"
				switch tc.family {
				case video.Agnes:
					_ = json.NewEncoder(w).Encode(map[string]any{"video_id": "runtime-id", "status": "success", "metadata": map[string]any{"url": url, "size_mapping": map[string]int{"width": 1280, "height": 720}}})
				case video.HFSY:
					_ = json.NewEncoder(w).Encode(map[string]any{"task_id": "runtime-id", "status": "completed", "result_url": url})
				default:
					_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "runtime-id", "status": "completed", "result": map[string]any{"videos": []any{map[string]any{"url": []string{url}}}}}})
				}
			}))
			defer api.Close()
			input := videoInputFixture()
			input["model_code"] = tc.code
			id := videoResponseData(t, videoJSONRequest(t, s, token, "runtime-provider-key", input))["id"].(string)
			videoConfigureEndpoint(t, s, u, id, api.URL)
			client := videoRuntimeClient(t)
			lease := videoRuntimeClaim(t, s, id)
			s.executeVideoSubmission(ctx, lease, client)
			claim := videoRuntimePoll(t, s, id)
			if claim.Provider.ID == "" || claim.Submission.Provider != tc.family {
				t.Fatal("durable provider identity lost")
			}
			s.executeVideoPollLease(ctx, claim, client)
			v, err := s.Store.VideoTask(ctx, u.ID, id)
			if err != nil || v.Status != "succeeded" || v.Attempt != 1 || v.RefundedAt != nil || strings.Contains(string(v.Result), cdn.URL) {
				t.Fatal("runtime did not safely complete", err, v.Status)
			}
			if creates.Load() != 1 || polls.Load() != 1 || downloads.Load() != 1 {
				t.Fatal("unexpected runtime network counts", creates.Load(), polls.Load(), downloads.Load())
			}
			if tc.edition == config.Personal {
				var ledger int
				if err := s.Store.Pool.QueryRow(ctx, "SELECT count(*) FROM credit_ledger").Scan(&ledger); err != nil || ledger != 0 || v.CreditsCost != 0 || claim.Provider.OwnerScope != u.ID {
					t.Fatal("personal runtime touched platform billing or credentials", err)
				}
			}
			view := s.videoTaskView(ctx, v, false)
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, httptest.NewRequest("GET", view["result_url"].(string), nil))
			if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), media) {
				t.Fatal("stored video did not round trip through signed serving", w.Code)
			}
		})
	}
}

func TestPostgresVideoRuntimeAmbiguityAndDefinitiveRejection(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
		refund int
	}{
		{"server-error", 503, `{"error":"fixture upstream unavailable"}`, "uncertain", 0},
		{"missing-id", 200, `{}`, "uncertain", 0},
		{"authentication", 403, `{"error":"fixture denial"}`, "failed", 1},
		{"contradictory-known-id", 200, `{"code":500,"data":[{"task_id":"known-despite-error"}]}`, "polling", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, token, u := videoHTTPFixture(t, config.Platform)
			var calls atomic.Int32
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer api.Close()
			id := videoResponseData(t, videoJSONRequest(t, s, token, "runtime-ambiguity-key", videoInputFixture()))["id"].(string)
			videoConfigureEndpoint(t, s, u, id, api.URL)
			lease := videoRuntimeClaim(t, s, id)
			client := videoRuntimeClient(t)
			s.executeVideoSubmission(context.Background(), lease, client)
			s.executeVideoSubmission(context.Background(), lease, client)
			v, err := s.Store.VideoTask(context.Background(), u.ID, id)
			if err != nil || v.Status != tc.want || calls.Load() != 1 {
				t.Fatal("submission outcome/replay", err, v.Status, calls.Load())
			}
			var refunds int
			if err := s.Store.Pool.QueryRow(context.Background(), "SELECT count(*) FROM credit_ledger WHERE biz_id=$1 AND biz_type='generation_refund'", id).Scan(&refunds); err != nil || refunds != tc.refund {
				t.Fatal("ambiguous/rejected settlement", err, refunds)
			}
			if _, err := s.Store.ClaimTaskCapabilities(context.Background(), "duplicate", time.Minute, []string{"video"}); !errors.Is(err, store.ErrNotFound) {
				t.Fatal("paid task returned to submission queue", err)
			}
		})
	}
}

func TestPostgresVideoRuntimePollStorageAndRestartRecovery(t *testing.T) {
	s, token, u := videoHTTPFixture(t, config.Platform)
	ctx := context.Background()
	var creates, polls, downloads atomic.Int32
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := downloads.Add(1)
		w.Header().Set("Content-Type", "video/mp4")
		if n == 1 {
			_, _ = w.Write([]byte("<html>not a video</html>"))
			return
		}
		_, _ = w.Write(videoMockMP4())
	}))
	defer cdn.Close()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			creates.Add(1)
			_, _ = w.Write([]byte(`{"data":[{"task_id":"restart-id"}]}`))
			return
		}
		if polls.Add(1) == 1 {
			w.WriteHeader(503)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "restart-id", "status": "completed", "url": cdn.URL + "/result.mp4"}})
	}))
	defer api.Close()
	id := videoResponseData(t, videoJSONRequest(t, s, token, "runtime-recovery-key", videoInputFixture()))["id"].(string)
	videoConfigureEndpoint(t, s, u, id, api.URL)
	client := videoRuntimeClient(t)
	s.executeVideoSubmission(ctx, videoRuntimeClaim(t, s, id), client)
	s.executeVideoPoll(ctx, videoRuntimePoll(t, s, id), client)
	v, err := s.Store.VideoTask(ctx, u.ID, id)
	if err != nil || v.Status != "polling" || v.RefundedAt != nil || !v.NextRunAt.After(time.Now()) {
		t.Fatal("poll transport error lost known task/backoff", err)
	}
	s.executeVideoPoll(ctx, videoRuntimePoll(t, s, id), client)
	v, err = s.Store.VideoTask(ctx, u.ID, id)
	if err != nil || v.Status != "polling" || len(v.PendingResult) == 0 || !v.NextRunAt.After(time.Now()) {
		t.Fatal("invalid result body erased completion or spun without backoff", err)
	}
	claim := videoRuntimePoll(t, s, id)
	g, err := s.videoGateway(claim.Provider, claim.Submission.Provider, client)
	if err != nil {
		t.Fatal(err)
	}
	asset, err := s.downloadVideoAsset(ctx, claim.Task, g, claim.Pending.ResultURL)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate shutdown after durable storage but before terminal settlement.
	if _, err := s.Store.Pool.Exec(ctx, "UPDATE tasks SET lease_until=now()-interval '1 second',next_run_at=now() WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	s.executeVideoPoll(ctx, videoRuntimePoll(t, s, id), client)
	v, err = s.Store.VideoTask(ctx, u.ID, id)
	if err != nil || v.Status != "succeeded" || !strings.Contains(string(v.Result), asset.ID) || creates.Load() != 1 || polls.Load() != 2 || downloads.Load() != 2 {
		t.Fatal("restart failed to reuse verified result", err, v.Status, creates.Load(), polls.Load(), downloads.Load())
	}
	var assets int
	if err := s.Store.Pool.QueryRow(ctx, "SELECT count(*) FROM assets WHERE task_id=$1", id).Scan(&assets); err != nil || assets != 1 {
		t.Fatal("recovery duplicated result assets", err, assets)
	}
}

func TestPostgresVideoRunTasksShutdownJoinsStreamingDownload(t *testing.T) {
	s, token, u := videoHTTPFixture(t, config.Platform)
	if status, _, _ := request(t, s, "PATCH", "/api/admin/video-config", token, map[string]any{"poll_seconds": 2}); status != 200 {
		t.Fatal("poll config")
	}
	started := make(chan struct{}, 1)
	closed := make(chan struct{}, 1)
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write(videoMockMP4())
		w.(http.Flusher).Flush()
		started <- struct{}{}
		<-r.Context().Done()
		closed <- struct{}{}
	}))
	defer cdn.Close()
	var creates atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			creates.Add(1)
			_, _ = w.Write([]byte(`{"data":[{"task_id":"shutdown-id"}]}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "shutdown-id", "status": "completed", "url": cdn.URL + "/result.mp4"}})
	}))
	defer api.Close()
	id := videoResponseData(t, videoJSONRequest(t, s, token, "runtime-shutdown-key", videoInputFixture()))["id"].(string)
	videoConfigureEndpoint(t, s, u, id, api.URL)
	imageTask, _, err := s.Store.SubmitTask(context.Background(), store.SubmitTask{UserID: u.ID, Capability: "image", Kind: "fixture", IdempotencyKey: "other-capability", Request: json.RawMessage(`{"version":1,"image":{"prompt":"fixture"}}`)})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.RunVideoTasks(ctx) }()
	defer cancel()
	select {
	case <-started:
	case <-time.After(15 * time.Second):
		cancel()
		<-done
		t.Fatal("runtime did not reach mock download")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(12 * time.Second):
		t.Fatal("video runtime did not join on shutdown")
	}
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("streaming result request outlived shutdown")
	}
	v, err := s.Store.VideoTask(context.Background(), u.ID, id)
	if err != nil || v.Status != "polling" || v.RefundedAt != nil || creates.Load() != 1 {
		t.Fatal("shutdown changed paid settlement", err, v.Status)
	}
	untouched, err := s.Store.Task(context.Background(), u.ID, imageTask.ID)
	if err != nil || untouched.Status != "queued" || untouched.Attempt != 0 {
		t.Fatal("video runtime claimed image work", err)
	}
	var assets int
	if err := s.Store.Pool.QueryRow(context.Background(), "SELECT count(*) FROM assets WHERE task_id=$1", id).Scan(&assets); err != nil || assets != 0 {
		t.Fatal("partial download published an asset", err, assets)
	}
}

func TestPostgresVideoRuntimeCircuitCapacityAndDemoGuards(t *testing.T) {
	s, token, u := videoHTTPFixture(t, config.Platform)
	ctx := context.Background()
	var creates atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		creates.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"task_id":"capacity-id"}]}`))
	}))
	defer api.Close()
	first := videoResponseData(t, videoJSONRequest(t, s, token, "capacity-first-key", videoInputFixture()))["id"].(string)
	task := videoConfigureEndpoint(t, s, u, first, api.URL)
	_, p, err := s.CompileVideoTask(ctx, task)
	if err != nil {
		t.Fatal(err)
	}
	p.MaxConcurrency = 1
	p, err = s.Store.PutProvider(ctx, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := videoRuntimeClient(t)
	if err := s.Store.RecordProviderOutcome(ctx, p, "MOCK_AUTH_FAILURE", true); err != nil {
		t.Fatal(err)
	}
	s.executeVideoSubmission(ctx, videoRuntimeClaim(t, s, first), client)
	v, err := s.Store.VideoTask(ctx, u.ID, first)
	if err != nil || v.Status != "queued" || v.Attempt != 0 || creates.Load() != 0 {
		t.Fatal("open circuit made a paid request or discarded queue", err)
	}
	if err := s.Store.RecordProviderOutcome(ctx, p, "", false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.Pool.Exec(ctx, "UPDATE tasks SET next_run_at=now() WHERE id=$1", first); err != nil {
		t.Fatal(err)
	}
	s.executeVideoSubmission(ctx, videoRuntimeClaim(t, s, first), client)
	second := videoResponseData(t, videoJSONRequest(t, s, token, "capacity-second-key", videoInputFixture()))["id"].(string)
	s.executeVideoSubmission(ctx, videoRuntimeClaim(t, s, second), client)
	v, err = s.Store.VideoTask(ctx, u.ID, second)
	if err != nil || v.Status != "queued" || v.Attempt != 0 || creates.Load() != 1 {
		t.Fatal("known-id attempt did not count against provider capacity", err)
	}
	if _, err := s.Store.CancelVideoTask(ctx, u.ID, second); err != nil {
		t.Fatal(err)
	}
	s.Config.DemoMode = true
	demo := videoResponseData(t, videoJSONRequest(t, s, token, "demo-no-paid-request", videoInputFixture()))["id"].(string)
	// Toggling demo off before execution must not turn a free queued job paid.
	s.Config.DemoMode = false
	s.executeVideoSubmission(ctx, videoRuntimeClaim(t, s, demo), client)
	v, err = s.Store.VideoTask(ctx, u.ID, demo)
	if err != nil || v.Status != "failed" || v.CreditsCost != 0 || creates.Load() != 1 || v.ErrorCode != "VIDEO_DEMO_EXECUTION_UNAVAILABLE" {
		t.Fatal("demo task became paid after restart/config change", err)
	}
}

func TestPostgresVideoPersonalLocalOriginUsesHTTPSProviderWithoutAdminConfig(t *testing.T) {
	s, token, u := videoHTTPFixture(t, config.Personal)
	s.Config.AllowPrivateProviders = false
	var calls atomic.Int32
	api := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var input map[string]any
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			t.Error("invalid mock request")
		}
		if input["webhook"] != nil || input["callback_url"] != nil || input["webhook_url"] != nil {
			t.Error("local personal origin advertised as a public callback")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"task_id":"personal-https-id"}]}`))
	}))
	defer api.Close()
	roots := x509.NewCertPool()
	roots.AddCert(api.Certificate())
	// Only the local test transport allows loopback; the gateway keeps its
	// production HTTPS requirement and the personal app still uses localhost.
	client, err := provider.NewHTTPClient(provider.Options{AllowLocalDevelopment: true, TrustedRoots: roots, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseIdleConnections()
	id := videoResponseData(t, videoJSONRequest(t, s, token, "personal-local-https", videoInputFixture()))["id"].(string)
	videoConfigureEndpoint(t, s, u, id, api.URL)
	s.executeVideoSubmission(context.Background(), videoRuntimeClaim(t, s, id), client)
	v, err := s.Store.VideoTask(context.Background(), u.ID, id)
	if err != nil || v.Status != "polling" || calls.Load() != 1 || v.CreditsCost != 0 {
		t.Fatal("local personal origin blocked owner-configured HTTPS provider", err, v.Status)
	}
	var configRows int
	if err := s.Store.Pool.QueryRow(context.Background(), "SELECT count(*) FROM app_settings WHERE key='video_runtime_config'").Scan(&configRows); err != nil || configRows != 0 {
		t.Fatal("personal path used platform-global configuration", err)
	}
}
