package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func loadHTMLFixture(t *testing.T, path string) *goquery.Document {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture %s: %v", path, err)
	}
	defer f.Close()
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatalf("parse html %s: %v", path, err)
	}
	return doc
}

func firstListingProduct(t *testing.T) *goquery.Selection {
	t.Helper()
	doc := loadHTMLFixture(t, "testdata/products_listing.html")
	sel := classicProductsLoop(doc)
	return sel.First()
}

// Given listing HTML, When looping products, Then returns items from the main list only
func TestProductsLoop_ReturnsMainListItems(t *testing.T) {
	doc := loadHTMLFixture(t, "testdata/products_listing.html")

	sel := classicProductsLoop(doc)

	if sel.Length() == 0 {
		t.Fatal("expected products, got 0")
	}
	// Should only return items from ul.products__lists, not the slider
	if sel.Length() != 12 {
		t.Errorf("expected 12 products from main list, got %d", sel.Length())
	}
}

// Given a product selection, When extracting listing, Then title matches expected
func TestExtractListing_Title(t *testing.T) {
	sel := firstListingProduct(t)

	title, _, _, _, _ := classicExtractListing(sel)

	if strings.TrimSpace(title) == "" {
		t.Error("expected non-empty title")
	}
	if title != "葬送のフリーレン Vol.2" {
		t.Errorf("expected '葬送のフリーレン Vol.2', got %q", title)
	}
}

// Given a product selection, When extracting listing, Then image URL is absolute https
func TestExtractListing_ImageURL(t *testing.T) {
	sel := firstListingProduct(t)

	_, imageURL, _, _, _ := classicExtractListing(sel)

	if !strings.HasPrefix(imageURL, "https://") {
		t.Errorf("expected full https image URL, got %q", imageURL)
	}
}

// Given a product selection, When extracting listing, Then detail URL is correct
func TestExtractListing_DetailURL(t *testing.T) {
	sel := firstListingProduct(t)

	_, _, _, detailURL, _ := classicExtractListing(sel)

	if detailURL != "https://ws-tcg.com/products/sfn_bp_vol2/" {
		t.Errorf("expected sfn_bp_vol2 URL, got %q", detailURL)
	}
}

// Given date text with double-digit month/day, When parsing, Then returns YYYY/MM/DD
func TestParseJaReleaseDate_Standard(t *testing.T) {
	result := parseJaReleaseDate("発売日：2026年5月15日(金)")
	if result != "2026/05/15" {
		t.Errorf("expected '2026/05/15', got %q", result)
	}
}

// Given date text with single-digit month and day, When parsing, Then zero-pads
func TestParseJaReleaseDate_ZeroPads(t *testing.T) {
	result := parseJaReleaseDate("発売日：2024年1月9日(火)")
	if result != "2024/01/09" {
		t.Errorf("expected '2024/01/09', got %q", result)
	}
}

// Given detail HTML with 作品番号, When extracting detail, Then returns correct LicenceCode
func TestExtractDetail_LicenceCode(t *testing.T) {
	doc := loadHTMLFixture(t, "testdata/products_detail_ddd.html")

	licenceCode, _ := classicExtractDetail(doc)

	if licenceCode != "DDD" {
		t.Errorf("expected 'DDD', got %q", licenceCode)
	}
}

// Given detail HTML with WS_xxx card images, When extracting detail, Then returns SetCode
func TestExtractDetail_SetCode(t *testing.T) {
	doc := loadHTMLFixture(t, "testdata/products_detail_ddd.html")

	_, setCode := classicExtractDetail(doc)

	if setCode != "S129" {
		t.Errorf("expected 'S129', got %q", setCode)
	}
}

// Given card list API JSON for last page, When parsing, Then returns booster SetCode
func TestParseSetCodeFromCardListJSON_BoosterCard(t *testing.T) {
	// Given
	data, err := os.ReadFile("testdata/cardlist_api_ddd_lastpage.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	// When
	setCode := classicParseSetCodeFromCardListJSON(data, "DDD")
	// Then
	if setCode != "S129" {
		t.Errorf("expected 'S129', got %q", setCode)
	}
}

// Given a product selection, When extracting listing, Then ProductType matches expected
func TestExtractListing_ProductType(t *testing.T) {
	sel := firstListingProduct(t)

	_, _, _, _, productType := classicExtractListing(sel)

	if productType != "ブースターパック" {
		t.Errorf("expected 'ブースターパック', got %q", productType)
	}
}

// Given detail HTML with no card images, When extracting detail, Then SetCode is empty
func TestExtractDetail_SetCodeMissing(t *testing.T) {
	doc := loadHTMLFixture(t, "testdata/products_detail_no_setcode.html")

	licenceCode, setCode := classicExtractDetail(doc)

	if licenceCode != "SFN" {
		t.Errorf("expected LicenceCode 'SFN', got %q", licenceCode)
	}
	if setCode != "" {
		t.Errorf("expected empty SetCode, got %q", setCode)
	}
}
