package internal

import (
	"log/slog"
	"strings"

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

func Fetch(gc *GameConfig) {
	mainHTML := gc.GetDocument(6)
	cards := gc.LoopCards(&mainHTML)
	if cards.Length() == 0 {
		slog.Warn("no cards found")
	}
	cards.Each(func(i int, s *goquery.Selection) {
		gc.ExtractData(gc, s)
	})

}
