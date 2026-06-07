package cmd

import (
	"testing"

	"wsallcli/internal"
)

// Given a valid wsoffdata directory, When building the index, Then setName maps to correct licenceCode and setCode
func TestBuildWsoffdataIndex_MapsSetNameToInfo(t *testing.T) {
	index, _, err := buildWsoffdataIndex("testdata/wsoffdata")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	infos, ok := index["東方Project ～ Black and White Lotus Land."]
	if !ok {
		t.Fatal("expected THP setName in index")
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(infos))
	}
	if infos[0].LicenceCode != "THP" {
		t.Errorf("expected LicenceCode 'THP', got %q", infos[0].LicenceCode)
	}
	if infos[0].SetCode != "W103" {
		t.Errorf("expected SetCode 'W103', got %q", infos[0].SetCode)
	}
}

// Given a wsoffdata directory with multiple sets, When building the index, Then all sets are indexed
func TestBuildWsoffdataIndex_IndexesAllSets(t *testing.T) {
	index, _, err := buildWsoffdataIndex("testdata/wsoffdata")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(index) != 3 {
		t.Errorf("expected 3 sets in index, got %d", len(index))
	}
}

// Given products with missing SetCode, When enriching, Then matched products are patched
func TestEnrichProducts_FillsMissingSetCode(t *testing.T) {
	index := map[string][]setInfo{
		"東方Project ～ Black and White Lotus Land.": {{LicenceCode: "THP", SetCode: "W103"}},
	}
	products := []internal.Product{
		{Title: "東方Project ～ Black and White Lotus Land.", LicenceCode: "THP", SetCode: ""},
	}

	enrichProducts(products, index)

	if products[0].SetCode != "W103" {
		t.Errorf("expected SetCode 'W103', got %q", products[0].SetCode)
	}
}

// Given a product with existing SetCode, When enriching, Then it is not overwritten
func TestEnrichProducts_DoesNotOverwriteExistingSetCode(t *testing.T) {
	index := map[string][]setInfo{
		"葬送のフリーレン": {{LicenceCode: "SFN", SetCode: "SE40"}},
	}
	products := []internal.Product{
		{Title: "葬送のフリーレン", LicenceCode: "SFN", SetCode: "S128"},
	}

	enrichProducts(products, index)

	if products[0].SetCode != "S128" {
		t.Errorf("expected SetCode to remain 'S128', got %q", products[0].SetCode)
	}
}

// Given a product with no match in index, When enriching, Then SetCode stays empty
func TestEnrichProducts_NoMatchLeavesSetCodeEmpty(t *testing.T) {
	index := map[string][]setInfo{}
	products := []internal.Product{
		{Title: "Unknown Title", LicenceCode: "UNK", SetCode: ""},
	}

	enrichProducts(products, index)

	if products[0].SetCode != "" {
		t.Errorf("expected SetCode to remain empty, got %q", products[0].SetCode)
	}
}

// Given a product matching multiple setcodes, When enriching, Then SetCode stays empty (ambiguous)
func TestEnrichProducts_MultipleMatchesSkips(t *testing.T) {
	index := map[string][]setInfo{
		"Ambiguous Title": {
			{LicenceCode: "AAA", SetCode: "W01"},
			{LicenceCode: "AAA", SetCode: "SE01"},
		},
	}
	products := []internal.Product{
		{Title: "Ambiguous Title", LicenceCode: "AAA", SetCode: ""},
	}

	enrichProducts(products, index)

	if products[0].SetCode != "" {
		t.Errorf("expected SetCode to stay empty on ambiguous match, got %q", products[0].SetCode)
	}
}

// Given a product title with curly quotes, When enriching, Then it matches a wsoffdata entry with straight quotes
func TestEnrichProducts_NormalizesCurlyQuotes(t *testing.T) {
	index := map[string][]setInfo{
		"魔法少女リリカルなのはA's": {{LicenceCode: "NA", SetCode: "W12"}},
	}
	products := []internal.Product{
		// Title uses RIGHT SINGLE QUOTATION MARK (U+2019) instead of apostrophe
		{Title: "魔法少女リリカルなのはA’s", LicenceCode: "NA", SetCode: ""},
	}

	enrichProducts(products, index)

	if products[0].SetCode != "W12" {
		t.Errorf("expected SetCode 'W12', got %q", products[0].SetCode)
	}
}

// Given an ambiguous title match where all candidates share the same SetCode, When enriching, Then the shared SetCode is used
func TestEnrichProducts_AmbiguousSameSetCodeResolves(t *testing.T) {
	index := map[string][]setInfo{
		"富士見ファンタジア文庫": {
			{LicenceCode: "F35", SetCode: "W65"},
			{LicenceCode: "Fab", SetCode: "W65"},
			{LicenceCode: "Fks", SetCode: "W65"},
		},
	}
	products := []internal.Product{
		{Title: "富士見ファンタジア文庫", LicenceCode: "", SetCode: ""},
	}

	enrichProducts(products, index)

	if products[0].SetCode != "W65" {
		t.Errorf("expected SetCode 'W65', got %q", products[0].SetCode)
	}
}

// Given a product with no title match but a unique licence code in wsoffdata, When enriching, Then SetCode is set via licence code fallback
func TestEnrichProducts_LicenceCodeFallback(t *testing.T) {
	index := map[string][]setInfo{}
	licenceIndex := map[string][]setInfo{
		"GCR": {{LicenceCode: "GCR", SetCode: "SE48"}},
	}
	products := []internal.Product{
		{Title: "ガールズバンドクライ", LicenceCode: "GCR", SetCode: ""},
	}

	enrichProductsFull(products, index, licenceIndex)

	if products[0].SetCode != "SE48" {
		t.Errorf("expected SetCode 'SE48', got %q", products[0].SetCode)
	}
}

// Given a product whose wsoffdata setName is prefixed with productType, When enriching, Then it matches via the productType+title fallback
func TestEnrichProducts_ProductTypePrefixFallback(t *testing.T) {
	index := map[string][]setInfo{
		"プレミアムブースター BanG Dream! 10th Anniversary！": {{LicenceCode: "BD", SetCode: "WE49"}},
	}
	products := []internal.Product{
		{Title: "BanG Dream! 10th Anniversary！", LicenceCode: "", SetCode: "", ProductType: "プレミアムブースター"},
	}

	enrichProductsFull(products, index, nil)

	if products[0].SetCode != "WE49" {
		t.Errorf("expected SetCode 'WE49', got %q", products[0].SetCode)
	}
	if products[0].LicenceCode != "BD" {
		t.Errorf("expected LicenceCode 'BD', got %q", products[0].LicenceCode)
	}
}

// Given a プレミアムブースター product with only W candidates, When enriching, Then no match is made (W sets are invalid for premium)
func TestEnrichProducts_PremiumBoosterRequiresExtraSet(t *testing.T) {
	index := map[string][]setInfo{
		"アイドルマスター ミリオンライブ！": {
			{LicenceCode: "IAS", SetCode: "S61"},
			{LicenceCode: "IMS", SetCode: "S61"},
		},
	}
	products := []internal.Product{
		{Title: "アイドルマスター ミリオンライブ！", LicenceCode: "", SetCode: "", ProductType: "プレミアムブースター"},
	}

	enrichProductsFull(products, index, nil)

	if products[0].SetCode != "" {
		t.Errorf("expected SetCode to stay empty (no SE/WE candidate), got %q", products[0].SetCode)
	}
}

// Given a ブースターパック product with SE/WE and W candidates, When enriching, Then SE/WE candidates are excluded
func TestEnrichProducts_BoosterPackExcludesExtraSets(t *testing.T) {
	index := map[string][]setInfo{
		"ラブライブ！サンシャイン!!": {
			{LicenceCode: "LSS", SetCode: "W45"},
			{LicenceCode: "LSS", SetCode: "WE27"},
		},
	}
	products := []internal.Product{
		{Title: "ラブライブ！サンシャイン!!", LicenceCode: "LLS", SetCode: "", ProductType: "ブースターパック"},
	}

	enrichProductsFull(products, index, nil)

	if products[0].SetCode != "W45" {
		t.Errorf("expected SetCode 'W45', got %q", products[0].SetCode)
	}
}

// Given a product with a filled LicenceCode, When enriching with a match, Then LicenceCode is also filled
func TestEnrichProducts_FillsLicenceCodeWhenEmpty(t *testing.T) {
	index := map[string][]setInfo{
		"TVアニメ『ダンダダン』Vol.2": {{LicenceCode: "DDD", SetCode: "S129"}},
	}
	products := []internal.Product{
		{Title: "TVアニメ『ダンダダン』Vol.2", LicenceCode: "", SetCode: ""},
	}

	enrichProducts(products, index)

	if products[0].LicenceCode != "DDD" {
		t.Errorf("expected LicenceCode 'DDD', got %q", products[0].LicenceCode)
	}
	if products[0].SetCode != "S129" {
		t.Errorf("expected SetCode 'S129', got %q", products[0].SetCode)
	}
}
