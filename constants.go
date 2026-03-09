package twitter

const bearerToken = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"

const secChUA = `"Chromium";v="133", "Not(A:Brand";v="99", "Google Chrome";v="133"`
const secChUAMobile = "?0"
const secChUAPlatform = `"macOS"`

const twitterOpenAPIURL = "https://raw.githubusercontent.com/fa0311/twitter-openapi/main/src/config/placeholder.json"

var fallbackQueryIDs = map[string]string{
	// Read operations
	"HomeTimeline":             "c-CzHF1LboFilMpsx4ZCrQ",
	"HomeLatestTimeline":       "BKB7oi212Fi7kQtCBGE4zA",
	"Bookmarks":                "VFdMm9iVZxlU6hD86gfW_A",
	"UserByScreenName":         "1VOOyvKkiI3FMmkeDNxM9A",
	"UserTweets":               "E3opETHurmVJflFsUBVuUQ",
	"SearchTimeline":           "nWemVnGJ6A5eQAR5-oQeAg",
	"Likes":                    "lIDpu_NWL7_VhimGGt0o6A",
	"TweetDetail":              "xd_EMdYvB9hfZsZ6Idri0w",
	"ListLatestTweetsTimeline": "RlZzktZY_9wJynoepm8ZsA",
	"Followers":                "IOh4aS6UdGWGJUYTqliQ7Q",
	"Following":                "zx6e-TLzRkeDO_a7p4b3JQ",
	// Write operations
	"CreateTweet":    "IID9x6WsdMnTlXnzXGq8ng",
	"DeleteTweet":    "VaenaVgh5q5ih7kvyVjgtg",
	"FavoriteTweet":  "lI07N6Otwv1PhnEgXILM7A",
	"UnfavoriteTweet": "ZYKSe-w7KEslx3JhSIk5LA",
	"CreateRetweet":  "ojPdsZsimiJrUGLR1sjUtA",
	"DeleteRetweet":  "iQtK4dl5hBmXewYZuEOKVw",
	"CreateBookmark": "aoDbu3RHznuiSkQ9aNM67Q",
	"DeleteBookmark": "Wlmlj2-xzyS1GN3a6cj-mQ",
}

// Essential features only — keep this list SMALL to avoid 414/431 URI Too Long.
// Twitter's API defaults missing features to False, so we only need True-valued ones
// that affect tweet data we actually consume. Each additional key adds ~60 chars to URL.
var defaultFeatures = map[string]bool{
	"creator_subscriptions_tweet_preview_api_enabled":                        true,
	"communities_web_enable_tweet_community_results_fetch":                   true,
	"c9s_tweet_anatomy_moderator_badge_enabled":                              true,
	"articles_preview_enabled":                                               true,
	"responsive_web_edit_tweet_api_enabled":                                  true,
	"graphql_is_translatable_rweb_tweet_is_translatable_enabled":             true,
	"view_counts_everywhere_api_enabled":                                     true,
	"longform_notetweets_consumption_enabled":                                true,
	"responsive_web_twitter_article_tweet_consumption_enabled":               true,
	"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
	"longform_notetweets_rich_text_read_enabled":                             true,
	"freedom_of_speech_not_reach_fetch_enabled":                              true,
	"standardized_nudges_misinfo":                                            true,
	"responsive_web_graphql_timeline_navigation_enabled":                     true,
	"responsive_web_enhance_cards_enabled":                                   false,
}
