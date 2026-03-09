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
	"SearchTimeline":           "qUm8YPFHJWjQ56E_dP4MDg",
	"Likes":                    "lIDpu_NWL7_VhimGGt0o6A",
	"TweetDetail":              "xd_EMdYvB9hfZsZ6Idri0w",
	"ListLatestTweetsTimeline": "RlZzktZY_9wJynoepm8ZsA",
	"Followers":                "a9TW9uKgJm6II3n6qKMrAA",
	"Following":                "zx6e-TLzRkeDO_a7p4b3JQ",
	// Write operations
	"CreateTweet":    "IID9x6WsdMnTlXnzXGq8ng",
	"DeleteTweet":    "VaenaVgh5q5ih7kvyVjgtg",
	"FavoriteTweet":  "lI07N6Otwv1PhnEgXILM7A",
	"UnfavoriteTweet": "ZYKSe-w7KEslx3JhSIk5LA",
	"CreateRetweet":  "mbRO74GrOvSfRcJnlMapnQ",
	"DeleteRetweet":  "iQtK4dl5hBmXewYZuEOKVw",
	"CreateBookmark": "aoDbu3RHznuiSkQ9aNM67Q",
	"DeleteBookmark": "Wlmlj2-xzyS1GN3a6cj-mQ",
}

var defaultFeatures = map[string]bool{
	"rweb_video_screen_enabled":                                              false,
	"profile_label_improvements_pcf_label_in_post_enabled":                   true,
	"responsive_web_profile_redirect_enabled":                                false,
	"rweb_tipjar_consumption_enabled":                                        false,
	"verified_phone_label_enabled":                                           false,
	"creator_subscriptions_tweet_preview_api_enabled":                        true,
	"responsive_web_graphql_timeline_navigation_enabled":                     true,
	"responsive_web_graphql_skip_user_profile_image_extensions_enabled":      false,
	"premium_content_api_read_enabled":                                       false,
	"communities_web_enable_tweet_community_results_fetch":                   true,
	"c9s_tweet_anatomy_moderator_badge_enabled":                              true,
	"responsive_web_grok_analyze_button_fetch_trends_enabled":                false,
	"responsive_web_grok_analyze_post_followups_enabled":                     true,
	"responsive_web_jetfuel_frame":                                           true,
	"responsive_web_grok_share_attachment_enabled":                           true,
	"responsive_web_grok_annotations_enabled":                                true,
	"articles_preview_enabled":                                               true,
	"responsive_web_edit_tweet_api_enabled":                                  true,
	"graphql_is_translatable_rweb_tweet_is_translatable_enabled":             true,
	"view_counts_everywhere_api_enabled":                                     true,
	"longform_notetweets_consumption_enabled":                                true,
	"responsive_web_twitter_article_tweet_consumption_enabled":               true,
	"tweet_awards_web_tipping_enabled":                                       false,
	"content_disclosure_indicator_enabled":                                   true,
	"content_disclosure_ai_generated_indicator_enabled":                      true,
	"responsive_web_grok_show_grok_translated_post":                         true,
	"responsive_web_grok_analysis_button_from_backend":                      true,
	"post_ctas_fetch_enabled":                                               true,
	"freedom_of_speech_not_reach_fetch_enabled":                              true,
	"standardized_nudges_misinfo":                                            true,
	"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
	"longform_notetweets_rich_text_read_enabled":                             true,
	"longform_notetweets_inline_media_enabled":                               false,
	"responsive_web_grok_image_annotation_enabled":                           true,
	"responsive_web_grok_imagine_annotation_enabled":                         true,
	"responsive_web_grok_community_note_auto_translation_is_enabled":         false,
	"responsive_web_enhance_cards_enabled":                                   false,
}
