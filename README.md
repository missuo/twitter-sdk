# twitter-sdk

A Go SDK for the Twitter/X GraphQL API using cookie-based authentication.

## Installation

```bash
go get github.com/missuo/twitter-sdk
```

## Authentication

This SDK uses cookie-based authentication. You need to extract `auth_token` and `ct0` from your browser cookies when logged into x.com.

### Getting Your Cookies

1. Open [x.com](https://x.com) in your browser and log in
2. Open DevTools (F12) > Application > Cookies > `https://x.com`
3. Copy the full cookie string, or at minimum the `auth_token` and `ct0` values

The SDK automatically parses the cookie string to extract the required fields.

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	twitter "github.com/missuo/twitter-sdk"
)

func main() {
	client, err := twitter.NewClient("auth_token=YOUR_AUTH_TOKEN; ct0=YOUR_CT0", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Fetch a user profile
	user, err := client.FetchUser("golang")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("@%s has %d followers\n", user.ScreenName, user.FollowersCount)

	// Fetch home timeline
	tweets, err := client.FetchHomeTimeline(10)
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range tweets {
		fmt.Printf("@%s: %s\n", t.Author.ScreenName, t.Text)
	}
}
```

You can also pass the full cookie string directly from the browser:

```go
cookies := "guest_id=v1%3A...; auth_token=abc123; ct0=def456; twid=u%3D12345; ..."
client, err := twitter.NewClient(cookies, nil)
```

## API Reference

### Creating a Client

```go
// With default rate limit settings
client, err := twitter.NewClient(cookies, nil)

// With custom rate limit settings
client, err := twitter.NewClient(cookies, &twitter.RateLimitConfig{
    RequestDelay:   1.0,   // seconds between paginated requests (default: 1.5)
    MaxRetries:     5,     // retry count on rate limit (default: 3)
    RetryBaseDelay: 10.0,  // exponential backoff base in seconds (default: 5.0)
    MaxCount:       300,   // hard cap on items per fetch (default: 200, max: 500)
})
```

### Read Operations

#### FetchUser

Fetch a user profile by screen name.

```go
user, err := client.FetchUser("elonmusk")
// user.ID, user.Name, user.ScreenName, user.Bio,
// user.FollowersCount, user.FollowingCount, user.TweetsCount, ...
```

#### FetchHomeTimeline

Fetch the algorithmic "For You" home timeline.

```go
tweets, err := client.FetchHomeTimeline(20)
```

#### FetchFollowingFeed

Fetch the chronological "Following" feed.

```go
tweets, err := client.FetchFollowingFeed(20)
```

#### FetchUserTweets

Fetch tweets posted by a user (requires user ID).

```go
user, _ := client.FetchUser("golang")
tweets, err := client.FetchUserTweets(user.ID, 20)
```

#### FetchUserLikes

Fetch tweets liked by a user (requires user ID).

```go
tweets, err := client.FetchUserLikes(user.ID, 20)
```

#### FetchSearch

Search tweets by query. Product can be `"Top"`, `"Latest"`, `"People"`, `"Photos"`, or `"Videos"`.

```go
tweets, err := client.FetchSearch("golang", 20, "Top")
```

#### FetchTweetDetail

Fetch a tweet and its conversation thread (replies).

```go
tweets, err := client.FetchTweetDetail("1234567890", 20)
// tweets[0] is the main tweet, followed by replies
```

#### FetchBookmarks

Fetch the authenticated user's bookmarked tweets.

```go
tweets, err := client.FetchBookmarks(50)
```

#### FetchListTimeline

Fetch tweets from a Twitter List.

```go
tweets, err := client.FetchListTimeline("LIST_ID", 20)
```

#### FetchFollowers

Fetch followers of a user (requires user ID).

```go
followers, err := client.FetchFollowers(user.ID, 20)
// returns []twitter.UserProfile
```

#### FetchFollowing

Fetch users that a user is following (requires user ID).

```go
following, err := client.FetchFollowing(user.ID, 20)
// returns []twitter.UserProfile
```

### Write Operations

#### CreateTweet

Post a new tweet. Returns the new tweet ID.

```go
tweetID, err := client.CreateTweet("Hello, world!")
```

#### CreateTweetReply

Post a reply to an existing tweet.

```go
tweetID, err := client.CreateTweetReply("Great post!", "TWEET_ID_TO_REPLY_TO")
```

#### DeleteTweet

Delete a tweet.

```go
err := client.DeleteTweet("TWEET_ID")
```

#### LikeTweet / UnlikeTweet

```go
err := client.LikeTweet("TWEET_ID")
err = client.UnlikeTweet("TWEET_ID")
```

#### Retweet / Unretweet

```go
err := client.Retweet("TWEET_ID")
err = client.Unretweet("TWEET_ID")
```

#### BookmarkTweet / UnbookmarkTweet

```go
err := client.BookmarkTweet("TWEET_ID")
err = client.UnbookmarkTweet("TWEET_ID")
```

## Data Models

### Tweet

```go
type Tweet struct {
    ID          string       // Tweet ID
    Text        string       // Full text content
    Author      Author       // Tweet author
    Metrics     Metrics      // Engagement metrics
    CreatedAt   string       // Creation timestamp
    Media       []TweetMedia // Attached media (photos, videos, GIFs)
    URLs        []string     // Expanded URLs in the tweet
    IsRetweet   bool         // Whether this is a retweet
    Lang        string       // Language code
    RetweetedBy string       // Screen name of retweeter (if retweet)
    QuotedTweet *Tweet       // Quoted tweet (if quote tweet)
    Score       float64      // Scoring value (for filtering)
}
```

### Author

```go
type Author struct {
    ID              string
    Name            string
    ScreenName      string
    ProfileImageURL string
    Verified        bool
}
```

### Metrics

```go
type Metrics struct {
    Likes     int
    Retweets  int
    Replies   int
    Quotes    int
    Views     int
    Bookmarks int
}
```

### TweetMedia

```go
type TweetMedia struct {
    Type   string // "photo", "video", "animated_gif"
    URL    string // Direct media URL (highest bitrate for videos)
    Width  int
    Height int
}
```

### UserProfile

```go
type UserProfile struct {
    ID              string
    Name            string
    ScreenName      string
    Bio             string
    Location        string
    URL             string
    FollowersCount  int
    FollowingCount  int
    TweetsCount     int
    LikesCount      int
    Verified        bool
    ProfileImageURL string
    CreatedAt       string
}
```

## Error Handling

API errors are returned as `*twitter.TwitterAPIError`, which includes the HTTP status code:

```go
tweets, err := client.FetchHomeTimeline(10)
if err != nil {
    if apiErr, ok := err.(*twitter.TwitterAPIError); ok {
        fmt.Printf("HTTP %d: %s\n", apiErr.StatusCode, apiErr.Message)
    }
}
```

The SDK automatically retries on HTTP 429 (rate limit) with exponential backoff.

## Query ID Resolution

Twitter's GraphQL API uses query IDs that change periodically. The SDK resolves them through a multi-level fallback:

1. In-memory cache
2. Hardcoded fallback IDs
3. Community-maintained [twitter-openapi](https://github.com/fa0311/twitter-openapi) on GitHub
4. Live scanning of Twitter's JS bundles

If a query ID returns 404, the SDK automatically invalidates it and retries with a freshly resolved ID.

## License

MIT
