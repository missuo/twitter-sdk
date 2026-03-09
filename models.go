package twitter

// Author represents a Twitter user as the author of a tweet.
type Author struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ScreenName      string `json:"screenName"`
	ProfileImageURL string `json:"profileImageUrl"`
	Verified        bool   `json:"verified"`
}

// Metrics contains engagement metrics for a tweet.
type Metrics struct {
	Likes     int `json:"likes"`
	Retweets  int `json:"retweets"`
	Replies   int `json:"replies"`
	Quotes    int `json:"quotes"`
	Views     int `json:"views"`
	Bookmarks int `json:"bookmarks"`
}

// TweetMedia represents a media attachment on a tweet.
type TweetMedia struct {
	Type   string `json:"type"` // "photo", "video", "animated_gif"
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// Tweet represents a single tweet.
type Tweet struct {
	ID          string       `json:"id"`
	Text        string       `json:"text"`
	Author      Author       `json:"author"`
	Metrics     Metrics      `json:"metrics"`
	CreatedAt   string       `json:"createdAt"`
	Media       []TweetMedia `json:"media"`
	URLs        []string     `json:"urls"`
	IsRetweet   bool         `json:"isRetweet"`
	Lang        string       `json:"lang"`
	RetweetedBy string       `json:"retweetedBy,omitempty"`
	QuotedTweet *Tweet       `json:"quotedTweet,omitempty"`
	Score       float64      `json:"score"`
}

// UserProfile represents a Twitter user's profile.
type UserProfile struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ScreenName      string `json:"screenName"`
	Bio             string `json:"bio"`
	Location        string `json:"location"`
	URL             string `json:"url"`
	FollowersCount  int    `json:"followersCount"`
	FollowingCount  int    `json:"followingCount"`
	TweetsCount     int    `json:"tweetsCount"`
	LikesCount      int    `json:"likesCount"`
	Verified        bool   `json:"verified"`
	ProfileImageURL string `json:"profileImageUrl"`
	CreatedAt       string `json:"createdAt"`
}
