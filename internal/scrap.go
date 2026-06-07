package internal

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
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

func ScrapeProducts(config *ProductsConfig, maxPages int) {
	var products []Product
	var missing []Product
	now := time.Now()

	for page := 1; ; page++ {
		if maxPages > 0 && page > maxPages {
			break
		}

		slog.Info("fetching listing page", "page", page)
		doc := config.GetListingPage(page)
		time.Sleep(500 * time.Millisecond)

		items := config.LoopProducts(&doc)
		if items.Length() == 0 {
			slog.Info("no more products found, stopping", "page", page)
			break
		}
		slog.Info("found products on page", "page", page, "count", items.Length())

		items.Each(func(_ int, s *goquery.Selection) {
			title, imageURL, releaseDateStr, detailURL, productType := config.ExtractListing(s)
			if releaseDateStr == "" {
				return
			}
			releaseDate, err := time.Parse("2006/01/02", releaseDateStr)
			if err != nil || releaseDate.After(now) {
				slog.Info("skipping unreleased product", "title", title, "releaseDate", releaseDateStr)
				return
			}

			slog.Info("fetching product detail", "title", title, "url", detailURL)
			detailDoc := config.GetDetailPage(detailURL)
			time.Sleep(500 * time.Millisecond)

			licenceCode, setCode := config.ExtractDetail(&detailDoc)
			if setCode == "" && config.GetSetCodeFallback != nil {
				slog.Info("image setcode not found, trying cardlist fallback", "title", title, "licenceCode", licenceCode)
				setCode = config.GetSetCodeFallback(licenceCode)
				time.Sleep(500 * time.Millisecond)
			}
			slog.Info("scraped product", "title", title, "licenceCode", licenceCode, "setCode", setCode)
			p := Product{
				ReleaseDate: releaseDateStr,
				Title:       title,
				LicenceCode: licenceCode,
				Image:       imageURL,
				SetCode:     setCode,
				ProductType: productType,
			}
			products = append(products, p)
			if setCode == "" {
				missing = append(missing, p)
			}
		})
	}

	res, _ := json.Marshal(products)
	var buf bytes.Buffer
	json.Indent(&buf, res, "", "\t")
	os.WriteFile("products.json", buf.Bytes(), 0o644)
	slog.Info("wrote products to products.json", "count", len(products))

	if len(missing) > 0 {
		slog.Warn("Products with missing SetCode — investigate manually:")
		for _, p := range missing {
			slog.Warn("  missing SetCode", "title", p.Title, "licenceCode", p.LicenceCode)
		}
	}
}

// ScrapeAllCards fetches cards across pages [startPage, endPage].
// endPage=0 means fetch all pages. When config.FetchPage is set the JSON
// path is used; otherwise the HTML path (GetDocument/LoopCards/ExtractData).
func ScrapeAllCards(config *GameConfig, startPage, endPage int) {
	if config.FetchPage != nil {
		scrapeAllCardsJSON(config, startPage, endPage)
		return
	}
	for page := startPage; ; page++ {
		if endPage > 0 && page > endPage {
			break
		}
		finished, err := scrapeCardsPage(config, page)
		if err != nil {
			slog.Error("scrape err", "err", err)
		}
		if finished {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func scrapeAllCardsJSON(config *GameConfig, startPage, endPage int) {
	for page := startPage; ; page++ {
		if endPage > 0 && page > endPage {
			break
		}
		slog.Info("fetching page", "page", page)
		cards, totalPages, err := config.FetchPage(config, page)
		if err != nil {
			slog.Error("fetch page error", "page", page, "err", err)
			break
		}
		if len(cards) == 0 {
			slog.Info("no cards returned, stopping", "page", page)
			break
		}
		for i := range cards {
			cards[i].SaveCardOnDisk()
		}
		if page >= totalPages {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
}
