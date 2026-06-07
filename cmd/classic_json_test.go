package cmd

import (
	"strconv"
	"testing"
)

var testTriggerMap = map[string]string{
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

// parseCardNumber

func TestParseCardNumber_standard(t *testing.T) {
	// Given a standard card number "DC/W01-001"
	// When parseCardNumber is called
	// Then it returns set, side, release, id correctly
	set, side, release, id, ok := parseCardNumber("DC/W01-001")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if set != "DC" {
		t.Errorf("set: got %q, want %q", set, "DC")
	}
	if side != "W" {
		t.Errorf("side: got %q, want %q", side, "W")
	}
	if release != "01" {
		t.Errorf("release: got %q, want %q", release, "01")
	}
	if id != "001" {
		t.Errorf("id: got %q, want %q", id, "001")
	}
}

func TestParseCardNumber_schwarzSide(t *testing.T) {
	// Given a Schwarz-side card number "SY/S01-042"
	// When parseCardNumber is called
	// Then side is "S"
	_, side, _, _, ok := parseCardNumber("SY/S01-042")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if side != "S" {
		t.Errorf("side: got %q, want %q", side, "S")
	}
}

func TestParseCardNumber_noSlash(t *testing.T) {
	// Given a card_number with no "/"
	// When parseCardNumber is called
	// Then ok=false is returned
	_, _, _, _, ok := parseCardNumber("BSF2024-03PR")
	if ok {
		t.Fatal("expected ok=false for card_number with no slash")
	}
}

// parseGifName

func TestParseGifName_yellow(t *testing.T) {
	// Given "[[yellow.gif]]"
	// When parseGifName is called
	// Then it returns "yellow"
	got := parseGifName("[[yellow.gif]]")
	if got != "yellow" {
		t.Errorf("got %q, want %q", got, "yellow")
	}
}

func TestParseGifName_soul(t *testing.T) {
	// Given "[[soul.gif]]"
	// When parseGifName is called
	// Then it returns "soul"
	got := parseGifName("[[soul.gif]]")
	if got != "soul" {
		t.Errorf("got %q, want %q", got, "soul")
	}
}

// parseSoul

func TestParseSoul_single(t *testing.T) {
	// Given "[[soul.gif]]"
	// When parseSoul is called
	// Then it returns "1"
	got := parseSoul("[[soul.gif]]")
	if got != "1" {
		t.Errorf("got %q, want %q", got, "1")
	}
}

func TestParseSoul_double(t *testing.T) {
	// Given "[[soul.gif]][[soul.gif]]"
	// When parseSoul is called
	// Then it returns "2"
	got := parseSoul("[[soul.gif]][[soul.gif]]")
	if got != "2" {
		t.Errorf("got %q, want %q", got, "2")
	}
}

func TestParseSoul_none(t *testing.T) {
	// Given "-"
	// When parseSoul is called
	// Then it returns "0"
	got := parseSoul("-")
	if got != "0" {
		t.Errorf("got %q, want %q", got, "0")
	}
}

// parseTriggers

func TestParseTriggers_none(t *testing.T) {
	// Given "-"
	// When parseTriggers is called
	// Then it returns nil
	got := parseTriggers("-", testTriggerMap)
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestParseTriggers_single(t *testing.T) {
	// Given "[[soul.gif]]"
	// When parseTriggers is called
	// Then it returns ["SOUL"]
	got := parseTriggers("[[soul.gif]]", testTriggerMap)
	if len(got) != 1 || got[0] != "SOUL" {
		t.Errorf("got %v, want [SOUL]", got)
	}
}

func TestParseTriggers_multi(t *testing.T) {
	// Given "[[soul.gif]][[bounce.gif]]"
	// When parseTriggers is called
	// Then it returns ["SOUL", "RETURN"]
	got := parseTriggers("[[soul.gif]][[bounce.gif]]", testTriggerMap)
	if len(got) != 2 || got[0] != "SOUL" || got[1] != "RETURN" {
		t.Errorf("got %v, want [SOUL RETURN]", got)
	}
}

func TestParseTriggers_unknownSkipped(t *testing.T) {
	// Given "[[unknown.gif]]"
	// When parseTriggers is called with a map that has no "unknown" key
	// Then it returns nil (unknown stems are skipped)
	got := parseTriggers("[[unknown.gif]]", testTriggerMap)
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

// parseFeatures

func TestParseFeatures_allValid(t *testing.T) {
	// Given three non-empty, non-dash features
	// When parseFeatures is called
	// Then all three are returned
	got := parseFeatures("魔法", "音楽", "学園")
	if len(got) != 3 {
		t.Errorf("got %v (len %d), want 3 items", got, len(got))
	}
}

func TestParseFeatures_filterDashAndEmpty(t *testing.T) {
	// Given feature1="魔法", feature2="-", feature3=""
	// When parseFeatures is called
	// Then only "魔法" is returned
	got := parseFeatures("魔法", "-", "")
	if len(got) != 1 || got[0] != "魔法" {
		t.Errorf("got %v, want [魔法]", got)
	}
}

func TestParseFeatures_allEmpty(t *testing.T) {
	// Given all features are "-" or ""
	// When parseFeatures is called
	// Then nil is returned
	got := parseFeatures("-", "-", "")
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

// mapCardKind

func TestMapCardKind(t *testing.T) {
	// Given card_kind values "2", "3", "4"
	// When mapCardKind is called
	// Then it returns "CH", "EV", "CX" respectively
	cases := []struct{ in, want string }{
		{"2", "CH"},
		{"3", "EV"},
		{"4", "CX"},
	}
	for _, c := range cases {
		got := mapCardKind(c.in)
		if got != c.want {
			t.Errorf("mapCardKind(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// itemToCard

func TestItemToCard(t *testing.T) {
	// Given a realistic classicAPIItem fixture
	// When itemToCard is called with a trigger map and expansion map
	// Then the Card fields are correctly populated
	item := classicAPIItem{
		CardNumber:  "DC/W01-001",
		CardName:    "学園長のさくら",
		CardKind:    "2",
		Color:       "[[yellow.gif]]",
		Level:       "0",
		Cost:        "0",
		Power:       "500",
		Soul:        "[[soul.gif]]",
		CardTrigger: "-",
		Text:        "【自】 絆／「桜内 義之」<br />【起】 ソウルを＋1。",
		Flavor:      "まあ、ボクも長いこと学園にいるからさ。",
		Picture:     "d/dc_w01/dc_w01_001.png",
		Expansion:   1,
		Rare:        "RR",
		Feature1:    "魔法",
		Feature2:    "-",
		Feature3:    "",
	}
	expansionMap := map[int]string{1: "D.C. D.C.II"}

	card := itemToCard(item, testTriggerMap, expansionMap)

	if card.Set != "DC" {
		t.Errorf("Set: got %q, want %q", card.Set, "DC")
	}
	if card.Side != "W" {
		t.Errorf("Side: got %q, want %q", card.Side, "W")
	}
	if card.Release != "01" {
		t.Errorf("Release: got %q, want %q", card.Release, "01")
	}
	if card.ID != "001" {
		t.Errorf("ID: got %q, want %q", card.ID, "001")
	}
	if card.Cardcode != "DC/W01-001" {
		t.Errorf("Cardcode: got %q", card.Cardcode)
	}
	if card.JpName != "学園長のさくら" {
		t.Errorf("JpName: got %q", card.JpName)
	}
	if card.CardType != "CH" {
		t.Errorf("CardType: got %q, want CH", card.CardType)
	}
	if card.Colour != "YELLOW" {
		t.Errorf("Colour: got %q, want YELLOW", card.Colour)
	}
	if card.Soul != "1" {
		t.Errorf("Soul: got %q, want 1", card.Soul)
	}
	if card.Trigger != nil {
		t.Errorf("Trigger: got %v, want nil", card.Trigger)
	}
	if len(card.Ability) != 2 {
		t.Errorf("Ability: got %d lines, want 2", len(card.Ability))
	}
	if card.FlavourText != item.Flavor {
		t.Errorf("FlavourText: got %q", card.FlavourText)
	}
	if card.SetName != "D.C. D.C.II" {
		t.Errorf("SetName: got %q, want D.C. D.C.II", card.SetName)
	}
	if card.ExpansionID != 1 {
		t.Errorf("ExpansionID: got %d, want 1", card.ExpansionID)
	}
	if len(card.SpecialAttrib) != 1 || card.SpecialAttrib[0] != "魔法" {
		t.Errorf("SpecialAttrib: got %v, want [魔法]", card.SpecialAttrib)
	}
	if card.ImageURL != "https://ws-tcg.com/wordpress/wp-content/images/d/dc_w01/dc_w01_001.png" {
		t.Errorf("ImageURL: got %q", card.ImageURL)
	}
	if card.Rarity != "RR" {
		t.Errorf("Rarity: got %q, want RR", card.Rarity)
	}
	_ = strconv.Itoa(card.ExpansionID) // ensure int type compiles
}
