package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"wsallcli/internal"

	"github.com/spf13/cobra"
)

type setInfo struct {
	LicenceCode string
	SetCode     string
}

var curlyQuoteReplacer = strings.NewReplacer(
	"’", "'",
	"‘", "'",
	"“", "\"",
	"”", "\"",
)

func normalizeTitle(s string) string {
	return curlyQuoteReplacer.Replace(s)
}

func filterSetCodes(infos []setInfo, keep func(string) bool) []setInfo {
	var out []setInfo
	for _, info := range infos {
		if keep(info.SetCode) {
			out = append(out, info)
		}
	}
	return out
}

func buildWsoffdataIndex(root string) (titleIndex map[string][]setInfo, licenceIndex map[string][]setInfo, err error) {
	titleIndex = map[string][]setInfo{}
	licenceIndex = map[string][]setInfo{}

	licenceDirs, err := os.ReadDir(root)
	if err != nil {
		return nil, nil, fmt.Errorf("read wsoffdata root: %w", err)
	}

	for _, licenceDir := range licenceDirs {
		if !licenceDir.IsDir() {
			continue
		}
		licenceCode := licenceDir.Name()
		setCodeDirs, err := os.ReadDir(filepath.Join(root, licenceCode))
		if err != nil {
			slog.Warn("could not read licence dir", "licenceCode", licenceCode, "err", err)
			continue
		}

		for _, setCodeDir := range setCodeDirs {
			if !setCodeDir.IsDir() {
				continue
			}
			setCode := setCodeDir.Name()
			dirPath := filepath.Join(root, licenceCode, setCode)

			entries, err := os.ReadDir(dirPath)
			if err != nil {
				slog.Warn("could not read setCode dir", "path", dirPath, "err", err)
				continue
			}

			info := setInfo{LicenceCode: licenceCode, SetCode: setCode}
			licenceIndex[licenceCode] = append(licenceIndex[licenceCode], info)

			// Read the first JSON file to get the setName
			for _, entry := range entries {
				if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
					continue
				}
				data, err := os.ReadFile(filepath.Join(dirPath, entry.Name()))
				if err != nil {
					break
				}
				var card struct {
					SetName string `json:"setName"`
				}
				if err := json.Unmarshal(data, &card); err != nil || card.SetName == "" {
					break
				}
				key := normalizeTitle(card.SetName)
				titleIndex[key] = append(titleIndex[key], info)
				break
			}
		}
	}

	return titleIndex, licenceIndex, nil
}

func enrichProducts(products []internal.Product, titleIndex map[string][]setInfo) {
	enrichProductsFull(products, titleIndex, nil)
}

func enrichProductsFull(products []internal.Product, titleIndex map[string][]setInfo, licenceIndex map[string][]setInfo) {
	for i := range products {
		p := &products[i]
		if p.SetCode != "" {
			continue
		}

		normalizedTitle := normalizeTitle(p.Title)
		infos := titleIndex[normalizedTitle]

		if len(infos) == 0 && p.ProductType != "" {
			infos = titleIndex[p.ProductType+" "+normalizedTitle]
		}

		if len(infos) == 0 {
			// Fallback: match by licence code
			if p.LicenceCode != "" && licenceIndex != nil {
				infos = licenceIndex[p.LicenceCode]
			}
			if len(infos) == 0 {
				slog.Warn("no wsoffdata match", "title", p.Title)
				continue
			}
		}

		switch p.ProductType {
		case "ブースターパック":
			// SE/WE sets are エクストラ/プレミアムブースター — exclude them
			if filtered := filterSetCodes(infos, func(code string) bool {
				return !strings.HasPrefix(code, "SE") && !strings.HasPrefix(code, "WE")
			}); len(filtered) > 0 {
				infos = filtered
			}
		case "プレミアムブースター", "エクストラブースター":
			// Only SE/WE sets are valid for these product types
			if filtered := filterSetCodes(infos, func(code string) bool {
				return strings.HasPrefix(code, "SE") || strings.HasPrefix(code, "WE")
			}); len(filtered) > 0 {
				infos = filtered
			} else {
				// No valid candidate — skip rather than assign a wrong set
				slog.Warn("no SE/WE candidate for premium/extra booster", "title", p.Title)
				continue
			}
		}

		if len(infos) > 1 {
			// Resolve if all candidates share the same SetCode
			shared := infos[0].SetCode
			for _, info := range infos[1:] {
				if info.SetCode != shared {
					shared = ""
					break
				}
			}
			if shared == "" {
				slog.Warn("ambiguous wsoffdata match — skipping", "title", p.Title, "candidates", infos)
				continue
			}
			// All share same SetCode — use it, leave LicenceCode alone
			p.SetCode = shared
			slog.Info("enriched product (shared setCode)", "title", p.Title, "setCode", p.SetCode)
			continue
		}

		p.SetCode = infos[0].SetCode
		if p.LicenceCode == "" {
			p.LicenceCode = infos[0].LicenceCode
		}
		slog.Info("enriched product", "title", p.Title, "setCode", p.SetCode, "licenceCode", p.LicenceCode)
	}
}

var enrichCmd = &cobra.Command{
	Use:   "enrich <wsoffdata-path>",
	Short: "Patch missing SetCode in products.json using a local wsoffdata clone",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		wsoffdataPath := args[0]

		data, err := os.ReadFile("products.json")
		if err != nil {
			slog.Error("read products.json", "err", err)
			os.Exit(1)
		}
		var products []internal.Product
		if err := json.Unmarshal(data, &products); err != nil {
			slog.Error("parse products.json", "err", err)
			os.Exit(1)
		}

		titleIndex, licenceIndex, err := buildWsoffdataIndex(wsoffdataPath)
		if err != nil {
			slog.Error("build wsoffdata index", "err", err)
			os.Exit(1)
		}
		slog.Info("built index", "sets", len(titleIndex))

		enrichProductsFull(products, titleIndex, licenceIndex)

		res, _ := json.Marshal(products)
		var buf bytes.Buffer
		json.Indent(&buf, res, "", "\t")
		if err := os.WriteFile("products.json", buf.Bytes(), 0o644); err != nil {
			slog.Error("write products.json", "err", err)
			os.Exit(1)
		}
		slog.Info("wrote products.json")
	},
}

func init() {
	productsCmd.AddCommand(enrichCmd)
}
