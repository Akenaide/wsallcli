package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"

	"github.com/PuerkitoBio/goquery"
)

const (
	CardModelVersion string = "3"
)

// Card info to export
type Card struct {
	Set               string   `json:"set"`
	SetName           string   `json:"setName"`
	Side              string   `json:"side"`
	Release           string   `json:"release"`
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	JpName            string   `json:"jpName"`
	CardType          string   `json:"cardType"`
	Colour            string   `json:"colour"`
	Level             string   `json:"level"`
	Cost              string   `json:"cost"`
	Power             string   `json:"power"`
	Soul              string   `json:"soul"`
	Rarity            string   `json:"rarity"`
	BreakDeckbuilding bool     `json:"breakDeckbuilding"`
	ENEquivalent      bool     `json:"EN_Equivalent"`
	FlavourText       string   `json:"flavourText"`
	Trigger           []string `json:"trigger"`
	Ability           []string `json:"ability"`
	SpecialAttrib     []string `json:"specialAttrib"`
	Version           string   `json:"version"`
	Cardcode          string   `json:"cardcode"`
	ImageURL          string   `json:"imageURL"`
	Tags              []string `json:"tags"`
}

func (card *Card) LogCard() {
	slog.Info("Card details",
		"set", card.Set,
		"setName", card.SetName,
		"side", card.Side,
		"release", card.Release,
		"id", card.ID,
		"name", card.Name,
		"jpName", card.JpName,
		"cardType", card.CardType,
		"colour", card.Colour,
		"level", card.Level,
		"cost", card.Cost,
		"power", card.Power,
		"soul", card.Soul,
		"rarity", card.Rarity,
		"flavourText", card.FlavourText,
		"trigger", card.Trigger,
		"ability", card.Ability,
		"specialAttrib", card.SpecialAttrib,
		"version", card.Version,
		"cardcode", card.Cardcode,
		"imageURL", card.ImageURL,
	)
}

func (c *Card) SaveCardOnDisk() {

	res, errMarshal := json.Marshal(c)
	if errMarshal != nil {
		slog.Error("error marshal", errMarshal)
	}
	var buffer bytes.Buffer
	cardName := fmt.Sprintf("%v-%v%v-%v.json", c.Set, c.Side, c.Release, c.ID)
	dirName := filepath.Join(c.Set, fmt.Sprintf("%v%v", c.Side, c.Release))
	os.MkdirAll(dirName, 0o744)
	out, err := os.Create(filepath.Join(dirName, cardName))
	defer out.Close()
	if err != nil {
		slog.Error("write error", err.Error())
	}
	json.Indent(&buffer, res, "", "\t")
	buffer.WriteTo(out)
	slog.Info("Finish card- : ", cardName)
}

type GameConfig struct {
	BaseRarity  []string
	FoilSuffix  []string
	TriggerMap  map[string]string
	ExtractData func(*GameConfig, *goquery.Selection) Card
	ListURL     string
	URLValue    url.Values
	LoopCards   func(*goquery.Document) goquery.Selection
	GetDocument func(page_num int) goquery.Document
}
