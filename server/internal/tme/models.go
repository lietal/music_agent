package tme

type CommParams struct {
	Ct      int    `json:"ct"`
	Cv      int64  `json:"cv,omitempty"`
	V       int64  `json:"v,omitempty"`
	QQ      string `json:"qq,omitempty"`
	Authst  string `json:"authst,omitempty"`
	QIMEI36 string `json:"QIMEI36,omitempty"`
}

type MusicuSubRequest struct {
	Module string         `json:"module"`
	Method string         `json:"method"`
	Param  map[string]any `json:"param"`
}

type MusicuResponse struct {
	Code int64                      `json:"code"`
	Req  map[string]MusicuSubResponse `json:"-"`
}

type MusicuSubResponse struct {
	Code int64          `json:"code"`
	Data map[string]any `json:"data"`
}

type Song struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Artists         []string `json:"artists"`
	Album           string   `json:"album"`
	DurationSeconds int      `json:"duration_seconds"`
	ArtworkURL      string   `json:"artwork_url"`
}

type SongDetail struct {
	Song
	SourceURL string `json:"source_url,omitempty"`
}

type SongURL struct {
	SongID           string `json:"song_id"`
	URL              string `json:"url,omitempty"`
	ExpiresInSeconds int    `json:"expires_in_seconds,omitempty"`
}

type Lyrics struct {
	SongID     string `json:"song_id"`
	PlainText  string `json:"plain_text"`
	SyncedText string `json:"synced_text"`
}

type Comment struct {
	ID         string `json:"id"`
	SongID     string `json:"song_id"`
	AuthorName string `json:"author_name"`
	Text       string `json:"text"`
	LikedCount int    `json:"liked_count,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
}

type Playlist struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SongCount  int    `json:"song_count,omitempty"`
	ArtworkURL string `json:"artwork_url,omitempty"`
}

type Album struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Artists     []string `json:"artists"`
	ArtworkURL  string   `json:"artwork_url,omitempty"`
	ReleaseDate string   `json:"release_date,omitempty"`
	SongCount   int      `json:"song_count,omitempty"`
	Description string   `json:"description,omitempty"`
}

type Artist struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	AvatarURL   string   `json:"avatar_url,omitempty"`
	Alias       []string `json:"alias,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	Description string   `json:"description,omitempty"`
	SongCount   int      `json:"song_count,omitempty"`
	AlbumCount  int      `json:"album_count,omitempty"`
}

type Chart struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	UpdateFrequency string `json:"update_frequency,omitempty"`
	ArtworkURL      string `json:"artwork_url,omitempty"`
}

type UserProfile struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

type UserConnectionStatus struct {
	State            string `json:"state"`
	CredentialStored bool   `json:"credential_stored"`
	Authenticated    bool   `json:"authenticated"`
	UserID           string `json:"user_id,omitempty"`
	DisplayName      string `json:"display_name,omitempty"`
	Message          string `json:"message"`
}
