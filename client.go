package twitter

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const absoluteMaxCount = 500

// RateLimitConfig controls request pacing and retry behavior.
type RateLimitConfig struct {
	RequestDelay   float64 // seconds between paginated requests (default 1.5)
	MaxRetries     int     // retry count on rate limit (default 3)
	RetryBaseDelay float64 // exponential backoff base in seconds (default 5.0)
	MaxCount       int     // hard cap on items per fetch (default 200)
}

// TwitterAPIError represents an HTTP/network error from Twitter APIs.
type TwitterAPIError struct {
	StatusCode int
	Message    string
}

func (e *TwitterAPIError) Error() string {
	return e.Message
}

// Client is the Twitter GraphQL API client using cookie authentication.
type Client struct {
	authToken      string
	ct0            string
	requestDelay   float64
	maxRetries     int
	retryBaseDelay float64
	maxCount       int
	httpClient     *http.Client

	mu             sync.Mutex
	cachedQueryIDs map[string]string
	bundlesScanned bool
}

// NewClient creates a new Twitter API client from a raw cookie string.
//
// cookies is the full cookie string (e.g. "auth_token=xxx; ct0=yyy; ...").
// The auth_token and ct0 values are automatically extracted.
// Pass nil for rlConfig to use defaults.
func NewClient(cookies string, rlConfig *RateLimitConfig) (*Client, error) {
	authToken, ct0 := parseCookies(cookies)
	if authToken == "" || ct0 == "" {
		return nil, fmt.Errorf("cookies must contain both auth_token and ct0")
	}
	c := &Client{
		authToken:      authToken,
		ct0:            ct0,
		requestDelay:   1.5,
		maxRetries:     3,
		retryBaseDelay: 5.0,
		maxCount:       200,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		cachedQueryIDs: make(map[string]string),
	}
	if rlConfig != nil {
		if rlConfig.RequestDelay > 0 {
			c.requestDelay = rlConfig.RequestDelay
		}
		if rlConfig.MaxRetries > 0 {
			c.maxRetries = rlConfig.MaxRetries
		}
		if rlConfig.RetryBaseDelay > 0 {
			c.retryBaseDelay = rlConfig.RetryBaseDelay
		}
		if rlConfig.MaxCount > 0 {
			c.maxCount = rlConfig.MaxCount
			if c.maxCount > absoluteMaxCount {
				c.maxCount = absoluteMaxCount
			}
		}
	}
	return c, nil
}

// parseCookies extracts auth_token and ct0 from a cookie string.
func parseCookies(cookies string) (authToken, ct0 string) {
	for _, part := range strings.Split(cookies, ";") {
		part = strings.TrimSpace(part)
		if k, v, ok := strings.Cut(part, "="); ok {
			switch strings.TrimSpace(k) {
			case "auth_token":
				authToken = strings.TrimSpace(v)
			case "ct0":
				ct0 = strings.TrimSpace(v)
			}
		}
	}
	return
}

// FetchHomeTimeline fetches the algorithmic home timeline.
func (c *Client) FetchHomeTimeline(count int) ([]Tweet, error) {
	return c.fetchTimeline("HomeTimeline", count, func(data map[string]any) any {
		return deepGet(data, "data", "home", "home_timeline_urt", "instructions")
	}, nil, false, nil)
}

// FetchFollowingFeed fetches the chronological following feed.
func (c *Client) FetchFollowingFeed(count int) ([]Tweet, error) {
	return c.fetchTimeline("HomeLatestTimeline", count, func(data map[string]any) any {
		return deepGet(data, "data", "home", "home_timeline_urt", "instructions")
	}, nil, false, nil)
}

// FetchBookmarks fetches bookmarked tweets.
func (c *Client) FetchBookmarks(count int) ([]Tweet, error) {
	return c.fetchTimeline("Bookmarks", count, func(data map[string]any) any {
		v := deepGet(data, "data", "bookmark_timeline", "timeline", "instructions")
		if v == nil {
			v = deepGet(data, "data", "bookmark_timeline_v2", "timeline", "instructions")
		}
		return v
	}, nil, false, nil)
}

// FetchUser fetches a user profile by screen name.
func (c *Client) FetchUser(screenName string) (*UserProfile, error) {
	variables := map[string]any{
		"screen_name":              screenName,
		"withSafetyModeUserFields": true,
	}
	features := map[string]any{
		"hidden_profile_subscriptions_enabled":                            true,
		"rweb_tipjar_consumption_enabled":                                 true,
		"responsive_web_graphql_exclude_directive_enabled":                true,
		"verified_phone_label_enabled":                                    false,
		"subscriptions_verification_info_is_identity_verified_enabled":    true,
		"subscriptions_verification_info_verified_since_enabled":          true,
		"highlights_tweets_tab_ui_enabled":                                true,
		"responsive_web_twitter_article_notes_tab_enabled":                true,
		"subscriptions_feature_can_gift_premium":                          true,
		"creator_subscriptions_tweet_preview_api_enabled":                 true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled": false,
		"responsive_web_graphql_timeline_navigation_enabled":              true,
	}
	data, err := c.graphqlGet("UserByScreenName", variables, features, nil)
	if err != nil {
		return nil, err
	}
	result := deepGetMap(data, "data", "user", "result")
	if result == nil {
		return nil, fmt.Errorf("user @%s not found", screenName)
	}
	legacy := getMap(result, "legacy")
	profile := &UserProfile{
		ID:              getString(result, "rest_id"),
		Name:            getString(legacy, "name"),
		ScreenName:      getStringOr(legacy, "screen_name", screenName),
		Bio:             getString(legacy, "description"),
		Location:        getString(legacy, "location"),
		URL:             deepGetString(legacy, "entities", "url", "urls", 0, "expanded_url"),
		FollowersCount:  getInt(legacy, "followers_count"),
		FollowingCount:  getInt(legacy, "friends_count"),
		TweetsCount:     getInt(legacy, "statuses_count"),
		LikesCount:      getInt(legacy, "favourites_count"),
		Verified:        getBool(result, "is_blue_verified") || getBool(legacy, "verified"),
		ProfileImageURL: getString(legacy, "profile_image_url_https"),
		CreatedAt:       getString(legacy, "created_at"),
	}
	return profile, nil
}

// FetchUserTweets fetches tweets posted by a user.
func (c *Client) FetchUserTweets(userID string, count int) ([]Tweet, error) {
	return c.fetchTimeline("UserTweets", count, func(data map[string]any) any {
		return deepGet(data, "data", "user", "result", "timeline_v2", "timeline", "instructions")
	}, map[string]any{
		"userId":                                 userID,
		"withQuickPromoteEligibilityTweetFields": true,
		"withVoice":                              true,
		"withV2Timeline":                         true,
	}, false, nil)
}

// FetchUserLikes fetches tweets liked by a user.
func (c *Client) FetchUserLikes(userID string, count int) ([]Tweet, error) {
	return c.fetchTimeline("Likes", count, func(data map[string]any) any {
		return deepGet(data, "data", "user", "result", "timeline_v2", "timeline", "instructions")
	}, map[string]any{
		"userId":                 userID,
		"includePromotedContent": false,
		"withClientEventToken":   false,
		"withBirdwatchNotes":     false,
		"withVoice":              true,
	}, true, nil)
}

// FetchSearch searches tweets by query.
// product can be "Top", "Latest", "People", "Photos", or "Videos".
func (c *Client) FetchSearch(query string, count int, product string) ([]Tweet, error) {
	if product == "" {
		product = "Top"
	}
	return c.fetchTimeline("SearchTimeline", count, func(data map[string]any) any {
		return deepGet(data, "data", "search_by_raw_query", "search_timeline", "timeline", "instructions")
	}, map[string]any{
		"rawQuery":    query,
		"querySource": "typed_query",
		"product":     product,
	}, true, nil)
}

// FetchTweetDetail fetches a tweet and its conversation thread.
func (c *Client) FetchTweetDetail(tweetID string, count int) ([]Tweet, error) {
	return c.fetchTimeline("TweetDetail", count, func(data map[string]any) any {
		v := deepGet(data, "data", "tweetResult", "result", "timeline", "instructions")
		if v == nil {
			v = deepGet(data, "data", "threaded_conversation_with_injections_v2", "instructions")
		}
		return v
	}, map[string]any{
		"focalTweetId":                           tweetID,
		"referrer":                               "tweet",
		"with_rux_injections":                    false,
		"includePromotedContent":                 true,
		"rankingMode":                            "Relevance",
		"withCommunity":                          true,
		"withQuickPromoteEligibilityTweetFields": true,
		"withBirdwatchNotes":                     true,
		"withVoice":                              true,
	}, true, map[string]any{
		"withArticleRichContentState":  true,
		"withArticlePlainText":         false,
		"withGrokAnalyze":              false,
		"withDisallowedReplyControls":  false,
	})
}

// FetchListTimeline fetches tweets from a Twitter List.
func (c *Client) FetchListTimeline(listID string, count int) ([]Tweet, error) {
	return c.fetchTimeline("ListLatestTweetsTimeline", count, func(data map[string]any) any {
		return deepGet(data, "data", "list", "tweets_timeline", "timeline", "instructions")
	}, map[string]any{
		"listId": listID,
	}, true, nil)
}

// FetchFollowers fetches followers of a user.
func (c *Client) FetchFollowers(userID string, count int) ([]UserProfile, error) {
	return c.fetchUserList("Followers", userID, count, func(data map[string]any) any {
		return deepGet(data, "data", "user", "result", "timeline", "timeline", "instructions")
	})
}

// FetchFollowing fetches users that a user is following.
func (c *Client) FetchFollowing(userID string, count int) ([]UserProfile, error) {
	return c.fetchUserList("Following", userID, count, func(data map[string]any) any {
		return deepGet(data, "data", "user", "result", "timeline", "timeline", "instructions")
	})
}

// CreateTweet posts a new tweet. Returns the new tweet ID.
func (c *Client) CreateTweet(text string) (string, error) {
	return c.CreateTweetReply(text, "")
}

// CreateTweetReply posts a tweet as a reply. Returns the new tweet ID.
func (c *Client) CreateTweetReply(text, replyToID string) (string, error) {
	variables := map[string]any{
		"tweet_text":              text,
		"media":                   map[string]any{"media_entities": []any{}, "possibly_sensitive": false},
		"semantic_annotation_ids": []any{},
		"dark_request":            false,
	}
	if replyToID != "" {
		variables["reply"] = map[string]any{
			"in_reply_to_tweet_id":   replyToID,
			"exclude_reply_user_ids": []any{},
		}
	}
	data, err := c.graphqlPost("CreateTweet", variables, boolMapToAnyMap(defaultFeatures))
	if err != nil {
		return "", err
	}
	result := deepGetMap(data, "data", "create_tweet", "tweet_results", "result")
	if result != nil {
		return getString(result, "rest_id"), nil
	}
	return "", fmt.Errorf("failed to create tweet")
}

// DeleteTweet deletes a tweet.
func (c *Client) DeleteTweet(tweetID string) error {
	_, err := c.graphqlPost("DeleteTweet", map[string]any{
		"tweet_id":     tweetID,
		"dark_request": false,
	}, nil)
	return err
}

// LikeTweet likes a tweet.
func (c *Client) LikeTweet(tweetID string) error {
	_, err := c.graphqlPost("FavoriteTweet", map[string]any{
		"tweet_id": tweetID,
	}, nil)
	return err
}

// UnlikeTweet unlikes a tweet.
func (c *Client) UnlikeTweet(tweetID string) error {
	_, err := c.graphqlPost("UnfavoriteTweet", map[string]any{
		"tweet_id":     tweetID,
		"dark_request": false,
	}, nil)
	return err
}

// Retweet retweets a tweet.
func (c *Client) Retweet(tweetID string) error {
	_, err := c.graphqlPost("CreateRetweet", map[string]any{
		"tweet_id":     tweetID,
		"dark_request": false,
	}, nil)
	return err
}

// Unretweet undoes a retweet.
func (c *Client) Unretweet(tweetID string) error {
	_, err := c.graphqlPost("DeleteRetweet", map[string]any{
		"source_tweet_id": tweetID,
		"dark_request":    false,
	}, nil)
	return err
}

// BookmarkTweet bookmarks a tweet.
func (c *Client) BookmarkTweet(tweetID string) error {
	_, err := c.graphqlPost("CreateBookmark", map[string]any{
		"tweet_id": tweetID,
	}, nil)
	return err
}

// UnbookmarkTweet removes a tweet from bookmarks.
func (c *Client) UnbookmarkTweet(tweetID string) error {
	_, err := c.graphqlPost("DeleteBookmark", map[string]any{
		"tweet_id": tweetID,
	}, nil)
	return err
}

// --- Internal methods ---

type instructionsExtractor func(data map[string]any) any

func (c *Client) fetchTimeline(operationName string, count int, getInstructions instructionsExtractor, extraVariables map[string]any, overrideBaseVariables bool, fieldToggles map[string]any) ([]Tweet, error) {
	if count <= 0 {
		return nil, nil
	}
	if count > c.maxCount {
		count = c.maxCount
	}

	var tweets []Tweet
	seenIDs := make(map[string]bool)
	var cursor string
	maxAttempts := int(math.Ceil(float64(count)/20.0)) + 2

	for attempt := 0; len(tweets) < count && attempt < maxAttempts; attempt++ {
		batchSize := count - len(tweets) + 5
		if batchSize > 40 {
			batchSize = 40
		}

		var variables map[string]any
		if overrideBaseVariables {
			variables = map[string]any{"count": batchSize}
		} else {
			variables = map[string]any{
				"count":                  batchSize,
				"includePromotedContent": false,
				"latestControlAvailable": true,
				"requestContext":         "launch",
			}
		}
		for k, v := range extraVariables {
			variables[k] = v
		}
		if cursor != "" {
			variables["cursor"] = cursor
		}

		data, err := c.graphqlGet(operationName, variables, boolMapToAnyMap(defaultFeatures), fieldToggles)
		if err != nil {
			return tweets, err
		}

		newTweets, nextCursor := c.parseTimelineResponse(data, getInstructions)
		for _, tweet := range newTweets {
			if tweet.ID != "" && !seenIDs[tweet.ID] {
				seenIDs[tweet.ID] = true
				tweets = append(tweets, tweet)
			}
		}

		if nextCursor == "" || len(newTweets) == 0 {
			break
		}
		cursor = nextCursor

		if len(tweets) < count && c.requestDelay > 0 {
			time.Sleep(time.Duration(c.requestDelay * float64(time.Second)))
		}
	}

	if len(tweets) > count {
		tweets = tweets[:count]
	}
	return tweets, nil
}

func (c *Client) fetchUserList(operationName, userID string, count int, getInstructions instructionsExtractor) ([]UserProfile, error) {
	if count <= 0 {
		return nil, nil
	}
	if count > c.maxCount {
		count = c.maxCount
	}

	var users []UserProfile
	seenIDs := make(map[string]bool)
	var cursor string
	maxAttempts := int(math.Ceil(float64(count)/20.0)) + 2

	for attempt := 0; len(users) < count && attempt < maxAttempts; attempt++ {
		batchSize := count - len(users) + 5
		if batchSize > 40 {
			batchSize = 40
		}

		variables := map[string]any{
			"userId":                 userID,
			"count":                  batchSize,
			"includePromotedContent": false,
		}
		if cursor != "" {
			variables["cursor"] = cursor
		}

		data, err := c.graphqlGet(operationName, variables, boolMapToAnyMap(defaultFeatures), nil)
		if err != nil {
			return users, err
		}

		instructions := getInstructions(data)
		instrList, ok := instructions.([]any)
		if !ok {
			break
		}

		var newUsers []UserProfile
		var nextCursor string

		for _, instr := range instrList {
			instrMap, ok := instr.(map[string]any)
			if !ok {
				continue
			}
			entries, _ := instrMap["entries"].([]any)
			for _, entry := range entries {
				entryMap, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				content, _ := entryMap["content"].(map[string]any)
				if content == nil {
					continue
				}
				entryType, _ := content["entryType"].(string)

				if entryType == "TimelineTimelineItem" {
					itemContent, _ := content["itemContent"].(map[string]any)
					if itemContent != nil {
						userResults := deepGetMap(itemContent, "user_results", "result")
						if userResults != nil {
							if user := parseUserResult(userResults); user != nil {
								newUsers = append(newUsers, *user)
							}
						}
					}
				} else if entryType == "TimelineTimelineCursor" {
					if ct, _ := content["cursorType"].(string); ct == "Bottom" {
						if v, _ := content["value"].(string); v != "" {
							nextCursor = v
						}
					}
				}
			}
		}

		for _, user := range newUsers {
			if user.ID != "" && !seenIDs[user.ID] {
				seenIDs[user.ID] = true
				users = append(users, user)
			}
		}

		if nextCursor == "" || len(newUsers) == 0 {
			break
		}
		cursor = nextCursor

		if len(users) < count && c.requestDelay > 0 {
			time.Sleep(time.Duration(c.requestDelay * float64(time.Second)))
		}
	}

	if len(users) > count {
		users = users[:count]
	}
	return users, nil
}

func (c *Client) graphqlGet(operationName string, variables, features, fieldToggles map[string]any) (map[string]any, error) {
	queryID := c.resolveQueryID(operationName, true)
	usingFallback := queryID == fallbackQueryIDs[operationName]
	u := buildGraphQLURL(queryID, operationName, variables, features, fieldToggles)

	data, err := c.apiGet(u)
	if err != nil {
		apiErr, ok := err.(*TwitterAPIError)
		if ok && apiErr.StatusCode == 404 && usingFallback {
			c.invalidateQueryID(operationName)
			refreshed := c.resolveQueryID(operationName, false)
			retryURL := buildGraphQLURL(refreshed, operationName, variables, features, fieldToggles)
			return c.apiGet(retryURL)
		}
		return nil, err
	}
	return data, nil
}

func (c *Client) graphqlPost(operationName string, variables, features map[string]any) (map[string]any, error) {
	queryID := c.resolveQueryID(operationName, true)
	usingFallback := queryID == fallbackQueryIDs[operationName]

	doPost := func(qid string) (map[string]any, error) {
		u := fmt.Sprintf("https://x.com/i/api/graphql/%s/%s", qid, operationName)
		body := map[string]any{"variables": variables, "queryId": qid}
		if features != nil {
			body["features"] = features
		}
		return c.apiRequest(u, "POST", body)
	}

	data, err := doPost(queryID)
	if err != nil {
		apiErr, ok := err.(*TwitterAPIError)
		if ok && apiErr.StatusCode == 404 && usingFallback {
			c.invalidateQueryID(operationName)
			refreshed := c.resolveQueryID(operationName, false)
			return doPost(refreshed)
		}
		return nil, err
	}
	return data, nil
}

func (c *Client) apiGet(u string) (map[string]any, error) {
	return c.apiRequest(u, "GET", nil)
}

func (c *Client) apiRequest(u, method string, body map[string]any) (map[string]any, error) {
	headers := c.buildHeaders(method)

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = strings.NewReader(string(b))
	}

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequest(method, u, bodyReader)
		if err != nil {
			return nil, &TwitterAPIError{StatusCode: 0, Message: fmt.Sprintf("failed to create request: %v", err)}
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, &TwitterAPIError{StatusCode: 0, Message: fmt.Sprintf("Twitter API network error: %v", err)}
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, &TwitterAPIError{StatusCode: 0, Message: fmt.Sprintf("failed to read response: %v", err)}
		}

		if resp.StatusCode == 429 && attempt < c.maxRetries {
			wait := c.retryBaseDelay * math.Pow(2, float64(attempt))
			time.Sleep(time.Duration(wait * float64(time.Second)))
			// Reset body reader for retry
			if body != nil {
				b, _ := json.Marshal(body)
				bodyReader = strings.NewReader(string(b))
			}
			continue
		}

		if resp.StatusCode >= 400 {
			msg := fmt.Sprintf("Twitter API error %d: %s", resp.StatusCode, truncate(string(respBody), 500))
			return nil, &TwitterAPIError{StatusCode: resp.StatusCode, Message: msg}
		}

		var parsed map[string]any
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return nil, &TwitterAPIError{StatusCode: 0, Message: "Twitter API returned invalid JSON"}
		}

		if errs, ok := parsed["errors"].([]any); ok && len(errs) > 0 {
			if errMap, ok := errs[0].(map[string]any); ok {
				errMsg, _ := errMap["message"].(string)
				errCode, _ := errMap["code"].(float64)
				if int(errCode) == 88 && attempt < c.maxRetries {
					wait := c.retryBaseDelay * math.Pow(2, float64(attempt))
					time.Sleep(time.Duration(wait * float64(time.Second)))
					if body != nil {
						b, _ := json.Marshal(body)
						bodyReader = strings.NewReader(string(b))
					}
					continue
				}
				// Write operation rate limits (retweet/like/bookmark limits)
				if int(errCode) == 348 || int(errCode) == 349 {
					return nil, &TwitterAPIError{StatusCode: 429, Message: fmt.Sprintf("Rate limited: %s (try again later, recommended wait: 15+ minutes)", errMsg)}
				}
				return nil, &TwitterAPIError{StatusCode: 0, Message: fmt.Sprintf("Twitter API returned errors: %s", errMsg)}
			}
		}

		// GraphQL write mutations may return errors nested in data
		if dataObj, ok := parsed["data"].(map[string]any); ok {
			for _, val := range dataObj {
				if valMap, ok := val.(map[string]any); ok {
					if innerErrs, ok := valMap["errors"].([]any); ok && len(innerErrs) > 0 {
						if innerErr, ok := innerErrs[0].(map[string]any); ok {
							innerMsg, _ := innerErr["message"].(string)
							return nil, &TwitterAPIError{StatusCode: 0, Message: fmt.Sprintf("Twitter API: %s", innerMsg)}
						}
					}
				}
			}
		}

		return parsed, nil
	}

	return nil, &TwitterAPIError{StatusCode: 429, Message: fmt.Sprintf("rate limited after %d retries", c.maxRetries)}
}

func (c *Client) buildHeaders(method string) map[string]string {
	headers := map[string]string{
		"Authorization":             "Bearer " + bearerToken,
		"Cookie":                    fmt.Sprintf("auth_token=%s; ct0=%s", c.authToken, c.ct0),
		"X-Csrf-Token":              c.ct0,
		"X-Twitter-Active-User":     "yes",
		"X-Twitter-Auth-Type":       "OAuth2Session",
		"X-Twitter-Client-Language": "en",
		"User-Agent":                userAgent,
		"Origin":                    "https://x.com",
		"Referer":                   "https://x.com",
		"Accept":                    "*/*",
		"Accept-Language":           "en-US,en;q=0.9",
		"sec-ch-ua":                 secChUA,
		"sec-ch-ua-mobile":          secChUAMobile,
		"sec-ch-ua-platform":        secChUAPlatform,
		"Sec-Fetch-Dest":            "empty",
		"Sec-Fetch-Mode":            "cors",
		"Sec-Fetch-Site":            "same-origin",
	}
	if method == "POST" {
		headers["Content-Type"] = "application/json"
	}
	return headers
}

// --- Query ID resolution ---

func (c *Client) resolveQueryID(operationName string, preferFallback bool) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.cachedQueryIDs[operationName]; ok {
		return cached
	}

	if preferFallback {
		if fb, ok := fallbackQueryIDs[operationName]; ok {
			c.cachedQueryIDs[operationName] = fb
			return fb
		}
	}

	if ghID := fetchFromGitHub(operationName); ghID != "" {
		c.cachedQueryIDs[operationName] = ghID
		return ghID
	}

	c.scanBundles()
	if cached, ok := c.cachedQueryIDs[operationName]; ok {
		return cached
	}

	if fb, ok := fallbackQueryIDs[operationName]; ok {
		c.cachedQueryIDs[operationName] = fb
		return fb
	}

	return ""
}

func (c *Client) invalidateQueryID(operationName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cachedQueryIDs, operationName)
}

func fetchFromGitHub(operationName string) string {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(twitterOpenAPIURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	op, ok := parsed[operationName].(map[string]any)
	if !ok {
		return ""
	}
	qid, _ := op["queryId"].(string)
	return qid
}

func (c *Client) scanBundles() {
	if c.bundlesScanned {
		return
	}
	c.bundlesScanned = true

	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("GET", "https://x.com", nil)
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	html := string(body)

	scriptPattern := regexp.MustCompile(`(?:src|href)=["'](https://abs\.twimg\.com/responsive-web/client-web[^"']+\.js)["']`)
	matches := scriptPattern.FindAllStringSubmatch(html, -1)

	opPattern := regexp.MustCompile(`queryId:\s*"([A-Za-z0-9_-]+)"[^}]{0,200}operationName:\s*"([^"]+)"`)

	for _, match := range matches {
		scriptURL := match[1]
		resp, err := client.Get(scriptURL)
		if err != nil {
			continue
		}
		bundle, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		for _, m := range opPattern.FindAllStringSubmatch(string(bundle), -1) {
			queryID, opName := m[1], m[2]
			if _, exists := c.cachedQueryIDs[opName]; !exists {
				c.cachedQueryIDs[opName] = queryID
			}
		}
	}
}

// --- Response parsing ---

func (c *Client) parseTimelineResponse(data map[string]any, getInstructions instructionsExtractor) ([]Tweet, string) {
	var tweets []Tweet
	var nextCursor string

	instructions := getInstructions(data)
	instrList, ok := instructions.([]any)
	if !ok {
		return tweets, nextCursor
	}

	for _, instr := range instrList {
		instrMap, ok := instr.(map[string]any)
		if !ok {
			continue
		}

		var entries []any
		if e, ok := instrMap["entries"].([]any); ok {
			entries = e
		} else if e, ok := instrMap["moduleItems"].([]any); ok {
			entries = e
		}

		for _, entry := range entries {
			entryMap, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			content, _ := entryMap["content"].(map[string]any)
			if content == nil {
				continue
			}

			if cur := extractCursor(content); cur != "" {
				nextCursor = cur
			}

			itemContent, _ := content["itemContent"].(map[string]any)
			if itemContent != nil {
				result := deepGetMap(itemContent, "tweet_results", "result")
				if result != nil {
					if tweet := c.parseTweetResult(result, 0); tweet != nil {
						tweets = append(tweets, *tweet)
					}
				}
			}

			if items, ok := content["items"].([]any); ok {
				for _, nestedItem := range items {
					nestedMap, ok := nestedItem.(map[string]any)
					if !ok {
						continue
					}
					result := deepGetMap(nestedMap, "item", "itemContent", "tweet_results", "result")
					if result != nil {
						if tweet := c.parseTweetResult(result, 0); tweet != nil {
							tweets = append(tweets, *tweet)
						}
					}
				}
			}
		}
	}

	return tweets, nextCursor
}

func (c *Client) parseTweetResult(result map[string]any, depth int) *Tweet {
	if depth > 2 {
		return nil
	}

	tweetData := result
	if typeName, _ := result["__typename"].(string); typeName == "TweetWithVisibilityResults" {
		if inner, ok := result["tweet"].(map[string]any); ok {
			tweetData = inner
		}
	}
	if typeName, _ := tweetData["__typename"].(string); typeName == "TweetTombstone" {
		return nil
	}

	legacy, ok := tweetData["legacy"].(map[string]any)
	if !ok {
		return nil
	}
	core, ok := tweetData["core"].(map[string]any)
	if !ok {
		return nil
	}

	user := deepGetMapOr(core, "user_results", "result")
	userLegacy := getMap(user, "legacy")
	userCore := getMap(user, "core")

	isRetweet := deepGet(legacy, "retweeted_status_result", "result") != nil
	actualData := tweetData
	actualLegacy := legacy
	actualUser := user
	actualUserLegacy := userLegacy

	if isRetweet {
		rtResult := deepGetMap(legacy, "retweeted_status_result", "result")
		if rtResult != nil {
			if tn, _ := rtResult["__typename"].(string); tn == "TweetWithVisibilityResults" {
				if inner, ok := rtResult["tweet"].(map[string]any); ok {
					rtResult = inner
				}
			}
			if rtLegacy, ok := rtResult["legacy"].(map[string]any); ok {
				if rtCore, ok := rtResult["core"].(map[string]any); ok {
					actualData = rtResult
					actualLegacy = rtLegacy
					actualUser = deepGetMapOr(rtCore, "user_results", "result")
					actualUserLegacy = getMap(actualUser, "legacy")
				}
			}
		}
	}

	media := extractMedia(actualLegacy)
	urls := extractURLs(actualLegacy)

	var quotedTweet *Tweet
	if quoted := deepGetMap(actualData, "quoted_status_result", "result"); quoted != nil {
		quotedTweet = c.parseTweetResult(quoted, depth+1)
	}

	author := extractAuthor(actualUser, actualUserLegacy)

	var retweetedBy string
	if isRetweet {
		retweetedBy = getStringFirst(userCore, "screen_name")
		if retweetedBy == "" {
			retweetedBy = getStringOr(userLegacy, "screen_name", "unknown")
		}
	}

	return &Tweet{
		ID:     getString(actualData, "rest_id"),
		Text:   getString(actualLegacy, "full_text"),
		Author: author,
		Metrics: Metrics{
			Likes:     getInt(actualLegacy, "favorite_count"),
			Retweets:  getInt(actualLegacy, "retweet_count"),
			Replies:   getInt(actualLegacy, "reply_count"),
			Quotes:    getInt(actualLegacy, "quote_count"),
			Views:     deepGetInt(actualData, "views", "count"),
			Bookmarks: getInt(actualLegacy, "bookmark_count"),
		},
		CreatedAt:   getString(actualLegacy, "created_at"),
		Media:       media,
		URLs:        urls,
		IsRetweet:   isRetweet,
		RetweetedBy: retweetedBy,
		QuotedTweet: quotedTweet,
		Lang:        getString(actualLegacy, "lang"),
	}
}

func parseUserResult(userData map[string]any) *UserProfile {
	if tn, _ := userData["__typename"].(string); tn == "UserUnavailable" {
		return nil
	}
	legacy := getMap(userData, "legacy")
	if legacy == nil {
		return nil
	}
	return &UserProfile{
		ID:              getString(userData, "rest_id"),
		Name:            getString(legacy, "name"),
		ScreenName:      getString(legacy, "screen_name"),
		Bio:             getString(legacy, "description"),
		Location:        getString(legacy, "location"),
		URL:             deepGetString(legacy, "entities", "url", "urls", 0, "expanded_url"),
		FollowersCount:  getInt(legacy, "followers_count"),
		FollowingCount:  getInt(legacy, "friends_count"),
		TweetsCount:     getInt(legacy, "statuses_count"),
		LikesCount:      getInt(legacy, "favourites_count"),
		Verified:        getBool(userData, "is_blue_verified") || getBool(legacy, "verified"),
		ProfileImageURL: getString(legacy, "profile_image_url_https"),
		CreatedAt:       getString(legacy, "created_at"),
	}
}

// --- Helpers ---

func buildGraphQLURL(queryID, operationName string, variables, features, fieldToggles map[string]any) string {
	varsJSON, _ := json.Marshal(variables)
	// Compact features: omit false values to keep URL under server limits.
	// Twitter's API defaults missing features to False.
	compactFeatures := make(map[string]any)
	for k, v := range features {
		if b, ok := v.(bool); ok && !b {
			continue
		}
		compactFeatures[k] = v
	}
	featJSON, _ := json.Marshal(compactFeatures)
	u := fmt.Sprintf("https://x.com/i/api/graphql/%s/%s?variables=%s&features=%s",
		queryID, operationName,
		url.QueryEscape(string(varsJSON)),
		url.QueryEscape(string(featJSON)),
	)
	if fieldToggles != nil {
		ftJSON, _ := json.Marshal(fieldToggles)
		u += "&fieldToggles=" + url.QueryEscape(string(ftJSON))
	}
	return u
}

func extractMedia(legacy map[string]any) []TweetMedia {
	var media []TweetMedia
	items, _ := deepGet(legacy, "extended_entities", "media").([]any)
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		mediaType, _ := m["type"].(string)

		switch mediaType {
		case "photo":
			media = append(media, TweetMedia{
				Type:   "photo",
				URL:    getString(m, "media_url_https"),
				Width:  deepGetInt(m, "original_info", "width"),
				Height: deepGetInt(m, "original_info", "height"),
			})
		case "video", "animated_gif":
			videoURL := getString(m, "media_url_https")
			if videoInfo, ok := m["video_info"].(map[string]any); ok {
				if variants, ok := videoInfo["variants"].([]any); ok {
					bestBitrate := -1
					for _, v := range variants {
						vm, ok := v.(map[string]any)
						if !ok {
							continue
						}
						ct, _ := vm["content_type"].(string)
						if ct != "video/mp4" {
							continue
						}
						br := 0
						if b, ok := vm["bitrate"].(float64); ok {
							br = int(b)
						}
						if br > bestBitrate {
							bestBitrate = br
							if u, ok := vm["url"].(string); ok {
								videoURL = u
							}
						}
					}
				}
			}
			media = append(media, TweetMedia{
				Type:   mediaType,
				URL:    videoURL,
				Width:  deepGetInt(m, "original_info", "width"),
				Height: deepGetInt(m, "original_info", "height"),
			})
		}
	}
	return media
}

func extractURLs(legacy map[string]any) []string {
	var urls []string
	items, _ := deepGet(legacy, "entities", "urls").([]any)
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			if u, ok := m["expanded_url"].(string); ok && u != "" {
				urls = append(urls, u)
			}
		}
	}
	return urls
}

func extractAuthor(userData, userLegacy map[string]any) Author {
	userCore := getMap(userData, "core")
	name := getStringFirst(userCore, "name")
	if name == "" {
		name = getStringFirst(userLegacy, "name")
	}
	if name == "" {
		name = getStringOr(userData, "name", "Unknown")
	}

	screenName := getStringFirst(userCore, "screen_name")
	if screenName == "" {
		screenName = getStringFirst(userLegacy, "screen_name")
	}
	if screenName == "" {
		screenName = getStringOr(userData, "screen_name", "unknown")
	}

	profileImageURL := ""
	if avatar, ok := userData["avatar"].(map[string]any); ok {
		profileImageURL, _ = avatar["image_url"].(string)
	}
	if profileImageURL == "" {
		profileImageURL = getString(userLegacy, "profile_image_url_https")
	}

	return Author{
		ID:              getString(userData, "rest_id"),
		Name:            name,
		ScreenName:      screenName,
		ProfileImageURL: profileImageURL,
		Verified:        getBool(userData, "is_blue_verified") || getBool(userLegacy, "verified"),
	}
}

func extractCursor(content map[string]any) string {
	if ct, _ := content["cursorType"].(string); ct == "Bottom" {
		if v, _ := content["value"].(string); v != "" {
			return v
		}
	}
	return ""
}

// deepGet navigates nested map[string]any / []any structures.
// Supports string keys for maps and int keys for slices.
func deepGet(data any, keys ...any) any {
	current := data
	for _, key := range keys {
		switch k := key.(type) {
		case string:
			m, ok := current.(map[string]any)
			if !ok {
				return nil
			}
			current = m[k]
		case int:
			s, ok := current.([]any)
			if !ok || k < 0 || k >= len(s) {
				return nil
			}
			current = s[k]
		default:
			return nil
		}
	}
	return current
}

func deepGetMap(data any, keys ...string) map[string]any {
	current := data
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}
	if m, ok := current.(map[string]any); ok {
		return m
	}
	return nil
}

func deepGetMapOr(data any, keys ...string) map[string]any {
	if m := deepGetMap(data, keys...); m != nil {
		return m
	}
	return map[string]any{}
}

func deepGetString(data any, keys ...any) string {
	v := deepGet(data, keys...)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func deepGetInt(data any, keys ...any) int {
	v := deepGet(data, keys...)
	return toInt(v)
}

func getMap(data map[string]any, key string) map[string]any {
	if data == nil {
		return nil
	}
	if v, ok := data[key].(map[string]any); ok {
		return v
	}
	return nil
}

func getString(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func getStringOr(data map[string]any, key, fallback string) string {
	if s := getString(data, key); s != "" {
		return s
	}
	return fallback
}

func getStringFirst(data map[string]any, key string) string {
	return getString(data, key)
}

func getBool(data map[string]any, key string) bool {
	if data == nil {
		return false
	}
	if v, ok := data[key].(bool); ok {
		return v
	}
	return false
}

func getInt(data map[string]any, key string) int {
	if data == nil {
		return 0
	}
	return toInt(data[key])
}

func toInt(v any) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case string:
		s := strings.ReplaceAll(strings.TrimSpace(val), ",", "")
		if s == "" {
			return 0
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int(f)
		}
		return 0
	default:
		return 0
	}
}

func boolMapToAnyMap(m map[string]bool) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
