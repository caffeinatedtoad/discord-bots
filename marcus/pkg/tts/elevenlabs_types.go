package tts

type SpeakerSeparation struct {
	VoiceID  string `json:"voice_id"`
	SampleID string `json:"sample_id"`
	Status   string `json:"status"`
}

type Sample struct {
	SampleID                string            `json:"sample_id"`
	FileName                string            `json:"file_name"`
	MimeType                string            `json:"mime_type"`
	SizeBytes               int               `json:"size_bytes"`
	Hash                    string            `json:"hash"`
	DurationSecs            float64           `json:"duration_secs"`
	RemoveBackgroundNoise   bool              `json:"remove_background_noise"`
	HasIsolatedAudio        bool              `json:"has_isolated_audio"`
	HasIsolatedAudioPreview bool              `json:"has_isolated_audio_preview"`
	SpeakerSeparation       SpeakerSeparation `json:"speaker_separation"`
	TrimStart               int               `json:"trim_start"`
	TrimEnd                 int               `json:"trim_end"`
}

type FineTuningState struct {
	ElevenMultilingualV2 string `json:"eleven_multilingual_v2"`
}

type FineTuning struct {
	IsAllowedToFineTune         bool            `json:"is_allowed_to_fine_tune"`
	State                       FineTuningState `json:"state"`
	VerificationFailures        []any           `json:"verification_failures"`
	VerificationAttemptsCount   int             `json:"verification_attempts_count"`
	ManualVerificationRequested bool            `json:"manual_verification_requested"`
}

type Labels struct {
	Accent      string `json:"accent"`
	Age         string `json:"age"`
	Description string `json:"description"`
	Gender      string `json:"gender"`
	UseCase     string `json:"use_case"`
}

type Settings struct {
	Stability       int  `json:"stability"`
	UseSpeakerBoost bool `json:"use_speaker_boost"`
	SimilarityBoost int  `json:"similarity_boost"`
	Style           int  `json:"style"`
	Speed           int  `json:"speed"`
}

type SharingLabels struct {
	Accent string `json:"accent"`
	Gender string `json:"gender"`
}

type ModerationCheck struct {
	DateCheckedUnix  int       `json:"date_checked_unix"`
	NameValue        string    `json:"name_value"`
	NameCheck        bool      `json:"name_check"`
	DescriptionValue string    `json:"description_value"`
	DescriptionCheck bool      `json:"description_check"`
	SampleIds        []string  `json:"sample_ids"`
	SampleChecks     []float64 `json:"sample_checks"`
	CaptchaIds       []string  `json:"captcha_ids"`
	CaptchaChecks    []float64 `json:"captcha_checks"`
}

type ReaderRestriction struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

type Sharing struct {
	Status                  string              `json:"status"`
	HistoryItemSampleID     string              `json:"history_item_sample_id"`
	DateUnix                int                 `json:"date_unix"`
	WhitelistedEmails       []string            `json:"whitelisted_emails"`
	PublicOwnerID           string              `json:"public_owner_id"`
	OriginalVoiceID         string              `json:"original_voice_id"`
	FinancialRewardsEnabled bool                `json:"financial_rewards_enabled"`
	FreeUsersAllowed        bool                `json:"free_users_allowed"`
	LiveModerationEnabled   bool                `json:"live_moderation_enabled"`
	Rate                    float64             `json:"rate"`
	NoticePeriod            int                 `json:"notice_period"`
	DisableAtUnix           int                 `json:"disable_at_unix"`
	VoiceMixingAllowed      bool                `json:"voice_mixing_allowed"`
	Featured                bool                `json:"featured"`
	Category                string              `json:"category"`
	ReaderAppEnabled        bool                `json:"reader_app_enabled"`
	LikedByCount            int                 `json:"liked_by_count"`
	ClonedByCount           int                 `json:"cloned_by_count"`
	Name                    string              `json:"name"`
	Description             string              `json:"description"`
	Labels                  SharingLabels       `json:"labels"`
	ReviewStatus            string              `json:"review_status"`
	EnabledInLibrary        bool                `json:"enabled_in_library"`
	ModerationCheck         ModerationCheck     `json:"moderation_check"`
	ReaderRestrictedOn      []ReaderRestriction `json:"reader_restricted_on"`
}

type VerifiedLanguage struct {
	Language   string `json:"language"`
	ModelID    string `json:"model_id"`
	Accent     string `json:"accent"`
	Locale     string `json:"locale"`
	PreviewURL string `json:"preview_url"`
}

type Recording struct {
	RecordingID    string `json:"recording_id"`
	MimeType       string `json:"mime_type"`
	SizeBytes      int    `json:"size_bytes"`
	UploadDateUnix int    `json:"upload_date_unix"`
	Transcription  string `json:"transcription"`
}

type VerificationAttempt struct {
	Text                string    `json:"text"`
	DateUnix            int       `json:"date_unix"`
	Accepted            bool      `json:"accepted"`
	Similarity          float64   `json:"similarity"`
	LevenshteinDistance int       `json:"levenshtein_distance"`
	Recording           Recording `json:"recording"`
}

type VoiceVerification struct {
	RequiresVerification      bool                  `json:"requires_verification"`
	IsVerified                bool                  `json:"is_verified"`
	VerificationFailures      []any                 `json:"verification_failures"`
	VerificationAttemptsCount int                   `json:"verification_attempts_count"`
	Language                  string                `json:"language"`
	VerificationAttempts      []VerificationAttempt `json:"verification_attempts"`
}

type Voice struct {
	VoiceID                 string             `json:"voice_id"`
	Name                    string             `json:"name"`
	Samples                 []Sample           `json:"samples"`
	Category                string             `json:"category"`
	FineTuning              FineTuning         `json:"fine_tuning"`
	Labels                  Labels             `json:"labels"`
	Description             string             `json:"description"`
	PreviewURL              string             `json:"preview_url"`
	AvailableForTiers       []string           `json:"available_for_tiers"`
	Settings                Settings           `json:"settings"`
	Sharing                 Sharing            `json:"sharing"`
	HighQualityBaseModelIds []string           `json:"high_quality_base_model_ids"`
	VerifiedLanguages       []VerifiedLanguage `json:"verified_languages"`
	CollectionIds           []string           `json:"collection_ids"`
	SafetyControl           string             `json:"safety_control"`
	VoiceVerification       VoiceVerification  `json:"voice_verification"`
	PermissionOnResource    string             `json:"permission_on_resource"`
	IsOwner                 bool               `json:"is_owner"`
	IsLegacy                bool               `json:"is_legacy"`
	IsMixed                 bool               `json:"is_mixed"`
	FavoritedAtUnix         int                `json:"favorited_at_unix"`
	CreatedAtUnix           int                `json:"created_at_unix"`
}

type VoiceListResponse struct {
	Voices        []Voice `json:"voices"`
	HasMore       bool    `json:"has_more"`
	TotalCount    int     `json:"total_count"`
	NextPageToken string  `json:"next_page_token"`
}
