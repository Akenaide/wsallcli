package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"wsallcli/internal"

	"github.com/spf13/cobra"
)

const (
	classicImageBase  = "https://ws-tcg.com/wordpress/wp-content/images/"
	classicCardsURL   = "https://ws-tcg.com/manage/CardListUser/searchJson?keyword_type[]=all&option_counter=0&option_clock=0&parallel=0&show_page_count=120&view=text&page=%d"
	classicOptionsURL = "https://ws-tcg.com/manage/CardListUser/filter-options"
	classicReferer    = "https://ws-tcg.com/cardlist/search/"
	classicUserAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36"
)


type classicAPIItem struct {
	CardNumber  string `json:"card_number"`
	CardName    string `json:"card_name"`
	CardKind    string `json:"card_kind"`
	Color       string `json:"color"`
	Level       string `json:"level"`
	Cost        string `json:"cost"`
	Power       string `json:"power"`
	Soul        string `json:"soul"`
	CardTrigger string `json:"card_trigger"`
	Text        string `json:"text"`
	Flavor      string `json:"flavor"`
	Picture     string `json:"picture"`
	Expansion   int    `json:"expansion"`
	Rare        string `json:"rare"`
	Feature1    string `json:"feature1"`
	Feature2    string `json:"feature2"`
	Feature3    string `json:"feature3"`
}

type classicAPIResponse struct {
	Items     []classicAPIItem `json:"items"`
	PageCount int              `json:"page_count"`
}

type filterOptionsExpansion struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type filterOptionsResponse struct {
	Expansions []filterOptionsExpansion `json:"expansions"`
}

// parseCardNumber parses "DC/W01-001" into set, side, release, id.
func parseCardNumber(cardNumber string) (set, side, release, id string, ok bool) {
	parts := strings.SplitN(cardNumber, "/", 2)
	if len(parts) != 2 {
		return "", "", "", "", false
	}
	set = parts[0]
	rest := parts[1] // "W01-001"
	if len(rest) == 0 {
		return "", "", "", "", false
	}
	side = string(rest[0])
	dashParts := strings.SplitN(rest[1:], "-", 2)
	if len(dashParts) != 2 {
		return "", "", "", "", false
	}
	release = dashParts[0]
	id = dashParts[1]
	return set, side, release, id, true
}

// parseGifName extracts the stem from "[[soul.gif]]" → "soul".
func parseGifName(s string) string {
	s = strings.TrimPrefix(s, "[[")
	s = strings.TrimSuffix(s, "]]")
	s = strings.TrimSuffix(s, ".gif")
	return s
}

// parseSoul counts occurrences of "[[soul.gif]]"; "-" → "0".
func parseSoul(s string) string {
	if s == "-" {
		return "0"
	}
	return fmt.Sprintf("%d", strings.Count(s, "[[soul.gif]]"))
}

// parseTriggers extracts trigger names from "[[soul.gif]][[bounce.gif]]".
func parseTriggers(s string, triggerMap map[string]string) []string {
	if s == "-" {
		return nil
	}
	var result []string
	// split on "[[" to get individual tokens
	for _, token := range strings.Split(s, "[[") {
		token = strings.TrimSuffix(token, "]]")
		token = strings.TrimSuffix(token, ".gif")
		if token == "" {
			continue
		}
		name, ok := triggerMap[token]
		if !ok {
			slog.Warn("unknown trigger gif", "stem", token)
			continue
		}
		result = append(result, name)
	}
	return result
}

// parseFeatures filters "-" and "" from feature fields.
func parseFeatures(f1, f2, f3 string) []string {
	var result []string
	for _, f := range []string{f1, f2, f3} {
		if f != "" && f != "-" {
			result = append(result, f)
		}
	}
	return result
}

// mapCardKind maps API card_kind value to card type abbreviation.
func mapCardKind(kind string) string {
	switch kind {
	case "2":
		return "CH"
	case "3":
		return "EV"
	case "4":
		return "CX"
	default:
		return ""
	}
}

// itemToCard converts a classicAPIItem to an internal.Card.
func itemToCard(item classicAPIItem, triggerMap map[string]string, expansionMap map[int]string) internal.Card {
	set, side, release, id, ok := parseCardNumber(item.CardNumber)
	if !ok {
		slog.Warn("skipping card with unparseable card_number", "card_number", item.CardNumber)
		return internal.Card{}
	}

	colour := strings.ToUpper(parseGifName(item.Color))

	var ability []string
	for _, line := range strings.Split(item.Text, "<br />") {
		ability = append(ability, line)
	}

	features := parseFeatures(item.Feature1, item.Feature2, item.Feature3)

	card := internal.Card{
		Set:         set,
		SetName:     expansionMap[item.Expansion],
		ExpansionID: item.Expansion,
		Side:        side,
		Release:     release,
		ID:          id,
		Cardcode:    item.CardNumber,
		JpName:      item.CardName,
		CardType:    mapCardKind(item.CardKind),
		Colour:      colour,
		Level:       item.Level,
		Cost:        item.Cost,
		Power:       item.Power,
		Soul:        parseSoul(item.Soul),
		Trigger:     parseTriggers(item.CardTrigger, triggerMap),
		Ability:     ability,
		FlavourText: item.Flavor,
		Rarity:      item.Rare,
		ImageURL:    classicImageBase + item.Picture,
		Version:     internal.CardModelVersion,
	}
	if len(features) > 0 {
		card.SpecialAttrib = features
	}
	return card
}

var classicTriggerMap = map[string]string{
	"soul":      "SOUL",
	"salvage":   "COMEBACK",
	"draw":      "DRAW",
	"stock":     "POOL",
	"treasure":  "TREASURE",
	"shot":      "SHOT",
	"bounce":    "RETURN",
	"gate":      "GATE",
	"standby":   "STANDBY",
	"choice":    "CHOICE",
	"discovery": "DISCOVERY",
	"chance":    "CHANCE",
}

var ClassicConfig = internal.GameConfig{
	TriggerMap: classicTriggerMap,
	FetchPage:  classicFetchPage,
}

var classicExpansionMap map[int]string

func classicRequest(url string) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", classicReferer)
	req.Header.Set("User-Agent", classicUserAgent)
	return client.Do(req)
}

func fetchExpansionMap() (map[int]string, error) {
	resp, err := classicRequest(classicOptionsURL)
	if err != nil {
		return nil, fmt.Errorf("filter-options fetch: %w", err)
	}
	defer resp.Body.Close()

	var opts filterOptionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&opts); err != nil {
		return nil, fmt.Errorf("filter-options decode: %w", err)
	}

	m := make(map[int]string, len(opts.Expansions))
	for _, e := range opts.Expansions {
		m[e.ID] = e.Name
	}
	return m, nil
}

func classicFetchPage(gc *internal.GameConfig, page int) ([]internal.Card, int, error) {
	if page == 1 {
		m, err := fetchExpansionMap()
		if err != nil {
			slog.Warn("could not fetch expansion map, SetName will be empty", "err", err)
			classicExpansionMap = map[int]string{}
		} else {
			classicExpansionMap = m
			slog.Info("loaded expansion map", "count", len(m))
		}
		time.Sleep(500 * time.Millisecond)
	}

	resp, err := classicRequest(fmt.Sprintf(classicCardsURL, page))
	if err != nil {
		return nil, 0, fmt.Errorf("cards fetch page %d: %w", page, err)
	}
	defer resp.Body.Close()

	var apiResp classicAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, 0, fmt.Errorf("cards decode page %d: %w", page, err)
	}

	cards := make([]internal.Card, 0, len(apiResp.Items))
	for _, item := range apiResp.Items {
		card := itemToCard(item, gc.TriggerMap, classicExpansionMap)
		if card.Cardcode != "" {
			cards = append(cards, card)
		}
	}
	return cards, apiResp.PageCount, nil
}

var classicCmd = &cobra.Command{
	Use:   "classic",
	Short: "Commands for the Classic variant of Weiss Schwarz (ws-tcg.com)",
	Long:  `Parent command for Classic-variant subcommands. Currently available: cards.`,
}

var (
	classicPagesFlag  int
	classicRecentFlag bool
)

var classicCardsCmd = &cobra.Command{
	Use:   "cards",
	Short: "Fetch Classic cards from ws-tcg.com JSON API",
	Long: `Fetches Classic Weiss Schwarz cards from the ws-tcg.com JSON API and saves
each card as a JSON file under {Set}/{Side}{Release}/{cardcode}.json.

By default all pages are fetched (120 cards per page, 500ms between requests).
With --pages N, the last N pages are fetched (newest cards). --recent is a
shortcut for --pages 5.

Examples:
  classic cards           # fetch everything
  classic cards -p 1      # fetch only the last page (newest)
  classic cards -p 5      # fetch the last 5 pages
  classic cards --recent  # same as -p 5`,
	Run: func(cmd *cobra.Command, args []string) {
		pages := classicPagesFlag
		if classicRecentFlag {
			pages = 5
		}

		startPage := 1
		endPage := 0

		if pages > 0 {
			// Probe page 1 to get total page count, then fetch last N pages.
			resp, err := classicRequest(fmt.Sprintf(classicCardsURL, 1))
			if err != nil {
				slog.Error("failed to probe page count", "err", err)
				return
			}
			var probe classicAPIResponse
			json.NewDecoder(resp.Body).Decode(&probe)
			resp.Body.Close()
			time.Sleep(500 * time.Millisecond)

			startPage = probe.PageCount - pages + 1
			if startPage < 1 {
				startPage = 1
			}
			endPage = probe.PageCount
		}

		internal.ScrapeAllCards(&ClassicConfig, startPage, endPage)
	},
}

func init() {
	rootCmd.AddCommand(classicCmd)
	classicCmd.AddCommand(classicCardsCmd)

	classicCardsCmd.Flags().IntVarP(&classicPagesFlag, "pages", "p", 0, "Fetch the last N pages (newest cards)")
	classicCardsCmd.Flags().BoolVarP(&classicRecentFlag, "recent", "r", false, "Fetch the last 5 pages (shortcut for -p 5)")
}
