package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"wsallcli/internal"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/cobra"
)

var jaDateRe = regexp.MustCompile(`(\d{4})年(\d+)月(\d+)日`)
var setCodeRe = regexp.MustCompile(`WS_([A-Z0-9]+)_([A-Z0-9]+)_\d+`)
var licenceCodeRe = regexp.MustCompile(`作品番号：(\w+)`)

func parseJaReleaseDate(text string) string {
	m := jaDateRe.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	year, month, day := m[1], m[2], m[3]
	return fmt.Sprintf("%s/%02s/%02s", year, month, day)
}

func productsHTTPGet(url string) *http.Response {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "ja,en;q=0.9")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, _ := client.Do(req)
	return resp
}

func classicProductsGetListingPage(pageNum int) goquery.Document {
	var url string
	if pageNum == 1 {
		url = "https://ws-tcg.com/products/"
	} else {
		url = fmt.Sprintf("https://ws-tcg.com/products/page/%d/", pageNum)
	}
	resp := productsHTTPGet(url)
	defer resp.Body.Close()
	doc, _ := goquery.NewDocumentFromReader(resp.Body)
	return *doc
}

func classicProductsGetDetailPage(url string) goquery.Document {
	resp := productsHTTPGet(url)
	defer resp.Body.Close()
	doc, _ := goquery.NewDocumentFromReader(resp.Body)
	return *doc
}

func classicProductsLoop(doc *goquery.Document) goquery.Selection {
	return *doc.Find("ul.products__lists a.products__link")
}

func classicExtractListing(sel *goquery.Selection) (title, imageURL, releaseDate, detailURL, productType string) {
	detailURL, _ = sel.Attr("href")
	imageURL, _ = sel.Find(".products__thumbImg img").Attr("src")
	title = strings.TrimSpace(sel.Find("p.products__name").Text())
	rawDate := sel.Find("p.products__salesdate").Text()
	releaseDate = parseJaReleaseDate(rawDate)
	productType = strings.TrimSpace(sel.Find(".products__catItem").First().Text())
	return
}

func classicExtractDetail(doc *goquery.Document) (licenceCode, setCode string) {
	doc.Find("p").Each(func(_ int, s *goquery.Selection) {
		text := s.Text()
		if m := licenceCodeRe.FindStringSubmatch(text); m != nil {
			licenceCode = m[1]
		}
	})
	if licenceCode == "" {
		return
	}
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		if setCode != "" {
			return
		}
		src, _ := s.Attr("src")
		if m := setCodeRe.FindStringSubmatch(src); m != nil && m[1] == licenceCode {
			setCode = m[2]
		}
	})
	return
}

var cardListCardNoRe = regexp.MustCompile(`[^/]+/([A-Z]+\d+)-(\S+)`)

type cardListResponse struct {
	Total int `json:"total"`
	Items []struct {
		CardNumber string `json:"card_number"`
	} `json:"items"`
}

// cardListCardNoRe matches card numbers like "DDD/S129-001" → groups: (S129, 001)
// m[1] = set code (e.g. "S129"), m[2] = card suffix (e.g. "001", "P01", "T01")
func classicParseSetCodeFromCardListJSON(body []byte, licenceCode string) string {
	var resp cardListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return ""
	}
	prefix := licenceCode + "/"
	for _, item := range resp.Items {
		if !strings.HasPrefix(item.CardNumber, prefix) {
			continue
		}
		m := cardListCardNoRe.FindStringSubmatch(item.CardNumber)
		if m == nil {
			continue
		}
		suffix := m[2]
		if len(suffix) > 0 && suffix[0] >= '0' && suffix[0] <= '9' {
			return m[1]
		}
	}
	return ""
}

func classicFetchCardListPage(licenceCode string, page int) ([]byte, error) {
	url := fmt.Sprintf(
		"https://ws-tcg.com/manage/CardListUser/searchJson?keyword=&keyword_type%%5B%%5D=all&title_number=%%23%%23%s%%23%%23&show_page_count=30&page=%d",
		licenceCode, page,
	)
	resp := productsHTTPGet(url)
	if resp == nil {
		return nil, fmt.Errorf("nil response")
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func classicGetSetCodeFallback(licenceCode string) string {
	body, err := classicFetchCardListPage(licenceCode, 1)
	if err != nil {
		return ""
	}
	var first cardListResponse
	if err := json.Unmarshal(body, &first); err != nil || first.Total == 0 {
		return ""
	}
	lastPage := (first.Total + 29) / 30
	if lastPage == 1 {
		return classicParseSetCodeFromCardListJSON(body, licenceCode)
	}
	time.Sleep(500 * time.Millisecond)
	lastBody, err := classicFetchCardListPage(licenceCode, lastPage)
	if err != nil {
		return ""
	}
	return classicParseSetCodeFromCardListJSON(lastBody, licenceCode)
}

var ClassicProductsConfig = internal.ProductsConfig{
	GetListingPage:     classicProductsGetListingPage,
	GetDetailPage:      classicProductsGetDetailPage,
	LoopProducts:       classicProductsLoop,
	ExtractListing:     classicExtractListing,
	ExtractDetail:      classicExtractDetail,
	GetSetCodeFallback: classicGetSetCodeFallback,
}

var productsMaxPages int

var productsCmd = &cobra.Command{
	Use:   "products",
	Short: "Scrape Classic WS product catalog from ws-tcg.com/products/",
	Run: func(cmd *cobra.Command, args []string) {
		internal.ScrapeProducts(&ClassicProductsConfig, productsMaxPages)
	},
}

func init() {
	classicCmd.AddCommand(productsCmd)
	productsCmd.Flags().IntVarP(&productsMaxPages, "pages", "p", 0, "number of listing pages to fetch (0 = all)")
}
