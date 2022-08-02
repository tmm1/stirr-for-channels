package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robbiet480/stirr-for-channels/internal/xmltv"
)

// timestamps look like 20210422030000, UTC
// YYYYMMDDHHMMSS
// 20060102150405

const stLayout = "20060102150405"

var (
	seasonEpisodeTitleMatcher = regexp.MustCompile(`^(.+)\s*\(S(\d+),E(\d+)\)$`)
	seasonEpisodeDescMatcher = regexp.MustCompile(`^\s*(?:S(\d+)[ :,]?)?E(\d+)[: -]\s*(.+)$`)
	ratingDescMatcher = regexp.MustCompile(`\[(?:Not Rated|Rated (.+?))\]`)
)

// type StirrTime time.Time

// // UnmarshalJSON Parses the json string in the custom format
// func (st *StirrTime) UnmarshalJSON(b []byte) (err error) {
// 	s := strings.Trim(string(b), `"`)
// 	nt, err := time.Parse(stLayout, s)
// 	*st = StirrTime(nt)
// 	return
// }

// // MarshalJSON writes a quoted string in the custom format
// func (st StirrTime) MarshalJSON() ([]byte, error) {
// 	return []byte(st.String()), nil
// }

// // String returns the time in the custom format
// func (st *StirrTime) String() string {
// 	t := time.Time(*st)
// 	return fmt.Sprintf("%q", t.Format(stLayout))
// }

// StirrTime is the time.Time with JSON marshal and unmarshal capability
type StirrTime struct {
	time.Time
}

// UnmarshalJSON will unmarshal using 2006-01-02T15:04:05+07:00 layout
func (t *StirrTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	parsed, err := time.Parse(stLayout, s)
	if err != nil {
		return err
	}

	t.Time = parsed
	return nil
}

// MarshalJSON will marshal using 2006-01-02T15:04:05+07:00 layout
func (t *StirrTime) MarshalJSON() ([]byte, error) {
	s := t.Format(stLayout)
	return []byte(s), nil
}

type Channel struct {
	DisplayName string `json:"display-name"`
	Icon        struct {
		Src string `json:"src"`
	} `json:"icon"`
	ID         string `json:"id"`
	Categories []struct {
		Name string `json:"name"`
		UUID string `json:"uuid"`
	} `json:"categories"`
}

type Category struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

type Lineup struct {
	Categories []Category `json:"categories"`
	Channels   []Channel  `json:"channel"`
}

type Image struct {
	Width  int    `json:"width,string"`
	Height int    `json:"height,string"`
	Text   string `json:"text"`
	URL    string `json:"url"`
}

type ChannelStatus struct {
	ID       string
	Channel  *Channel
	Programs []Program
	Number   int
	Rss      struct {
		XmlnsSinclair string `json:"xmlns:sinclair"`
		XmlnsMedia    string `json:"xmlns:media"`
		Channel       struct {
			Item struct {
				Link     string `json:"link"`
				Category string `json:"category"`
				GUID     struct {
					IsPermalink string `json:"isPermaLink"`
					Content     string `json:"content"`
				} `json:"guid"`
				Pubdate      time.Time `json:"pubDate"`
				MediaContent struct {
					AdPreroll  string `json:"sinclair:ad_preroll"`
					AdPostroll string `json:"sinclair:ad_postroll"`
					Attributes struct {
						Ads struct {
							DAIAdParams struct {
								DescriptionURL string `json:"description_url"`
							} `json:"sinclair:daiAdParams"`
						} `json:"sinclair:ads"`
					} `json:"sinclair:attributes"`
					Status struct {
						DisplayText string `json:"sinclair:displayText"`
						State       string `json:"state"`
						Reason      string `json:"reason"`
					} `json:"sinclair:status"`
					MediaTitle struct {
						Content string `json:"content"`
					} `json:"media:title"`
					MediaDescription struct {
						Content string `json:"content"`
					} `json:"media:description"`
					MediaThumbnail []Image `json:"media:thumbnail"`
					Logo           Image   `json:"sinclair:logo"`
					Medium         string  `json:"medium"`
					SinclairIdent  string  `json:"sinclair:ident"`
					Expression     string  `json:"expression"`
					Type           string  `json:"type"`
					URL            string  `json:"url"`
					Duration       string  `json:"duration"`
					IsLive         bool    `json:"sinclair:isLive,string"`
					IsLiveProgram  bool    `json:"sinclair:isLiveProgram,string"`
					SinclairURL    string  `json:"sinclair:url"`
				} `json:"media:content"`
			} `json:"item"`
			ID          string `json:"id"`
			Link        string `json:"link"`
			Description string `json:"description"`
			Title       string `json:"title"`
		} `json:"channel"`
		Version string `json:"version"`
	} `json:"rss"`
}

func (c *ChannelStatus) M3ULine() string {
	headerPieces := []string{
		"#EXTINF:0",
		fmt.Sprintf(`channel-id="%s"`, c.ID),
		fmt.Sprintf(`tvg-logo="%s"`, c.Rss.Channel.Item.MediaContent.Logo.URL),
		fmt.Sprintf(`tvg-name="%s"`, c.Rss.Channel.Title),
	}

	cleaned := strings.ReplaceAll(strings.Join(headerPieces, " "), "\n", "")

	return fmt.Sprintf("%s, %s\n%s", cleaned, c.Rss.Channel.Title, c.Rss.Channel.Item.Link)
}

func (c *ChannelStatus) XMLTV() xmltv.Channel {
	return xmltv.Channel{
		DisplayNames: []xmltv.CommonElement{{
			Value: strings.TrimSpace(c.Rss.Channel.Title),
		}, {
			Value: strconv.Itoa(c.Number),
		}},
		Icons: []xmltv.Icon{{
			Source: c.Rss.Channel.Item.MediaContent.Logo.URL,
			Width:  340,
			Height: 255,
		}},
		ID: c.ID,
	}
}

type Program struct {
	Title       xmltv.CommonElement   `json:"title"`
	IsLive      bool                  `json:"sinclair:isLiveProgram,string"`
	Description xmltv.CommonElement   `json:"desc"`
	Start       StirrTime             `json:"start"`
	Stop        StirrTime             `json:"stop"`
	Channel     string                `json:"channel"`
	Categories  []xmltv.CommonElement `json:"category"`
}

func (p *Program) XMLTV(cs ChannelStatus) xmltv.Programme {
	start := xmltv.Time(p.Start)
	stop := xmltv.Time(p.Stop)
	var live *xmltv.ElementPresent
	if p.IsLive {
		tmp := xmltv.ElementPresent(p.IsLive)
		live = &tmp
	}
	var categories []xmltv.CommonElement
	for _, cat := range cs.Channel.Categories {
		name := cat.Name
		if name == "Movies" {
			name = "Movie"
		}
		if name == "" || name == "Other" {
			continue
		}
		categories = append(categories, xmltv.CommonElement{Value: name})
	}
	for _, c := range p.Categories {
		if c.Value == "HD Unknown" {
			categories = append(categories, xmltv.CommonElement{Value: "HD"})
		} else if c.Value != "category" && c.Value != "Other" {
			categories = append(categories, xmltv.CommonElement{Value: c.Value})
		}
	}
	var season, episode int
	var rating string
	title := p.Title
	desc := p.Description
	if matches := seasonEpisodeTitleMatcher.FindStringSubmatch(title.Value); len(matches) > 1 {
		title.Value = strings.TrimSpace(matches[1])
		season, _ = strconv.Atoi(matches[2])
		episode, _ = strconv.Atoi(matches[3])
	}
	if matches := seasonEpisodeDescMatcher.FindStringSubmatch(desc.Value); len(matches) > 1 {
		desc.Value = strings.TrimSpace(matches[3])
		season, _ = strconv.Atoi(matches[1])
		episode, _ = strconv.Atoi(matches[2])
	}
	if matches := ratingDescMatcher.FindStringSubmatch(desc.Value); len(matches) > 1 {
		// add rating
		rating = matches[1]
		if rating == "" {
			rating = matches[0]
		}
		if rating == "[Not Rated]" {
			rating = "NR"
		}

		// add Movie category if not present
		found := false
		for _, cat := range categories {
			if cat.Value == "Movie" {
				found = true
				break
			}
		}
		if !found {
			categories = append(categories, xmltv.CommonElement{Value: "Movie"})
		}
	}
	pg := xmltv.Programme{
		Titles:       []xmltv.CommonElement{title},
		Descriptions: []xmltv.CommonElement{desc},
		Categories:   categories,
		Start:        &start,
		Stop:         &stop,
		Live:         live,
		Channel:      fmt.Sprintf("stirr-%s", p.Channel),
	}
	if season > 0 && episode > 0 {
		pg.EpisodeNums = []xmltv.EpisodeNum{
			{
				System: "onscreen",
				Value:  fmt.Sprintf("S%dE%d", season, episode),
			},
		}
	} else if episode > 0 {
		pg.EpisodeNums = []xmltv.EpisodeNum{
			{
				System: "onscreen",
				Value:  fmt.Sprintf("E%d", episode),
			},
		}
	}
	if rating != "" {
		pg.Ratings = []xmltv.Rating{
			{
				Value: rating,
			},
		}
	}
	return pg
}

type GuideData struct {
	Channel  []Channel `json:"channel"`
	Programs []Program `json:"programme"`
}

type StationDetection struct {
	Page []struct {
		Type                string `json:"type"`
		PageComponentUUID   string `json:"pageComponentUuid"`
		ComponentInstanceID string `json:"componentInstanceId"`
		Content             string `json:"content"`
		Link                string `json:"link"`
		Background          string `json:"background"`
		DisplayTitle        string `json:"displayTitle"`
		Subtitle            string `json:"subTitle"`
		Button              struct {
			Category     string `json:"category"`
			MediaContent struct {
				MediaTitle struct {
					Content string `json:"content"`
				} `json:"media:title"`
				SinclairAction       string `json:"sinclair:action"`
				SinclairActionConfig struct {
					Station []string `json:"station"`
					City    string   `json:"city"`
				} `json:"sinclair:action_config"`
				URL string `json:"url"`
			} `json:"media:content"`
		} `json:"button"`
		Timer struct {
			Category     string `json:"category"`
			MediaContent struct {
				SinclairAction       string `json:"sinclair:action"`
				SinclairActionConfig struct {
					Station []string `json:"station"`
					City    string   `json:"city"`
				} `json:"sinclair:action_config"`
				URL string `json:"url"`
			} `json:"media:content"`
		} `json:"timer"`
		Logo        []string `json:"logo"`
		Stationtext string   `json:"stationText"`
		Promotext   string   `json:"promoText"`
	} `json:"page"`
	HideNav string `json:"hideNav"`
}
