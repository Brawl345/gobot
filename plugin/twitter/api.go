package twitter

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

const (
	bearerToken = "Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"

	apiBase          = "https://api.x.com"
	activateUrl      = apiBase + "/1.1/guest/activate.json"
	tweetDetailsPath = "/i/api/graphql/URPP6YZ5eDCjdVMSREn4gg/TweetResultByRestId"

	tweetVariables = `{"tweetId":"%s","withCommunity":false,"includePromotedContent":false,"withVoice":false}`
	tweetFeatures  = `{"creator_subscriptions_tweet_preview_api_enabled":true,"premium_content_api_read_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"responsive_web_grok_analyze_button_fetch_trends_enabled":false,"responsive_web_grok_analyze_post_followups_enabled":false,"responsive_web_jetfuel_frame":true,"responsive_web_grok_share_attachment_enabled":true,"articles_preview_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"responsive_web_grok_show_grok_translated_post":false,"responsive_web_grok_analysis_button_from_backend":false,"creator_subscriptions_quote_tweet_preview_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"payments_enabled":false,"profile_label_improvements_pcf_label_in_post_enabled":true,"rweb_tipjar_consumption_enabled":true,"verified_phone_label_enabled":false,"responsive_web_grok_image_annotation_enabled":true,"responsive_web_grok_imagine_annotation_enabled":true,"responsive_web_grok_community_note_auto_translation_is_enabled":false,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_enhance_cards_enabled":false}`
	fieldToggles   = `{"withArticleRichContentState":true,"withArticlePlainText":false,"withGrokAnalyze":false,"withDisallowedReplyControls":false}`
)

type (
	TokenResponse struct {
		GuestToken string `json:"guest_token"`
	}

	Option struct {
		Position int
		Label    string
		Votes    int
	}

	Poll struct {
		EndDatetime time.Time
		Options     []Option
		TotalVotes  int
	}

	Medium struct {
		DisplayUrl           string `json:"display_url"`
		ExpandedUrl          string `json:"expanded_url"`
		IdStr                string `json:"id_str"`
		Indices              []int  `json:"indices"`
		MediaKey             string `json:"media_key"`
		MediaUrlHttps        string `json:"media_url_https"`
		Type                 string `json:"type"`
		Url                  string `json:"url"`
		ExtMediaAvailability struct {
			Status string `json:"status"`
		} `json:"ext_media_availability"`
		MediaStats struct {
			ViewCount int `json:"viewCount"`
		} `json:"mediaStats"`
		VideoInfo struct {
			Variants []struct {
				Url     string `json:"url"`
				Bitrate int    `json:"bitrate"`
			} `json:"variants"`
		} `json:"video_info"`
	}

	Url struct {
		DisplayUrl  string `json:"display_url"`
		ExpandedUrl string `json:"expanded_url"`
		Url         string `json:"url"`
		Indices     []int  `json:"indices"`
	}

	NoteTweet struct {
		IsExpandable     bool `json:"is_expandable"`
		NoteTweetResults struct {
			Result struct {
				Id        string `json:"id"`
				Text      string `json:"text"`
				EntitySet struct {
					UserMentions []interface{} `json:"user_mentions"`
					Urls         []struct {
						DisplayUrl  string `json:"display_url"`
						ExpandedUrl string `json:"expanded_url"`
						Url         string `json:"url"`
						Indices     []int  `json:"indices"`
					} `json:"urls"`
					Hashtags []interface{} `json:"hashtags"`
					Symbols  []interface{} `json:"symbols"`
				} `json:"entity_set"`
			} `json:"result"`
		} `json:"note_tweet_results"`
	}

	Legacy struct {
		BookmarkCount     int    `json:"bookmark_count"`
		CreatedAt         string `json:"created_at"`
		ConversationIdStr string `json:"conversation_id_str"`
		DisplayTextRange  []int  `json:"display_text_range"`
		Entities          struct {
			Media []struct {
				DisplayUrl    string `json:"display_url"`
				ExpandedUrl   string `json:"expanded_url"`
				IdStr         string `json:"id_str"`
				Indices       []int  `json:"indices"`
				MediaUrlHttps string `json:"media_url_https"`
				Type          string `json:"type"`
				Url           string `json:"url"`
				Sizes         struct {
					Large struct {
						H      int    `json:"h"`
						W      int    `json:"w"`
						Resize string `json:"resize"`
					} `json:"large"`
					Medium struct {
						H      int    `json:"h"`
						W      int    `json:"w"`
						Resize string `json:"resize"`
					} `json:"medium"`
					Small struct {
						H      int    `json:"h"`
						W      int    `json:"w"`
						Resize string `json:"resize"`
					} `json:"small"`
					Thumb struct {
						H      int    `json:"h"`
						W      int    `json:"w"`
						Resize string `json:"resize"`
					} `json:"thumb"`
				} `json:"sizes"`
				OriginalInfo struct {
					Height     int `json:"height"`
					Width      int `json:"width"`
					FocusRects []struct {
						X int `json:"x"`
						Y int `json:"y"`
						W int `json:"w"`
						H int `json:"h"`
					} `json:"focus_rects"`
				} `json:"original_info"`
			} `json:"media"`
			UserMentions []interface{} `json:"user_mentions"`
			Urls         []Url         `json:"urls"`
		} `json:"entities"`
		ExtendedEntities struct {
			Media []Medium `json:"media"`
		} `json:"extended_entities"`
		FavoriteCount             int    `json:"favorite_count"`
		Favorited                 bool   `json:"favorited"`
		FullText                  string `json:"full_text"`
		IsQuoteStatus             bool   `json:"is_quote_status"`
		Lang                      string `json:"lang"`
		PossiblySensitive         bool   `json:"possibly_sensitive"`
		PossiblySensitiveEditable bool   `json:"possibly_sensitive_editable"`
		QuoteCount                int    `json:"quote_count"`
		ReplyCount                int    `json:"reply_count"`
		RetweetCount              int    `json:"retweet_count"`
		Retweeted                 bool   `json:"retweeted"`
		UserIdStr                 string `json:"user_id_str"`
		IdStr                     string `json:"id_str"`
	}

	UserResult struct {
		Result struct {
			Typename                   string `json:"__typename"`
			Id                         string `json:"id"`
			RestId                     string `json:"rest_id"`
			AffiliatesHighlightedLabel struct {
			} `json:"affiliates_highlighted_label"`
			Avatar struct {
				ImageUrl string `json:"image_url"`
			} `json:"avatar"`
			Core struct {
				CreatedAt  string `json:"created_at"`
				Name       string `json:"name"`
				ScreenName string `json:"screen_name"`
			} `json:"core"`
			HasNftAvatar   bool `json:"has_nft_avatar"`
			IsBlueVerified bool `json:"is_blue_verified"`
			Legacy         struct {
				DefaultProfile      bool   `json:"default_profile"`
				DefaultProfileImage bool   `json:"default_profile_image"`
				Description         string `json:"description"`
				Entities            struct {
					Description struct {
						Urls []struct {
							DisplayUrl  string `json:"display_url"`
							ExpandedUrl string `json:"expanded_url"`
							Url         string `json:"url"`
							Indices     []int  `json:"indices"`
						} `json:"urls"`
					} `json:"description"`
					Url struct {
						Urls []struct {
							DisplayUrl  string `json:"display_url"`
							ExpandedUrl string `json:"expanded_url"`
							Url         string `json:"url"`
							Indices     []int  `json:"indices"`
						} `json:"urls"`
					} `json:"url"`
				} `json:"entities"`
				FastFollowersCount      int           `json:"fast_followers_count"`
				FavouritesCount         int           `json:"favourites_count"`
				FollowersCount          int           `json:"followers_count"`
				FriendsCount            int           `json:"friends_count"`
				HasCustomTimelines      bool          `json:"has_custom_timelines"`
				IsTranslator            bool          `json:"is_translator"`
				ListedCount             int           `json:"listed_count"`
				MediaCount              int           `json:"media_count"`
				NormalFollowersCount    int           `json:"normal_followers_count"`
				PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
				PossiblySensitive       bool          `json:"possibly_sensitive"`
				ProfileBannerUrl        string        `json:"profile_banner_url"`
				ProfileInterstitialType string        `json:"profile_interstitial_type"`
				Protected               bool          `json:"protected"`
				StatusesCount           int           `json:"statuses_count"`
				TranslatorType          string        `json:"translator_type"`
				Url                     string        `json:"url"`
				VerifiedType            string        `json:"verified_type"`
				WithheldInCountries     []interface{} `json:"withheld_in_countries"`
			} `json:"legacy"`
			Location struct {
				Location string `json:"location"`
			} `json:"location"`
			Verification struct {
				Verified bool `json:"verified"`
			} `json:"verification"`
		} `json:"result"`
	}

	TweetInfo struct {
		RestId         string         `json:"rest_id"`
		BirdwatchPivot BirdwatchPivot `json:"birdwatch_pivot"`
		Core           struct {
			UserResults UserResult `json:"user_results"`
		} `json:"core"`
		UnmentionInfo struct {
		} `json:"unmention_info"`
		Source string `json:"source"`

		NoteTweet          NoteTweet `json:"note_tweet"`
		Legacy             Legacy    `json:"legacy"`
		QuotedStatusResult struct {
			Result struct {
				Tweet
				Reason   string `json:"reason"`
				Typename string `json:"__typename"`

				TweetSub Tweet `json:"tweet"`
			} `json:"result"`
		} `json:"quoted_status_result"`
	}

	Tweet struct {
		RestId         string         `json:"rest_id"`
		BirdwatchPivot BirdwatchPivot `json:"birdwatch_pivot"`
		Core           struct {
			UserResults UserResult `json:"user_results"`
		} `json:"core"`
		NoteTweet NoteTweet `json:"note_tweet"`
		Source    string    `json:"source"`
		Legacy    Legacy    `json:"legacy"`
		Card      Card      `json:"card"`
	}

	Result struct {
		TweetInfo           // TODO: On Withheld, TweetInfo is under "Tweet"
		Tweet     TweetInfo `json:"tweet"`
		Typename  string    `json:"__typename"`
		Reason    string    `json:"reason"`
		Card      Card      `json:"card"`
		Tombstone Tombstone `json:"tombstone"`
	}

	BirdwatchPivot struct {
		DestinationUrl string `json:"destinationUrl"`
		Footer         struct {
			Text     string `json:"text"`
			Entities []struct {
				FromIndex int `json:"fromIndex"`
				ToIndex   int `json:"toIndex"`
				Ref       struct {
					Type    string `json:"type"`
					Url     string `json:"url"`
					UrlType string `json:"urlType"`
				} `json:"ref"`
			} `json:"entities"`
		} `json:"footer"`
		Note struct {
			RestId string `json:"rest_id"`
		} `json:"note"`
		Subtitle struct {
			Text     string `json:"text"`
			Entities []struct {
				FromIndex int `json:"fromIndex"`
				ToIndex   int `json:"toIndex"`
				Ref       struct {
					Type    string `json:"type"`
					Url     string `json:"url"`
					UrlType string `json:"urlType"`
				} `json:"ref"`
			} `json:"entities"`
		} `json:"subtitle"`
		Title    string `json:"title"`
		IconType string `json:"iconType"`
	}

	Tombstone struct {
		Typename string `json:"__typename"`
		Text     struct {
			Text string `json:"text"`
		} `json:"text"`
	}

	Card struct {
		RestId string `json:"rest_id"`
		Legacy struct {
			BindingValues []struct {
				Key   string `json:"key"`
				Value struct {
					StringValue  string `json:"string_value,omitempty"`
					Type         string `json:"type"`
					BooleanValue bool   `json:"boolean_value,omitempty"`
					ScribeKey    string `json:"scribe_key,omitempty"`
				} `json:"value"`
			} `json:"binding_values"`
			CardPlatform struct {
				Platform struct {
					Audience struct {
						Name string `json:"name"`
					} `json:"audience"`
					Device struct {
						Name    string `json:"name"`
						Version string `json:"version"`
					} `json:"device"`
				} `json:"platform"`
			} `json:"card_platform"`
			Name            string        `json:"name"`
			Url             string        `json:"url"`
			UserRefsResults []interface{} `json:"user_refs_results"`
		} `json:"legacy"`
	}

	TweetResponse struct {
		Data struct {
			TweetResult struct {
				Result Result `json:"result"`
			} `json:"tweetResult"`
		} `json:"data"`
	}
)

func (u *UserResult) Author() string {
	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"<b>%s</b> (<a href=\"https://x.com/%s\">@%s</a>",
			utils.Escape(u.Result.Core.Name),
			u.Result.Core.ScreenName,
			u.Result.Core.ScreenName,
		),
	)

	if u.Result.Legacy.VerifiedType == "Government" {
		sb.WriteString(" ✅🏛")
	}
	if u.Result.Legacy.VerifiedType == "Business" {
		sb.WriteString(" ✅🏢")
	}

	if u.Result.Legacy.Protected {
		sb.WriteString(" 🔒")
	}

	sb.WriteString("):")

	return sb.String()
}

func (m *Medium) IsPhoto() bool {
	return m.Type == "photo"
}

func (m *Medium) IsVideo() bool {
	return m.Type == "video"
}

func (m *Medium) IsGIF() bool {
	// Well, not technically a GIF, but a video without sound
	return m.Type == "animated_gif"
}

func (m *Medium) Link() string {
	if m.IsPhoto() {
		return m.MediaUrlHttps + ":orig"
	}

	var highestRes int
	var highestResURL string

	for _, variant := range m.VideoInfo.Variants {
		if variant.Bitrate >= highestRes {
			highestRes = variant.Bitrate
			highestResURL = variant.Url
		}
	}

	return highestResURL
}

func (m *Medium) InputFile() gotgbot.InputFileOrString {
	return gotgbot.InputFileByURL(m.Link())
}

func (m *Medium) Caption() string {
	var caption string
	if m.IsVideo() {
		caption = m.Link()
		if m.MediaStats.ViewCount > 0 {
			plural := ""
			if m.MediaStats.ViewCount != 1 {
				plural = "e"
			}
			caption = fmt.Sprintf(
				"%s (%s Aufruf%s)",
				m.Link(),
				utils.FormatThousand(m.MediaStats.ViewCount),
				plural,
			)
		}
	} else {
		caption = m.Link()
	}

	return caption
}

func (l *Legacy) Metrics() string {
	var sb strings.Builder

	if l.RetweetCount > 0 {
		sb.WriteString(
			fmt.Sprintf(
				" | 🔁 %s",
				utils.FormatThousand(l.RetweetCount),
			),
		)
	}

	if l.QuoteCount > 0 {
		sb.WriteString(
			fmt.Sprintf(
				" | 💬 %s",
				utils.FormatThousand(l.QuoteCount),
			),
		)
	}

	if l.FavoriteCount > 0 {
		sb.WriteString(
			fmt.Sprintf(
				" | ❤ %s",
				utils.FormatThousand(l.FavoriteCount),
			),
		)
	}

	return sb.String()
}

func (c *Card) HasPoll() bool {
	return strings.Contains(c.Legacy.Name, "poll")
}

func (c *Card) Poll() (Poll, error) {
	var poll Poll

	if !c.HasPoll() {
		return poll, nil
	}

	choiceCount, err := strconv.Atoi(c.Legacy.Name[4:5])
	if err != nil {
		return poll, err
	}

	options := make([]Option, 0, choiceCount)
	for i := 1; i <= choiceCount; i++ {
		var option Option
		option.Position = i

		for _, bindingValue := range c.Legacy.BindingValues {
			if bindingValue.Key == fmt.Sprintf("choice%d_label", i) {
				option.Label = bindingValue.Value.StringValue
			}
			if bindingValue.Key == fmt.Sprintf("choice%d_count", i) {
				option.Votes, err = strconv.Atoi(bindingValue.Value.StringValue)
				if err != nil {
					return poll, err
				}
				poll.TotalVotes += option.Votes
			}
		}

		options = append(options, option)
	}

	poll.Options = options

	for _, bindingValue := range c.Legacy.BindingValues {
		if bindingValue.Key == "end_datetime_utc" {
			poll.EndDatetime, err = time.Parse("2006-01-02T15:04:05Z", bindingValue.Value.StringValue)
			if err != nil {
				return poll, err
			}
			break
		}
	}

	return poll, nil
}

func (p *Poll) Closed() bool {
	return p.EndDatetime.Before(time.Now().UTC())
}
