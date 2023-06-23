package twitter

import (
	"fmt"
	"github.com/Brawl345/gobot/utils"
	"strconv"
	"strings"
	"time"
)

const (
	bearerToken = "Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"

	apiBase          = "https://api.twitter.com"
	activateUrl      = apiBase + "/1.1/guest/activate.json"
	tweetDetailsPath = "/graphql/wTXkouwCKcMNQtY-NcDgAA/TweetDetail"

	tweetVariables = `{"focalTweetId":"%s","with_rux_injections":false,"includePromotedContent":true,"withCommunity":true,"withQuickPromoteEligibilityTweetFields":true,"withBirdwatchNotes":true,"withDownvotePerspective":false,"withReactionsMetadata":false,"withReactionsPerspective":false,"withVoice":true,"withV2Timeline":true}`
	tweetFeatures  = `{"responsive_web_twitter_blue_verified_badge_is_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"tweetypie_unmention_optimization_enabled":true,"vibe_api_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":false,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":false,"interactive_text_enabled":true,"responsive_web_text_conversations_enabled":false,"longform_notetweets_richtext_consumption_enabled":false,"responsive_web_enhance_cards_enabled":false}`
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
			HasNftAvatar bool `json:"has_nft_avatar"`
			Legacy       struct {
				CreatedAt           string `json:"created_at"`
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
				Location                string        `json:"location"`
				MediaCount              int           `json:"media_count"`
				Name                    string        `json:"name"`
				NormalFollowersCount    int           `json:"normal_followers_count"`
				PinnedTweetIdsStr       []string      `json:"pinned_tweet_ids_str"`
				PossiblySensitive       bool          `json:"possibly_sensitive"`
				ProfileBannerUrl        string        `json:"profile_banner_url"`
				ProfileImageUrlHttps    string        `json:"profile_image_url_https"`
				ProfileInterstitialType string        `json:"profile_interstitial_type"`
				Protected               bool          `json:"protected"`
				ScreenName              string        `json:"screen_name"`
				StatusesCount           int           `json:"statuses_count"`
				TranslatorType          string        `json:"translator_type"`
				Url                     string        `json:"url"`
				Verified                bool          `json:"verified"`
				WithheldInCountries     []interface{} `json:"withheld_in_countries"`
			} `json:"legacy"`
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
				Typename       string         `json:"__typename"`
				BirdwatchPivot BirdwatchPivot `json:"birdwatch_pivot"`
				Core           struct {
					UserResults UserResult `json:"user_results"`
				} `json:"core"`
				Source    string    `json:"source"`
				Legacy    Legacy    `json:"legacy"`
				Tombstone Tombstone `json:"tombstone"`
				Card      Card      `json:"card"`
			} `json:"result"`
		} `json:"quoted_status_result"`
	}

	Result struct {
		TweetInfo           // TODO: On Withheld, TweetInfo is under "Tweet"
		Tweet     TweetInfo `json:"tweet"`
		Typename  string    `json:"__typename"`
		Tombstone Tombstone `json:"tombstone"`
		Card      Card      `json:"card"`
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
			DataV1 struct {
				Classification string `json:"classification"`
				Summary        struct {
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
				} `json:"summary"`
				MisleadingTags     []string `json:"misleading_tags"`
				TrustworthySources bool     `json:"trustworthy_sources"`
			} `json:"data_v1"`
			DecidedBy    string   `json:"decided_by"`
			RatingStatus string   `json:"rating_status"`
			HelpfulTags  []string `json:"helpful_tags"`
			TweetResults struct {
				Result struct {
					RestId string `json:"rest_id"`
				} `json:"result"`
			} `json:"tweet_results"`
			CreatedAt int64 `json:"created_at"`
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
			ThreadedConversationWithInjectionsV2 struct {
				Instructions []struct {
					Type    string `json:"type"`
					Entries []struct {
						EntryId   string `json:"entryId"`
						SortIndex string `json:"sortIndex"`
						Content   struct {
							EntryType   string `json:"entryType"`
							Typename    string `json:"__typename"`
							ItemContent struct {
								ItemType     string `json:"itemType"`
								Typename     string `json:"__typename"`
								TweetResults struct {
									Result Result `json:"result"`
								} `json:"tweet_results"`
								TweetDisplayType    string `json:"tweetDisplayType"`
								HasModeratedReplies bool   `json:"hasModeratedReplies"`
							} `json:"itemContent"`
						} `json:"content"`
					} `json:"entries,omitempty"`
					Direction string `json:"direction,omitempty"`
				} `json:"instructions"`
			} `json:"threaded_conversation_with_injections_v2"`
		} `json:"data"`
	}
)

func (t *TweetResponse) Tweet(tweetID string) Result {
	if len(t.Data.ThreadedConversationWithInjectionsV2.Instructions) == 0 {
		return Result{}
	}

	for _, entry := range t.Data.ThreadedConversationWithInjectionsV2.Instructions[0].Entries {
		if entry.EntryId == fmt.Sprintf("tweet-%s", tweetID) {
			return entry.Content.ItemContent.TweetResults.Result
		}
	}

	return Result{}
}

func (u *UserResult) Author() string {
	var sb strings.Builder

	sb.WriteString(
		fmt.Sprintf(
			"<b>%s</b> (<a href=\"https://twitter.com/%s\">@%s</a>",
			utils.Escape(u.Result.Legacy.Name),
			u.Result.Legacy.ScreenName,
			u.Result.Legacy.ScreenName,
		),
	)

	if u.Result.Legacy.Verified {
		sb.WriteString(" ✅")
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
		return m.MediaUrlHttps
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
