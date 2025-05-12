package internal

import (
	"log/slog"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// IsbaseRarity check if a card is a C / U / R / RR
func (gc *GameConfig) IsbaseRarity(card Card) bool {
	for _, rarity := range gc.BaseRarity {
		if rarity == card.Rarity && gc.isTrullyNotFoil(card) {
			return true
		}
	}
	return false
}

func (gc *GameConfig) isTrullyNotFoil(card Card) bool {
	for _, _suffix := range gc.FoilSuffix {
		if strings.HasSuffix(card.ID, _suffix) {
			return false
		}
	}
	return true
}

func scrapeCardsPage(gc *GameConfig, page_num int) (bool, error) {
	mainHTML := gc.GetDocument(page_num)
	cards := gc.LoopCards(&mainHTML)
	if cards.Length() == 0 {
		slog.Warn("no cards found")
		return true, nil
	}
	cards.Each(func(i int, s *goquery.Selection) {
		card := gc.ExtractData(gc, s)
		card.SaveCardOnDisk()
	})

	return false, nil

}

func ScrapeAllCards(config *GameConfig) {
	page := 1
	for {
		finished, err := scrapeCardsPage(config, page)
		if err != nil {
			slog.Error("scrape err", err)
		}
		if finished {
			break
		}
		page++
		time.Sleep(500 * time.Millisecond)
	}
}
