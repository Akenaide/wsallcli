/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"wsallcli/internal"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/cobra"
	"golang.org/x/net/html"
)

func roseExtractData(gc *internal.GameConfig, mainHTML *goquery.Selection) internal.Card {
	// Helper function to process numeric values
	processInt := func(st string) string {
		if st == "" || st == "-" {
			return "0"
		}
		return st
	}

	// Extract basic info
	cardCode := strings.TrimSpace(mainHTML.Find(".dl-Item:has(dt span:contains('カード番号')) dd p").Text())
	var set, release, id string
	if parts := strings.Split(cardCode, "/"); len(parts) > 1 {
		set = parts[0]
		if subParts := strings.Split(parts[1], "-"); len(subParts) > 1 {
			release = subParts[0]
			id = subParts[1]
		}
	}

	// Extract names
	heading := mainHTML.Find(".item-Heading")
	jpName := strings.TrimSpace(heading.Contents().First().Text())
	name := strings.TrimSpace(heading.Find("span").Text())

	// Extract card type and map to abbreviation
	cardType := strings.TrimSpace(mainHTML.Find(".dl-Item:has(dt span:contains('種類')) dd p").Text())
	switch cardType {
	case "イベント":
		cardType = "EV"
	case "キャラ":
		cardType = "CH"
	case "クライマックス":
		cardType = "CX"
	}

	// Extract color from image
	colorImg, _ := mainHTML.Find(".dl-Item:has(dt span:contains('色')) dd img").Attr("src")
	color := "UNKNOWN"
	if colorImg != "" {
		colorParts := strings.Split(colorImg, "/")
		if len(colorParts) > 0 {
			color = strings.ToUpper(strings.TrimSuffix(colorParts[len(colorParts)-1], ".png"))
		}
	}

	// Extract soul count
	soul := strconv.Itoa(mainHTML.Find(".dl-Item:has(dt span:contains('ソウル')) dd img").Length())

	// Extract triggers
	var triggers []string
	triggerText := strings.TrimSpace(mainHTML.Find(".dl-Item:has(dt span:contains('トリガー')) dd").Text())
	if triggerText != "-" {
		mainHTML.Find(".dl-Item:has(dt span:contains('トリガー')) dd img").Each(func(i int, s *goquery.Selection) {
			if src, exists := s.Attr("src"); exists {
				triggerParts := strings.Split(src, "/")
				if len(triggerParts) > 0 {
					triggerKey := strings.TrimSuffix(triggerParts[len(triggerParts)-1], ".png")
					if mapped, ok := gc.TriggerMap[triggerKey]; ok {
						triggers = append(triggers, mapped)
					}
				}
			}
		})
	}

	// Extract ability text (split by <br>)
	var ability []string
	abilityText, _ := mainHTML.Find(".dl-Item:has(dt span:contains('テキスト')) dd p").Html()
	for _, line := range strings.Split(abilityText, "<br/>") {
		cleanLine := strings.TrimSpace(goquery.NewDocumentFromNode(&html.Node{
			Type: html.TextNode,
			Data: line,
		}).Text())
		if cleanLine != "" {
			ability = append(ability, cleanLine)
		}
	}

	// Extract special attributes (特徴)
	var specialAttrib []string
	specialAttrText := strings.TrimSpace(mainHTML.Find(".dl-Item:has(dt span:contains('特徴')) dd p").Text())
	slog.Info(specialAttrText)
	if specialAttrText != "" && specialAttrText != "-" {
		specialAttrib = strings.Split(specialAttrText, "・") // Split by Japanese middle dot
	}

	// Extract image URL
	imageURL, _ := mainHTML.Find(".thumbnail-Inner img").Attr("src")

	// Build the card
	card := internal.Card{
		Set:           set,
		SetName:       strings.TrimSpace(mainHTML.Find(".dl-Item:has(dt span:contains('ネオスタンダード区分')) dd p").Text()),
		Release:       release,
		ID:            id,
		JpName:        jpName,
		Name:          name,
		CardType:      cardType,
		Colour:        color,
		Level:         processInt(strings.TrimSpace(mainHTML.Find(".dl-Item:has(dt span:contains('レベル')) dd p").Text())),
		Cost:          processInt(strings.TrimSpace(mainHTML.Find(".dl-Item:has(dt span:contains('コスト')) dd p").Text())),
		Power:         processInt(strings.TrimSpace(mainHTML.Find(".dl-Item:has(dt span:contains('パワー')) dd p").Text())),
		Soul:          soul,
		Rarity:        strings.TrimSpace(mainHTML.Find(".dl-Item:has(dt span:contains('レアリティ')) dd p").Text()),
		FlavourText:   strings.TrimSpace(mainHTML.Find(".dl-Item:has(dt span:contains('フレーバー')) dd p").Text()),
		Trigger:       triggers,
		Ability:       ability,
		SpecialAttrib: specialAttrib,
		Version:       internal.CardModelVersion,
		Cardcode:      cardCode,
		ImageURL:      imageURL,
	}

	card.LogCard()
	return card
}

func getDocument(page_num int) goquery.Document {

	url := fmt.Sprint("https://ws-rose.com/cardlist/cardsearch_ex?page=", page_num)
	slog.Info("url", url)
	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Cookie", "cardlist_search_sort=new; cardlist_view=text")
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	response, err := client.Do(req)
	if err != nil {
		slog.Warn("Error", err)
	}
	defer response.Body.Close()
	mainHTML, err := goquery.NewDocumentFromReader(response.Body)
	fmt.Println(response.Header)
	if err != nil {
		slog.Warn("err get mainHTML", err)
	}

	return *mainHTML
}

func roseLoop(document *goquery.Document) goquery.Selection {
	return *document.Find(".ex-item.result-Item.card-Item")
}

var RoseConfig = internal.GameConfig{
	FoilSuffix: []string{
		"SP",
		"S",
		"R",
	},
	BaseRarity: []string{
		"C",
		"PR",
		"R",
		"RR",
		"T",
	},
	TriggerMap: map[string]string{
		"soul":     "SOUL",
		"salvage":  "COMEBACK",
		"draw":     "DRAW",
		"stock":    "POOL",
		"treasure": "TREASURE",
		"shot":     "SHOT",
		"bounce":   "RETURN",
		"gate":     "GATE",
		"standby":  "STANDBY",
		"choice":   "CHOICE",
	},
	ExtractData: roseExtractData,
	LoopCards:   roseLoop,
	GetDocument: getDocument,
}

// roseCmd represents the rose command
var roseCmd = &cobra.Command{
	Use:   "rose",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("rose called")
		internal.Fetch(&RoseConfig)
	},
}

func init() {
	rootCmd.AddCommand(roseCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// roseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// roseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
