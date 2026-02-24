package ports

// FeedItem represents a summary feed/note entry
type FeedItem struct {
	ID           string `json:"id"`
	XsecToken    string `json:"xsec_token"`
	Title        string `json:"title"`
	Type         string `json:"type"`
	PublishTime  string `json:"publish_time"`
	CoverURL     string `json:"cover_url"`
	LikedCount   string `json:"liked_count"`
	CommentCount string `json:"comment_count"`
	CollectCount string `json:"collect_count"`
	AuthorID     string `json:"author_id"`
	AuthorName   string `json:"author_name"`
}

// FeedDetail represents the full detail of a feed/note
type FeedDetail struct {
	NoteID      string      `json:"note_id"`
	XsecToken   string      `json:"xsec_token"`
	Title       string      `json:"title"`
	Desc        string      `json:"desc"`
	Type        string      `json:"type"`
	Time        int64       `json:"time"`
	IPLocation  string      `json:"ip_location"`
	AuthorID    string      `json:"author_id"`
	AuthorName  string      `json:"author_name"`
	Liked       bool        `json:"liked"`
	LikedCount  string      `json:"liked_count"`
	Collected   bool        `json:"collected"`
	CollectCount string     `json:"collect_count"`
	CommentCount string     `json:"comment_count"`
	Images      []string    `json:"images,omitempty"`
	Comments    []Comment   `json:"comments,omitempty"`
}

// Comment represents a single comment on a feed
type Comment struct {
	ID              string    `json:"id"`
	Content         string    `json:"content"`
	LikeCount       string    `json:"like_count"`
	CreateTime      int64     `json:"create_time"`
	Liked           bool      `json:"liked"`
	AuthorID        string    `json:"author_id"`
	AuthorName      string    `json:"author_name"`
	SubCommentCount string    `json:"sub_comment_count"`
	SubComments     []Comment `json:"sub_comments,omitempty"`
}

// UserProfile represents a user's public profile page data
type UserProfile struct {
	UserID       string `json:"user_id"`
	Nickname     string `json:"nickname"`
	RedID        string `json:"red_id"`
	Desc         string `json:"desc"`
	Gender       int    `json:"gender"`
	IPLocation   string `json:"ip_location"`
	Avatar       string `json:"avatar"`
	FollowCount  string `json:"follow_count"`
	FansCount    string `json:"fans_count"`
	LikedCount   string `json:"liked_count"`
}

// MyStats represents the current user's creator statistics
type MyStats struct {
	FollowerCount       int     `json:"follower_count"`
	FollowCount         int     `json:"follow_count"`
	LikedCount          int     `json:"liked_count"`
	NoteCount           int     `json:"note_count"`
	CollectCount        int     `json:"collect_count"`
	ExposureCount       int     `json:"exposure_count,omitempty"`
	ViewCount           int     `json:"view_count,omitempty"`
	CoverClickRate      float64 `json:"cover_click_rate,omitempty"`
	VideoCompleteRate   float64 `json:"video_complete_rate,omitempty"`
	LikeCount7d         int     `json:"like_count_7d,omitempty"`
	CommentCount7d      int     `json:"comment_count_7d,omitempty"`
	CollectCount7d      int     `json:"collect_count_7d,omitempty"`
	ShareCount7d        int     `json:"share_count_7d,omitempty"`
	NetFollowerGrowth   int     `json:"net_follower_growth,omitempty"`
	NewFollowerCount    int     `json:"new_follower_count,omitempty"`
	UnfollowCount       int     `json:"unfollow_count,omitempty"`
	ProfileVisitorCount int     `json:"profile_visitor_count,omitempty"`
}

// ContentAnalytics represents content performance analytics
type ContentAnalytics struct {
	Notes []NoteMetrics `json:"notes"`
}

// NoteMetrics represents per-note performance metrics
type NoteMetrics struct {
	FeedID          string  `json:"feed_id"`
	XsecToken       string  `json:"xsec_token"`
	Title           string  `json:"title"`
	PublishTime     string  `json:"publish_time"`
	Exposure        int     `json:"exposure"`
	Views           int     `json:"views"`
	ClickRate       float64 `json:"click_rate"`
	Likes           int     `json:"likes"`
	Comments        int     `json:"comments"`
	Collects        int     `json:"collects"`
	FollowerGrowth  int     `json:"follower_growth"`
	Shares          int     `json:"shares"`
	AvgViewDuration string  `json:"avg_view_duration"`
	FullScreen      int     `json:"full_screen"`
	Status          string  `json:"status"`
}

// FanAnalytics represents fan/follower analytics data
type FanAnalytics struct {
	Overview     FanOverview     `json:"overview"`
	Demographics FanDemographics `json:"demographics"`
	ActiveFans   []ActiveFan     `json:"active_fans"`
}

// FanOverview represents high-level fan counts
type FanOverview struct {
	TotalFans int `json:"total_fans"`
	NewFans   int `json:"new_fans"`
	LostFans  int `json:"lost_fans"`
}

// FanDemographics represents fan demographic breakdown
type FanDemographics struct {
	Gender    map[string]int `json:"gender"`
	Interests []string       `json:"interests"`
}

// ActiveFan represents an active fan with interaction count
type ActiveFan struct {
	Nickname     string `json:"nickname"`
	Interactions int    `json:"interactions"`
}

// LoginStatus represents the current login state
type LoginStatus struct {
	LoggedIn bool   `json:"logged_in"`
	UserID   string `json:"user_id,omitempty"`
	Nickname string `json:"nickname,omitempty"`
}

// QRCode represents a login QR code
type QRCode struct {
	ImageSrc  string `json:"image_src"`  // base64 data URL or URL
	LoggedIn  bool   `json:"logged_in"`  // true if already logged in
}
