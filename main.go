package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var (
	// Printing
	// If readied size is true, there will be discount
	materials     = []string{"art card 350gsm", "art card 300gsm", "art card 260gsm", "boxboard 350gsm", "boxboard 300gsm", "boxboard 260gsm", "carton box e flute wrapped by art card 250gsm : ac250/bt115(f)/bt150", "carton box e flute wrapped by boxboard 250gsm : bb250/bt115(f)/bt150", "carton box e flute wrapped by brown testliner 180gsm : bt180/bt125(f)/bt150", "carton box e flute wrapped by white testliner 175gsm : wt175/bt125(f)/bt150", "white coated kraft 350gsm", "white coated kraft 300gsm"}
	categorySize  = []string{"A2", "A3+", "A3", "A4+", "A4", "A5", "A5+"}
	quantityRange = []int{100, 200, 300, 400, 500, 1000, 1500, 2000}
	// Primary Addons
	surfaceProtectionPrinting = []string{"no finishing (may cause colour rubbing issue)", "water base normal 1side", "water base food grade 1side", "uv varnish 1side", "spot uv 1side", "gloss lam 1side", "matt lam 1side"}
	// Secondary Addons
	windowHoleWithoutTransparentPVCSheet = []string{"none", "within 3mm to 50mm", "within 90mm x 54mm", "within 148mm x 105mm", "within 210mm x 148mm", "within 222mm x 190mm", "within 297mm x 210mm", "within 300mm x 297mm", "within 420mm x 297mm"}
	windowHoleWithTransparentPVCSheet    = []string{"none", "within 45mm x 45mm", "within 90mm x 54mm", "within 90mm x 90mm", "within 148mm x 210mm", "within 297mm x 210mm"}
	hotstamping                          = []string{"none", "within 16 square inch", "within 24 square inch", "within 32 square inch"}
	emboss_deboss                        = []string{"none", "within 16 square inch", "within 24 square inch", "within 32 square inch"}
	stringFinishing                      = []string{"none", "12inch", "14inch", "16inch", "18inch", "20inch", "22inch", "24inch", "26inch", "28inch", "30inch"}
	// Third Addons
	// Double sides printing
	finishingAnotherSide = []string{"no finishing (may cause colour rubbing issue)", "uv varnish 1side", "spot uv 1side", "water base normal 1side", "water base food grade 1side", "matt lam 1side", "gloss lam 1side"}
	noOfColours          = []string{"0colour", "1colour", "4colours"}
	isReadiedSize        = []string{"custom", "readied"}
)

type ThirdAddOns struct {
	IsDoubleSide         bool   `json:"isDoubleSide"`
	FinishingAnotherSide string `json:"finishingAnotherSide"`
}

type SecondaryAddOns struct {
	SpotUV1Side                          string `json:"spotUV1Side"`
	WindowHoleWithoutTransparentPVCSheet string `json:"windowHoleWithoutTransparentPVCSheet"`
	WindowHoleWithTransparentPVCSheet    string `json:"windowHoleWithTransparentPVCSheet"`
	Hotstamping                          string `json:"hotstamping"`
	EmbossDeboss                         string `json:"embossDeboss"`
	String                               string `json:"string"`
}

type PrimaryAddOns struct {
	SurfaceProtectionPrinting string `json:"surfaceProtectionPrinting"`
}

type Quotation struct {
	SizeCategory    string          `json:"sizeCategory"`
	Quantity        []int           `json:"quantity"`
	Material        string          `json:"material"`
	NoOfColours     string          `json:"noOfColours"`
	ReadiedSize     bool            `json:"readiedSize"`
	PrimaryAddOns   PrimaryAddOns   `json:"primaryAddOns"`
	SecondaryAddOns SecondaryAddOns `json:"secondaryAddOns"`
	ThirdAddOns     ThirdAddOns     `json:"thirdAddOns"`
}

type Pricing struct {
	Quantity     int      `json:"quantity"`
	Price        []string `json:"price"`
	PriceLabel   string   `json:"priceLabel"`
	ReadiedSize  bool     `json:"readiedSize"`
	NoOfColours  int      `json:"noOfColours"`
	IsDoubleSide bool     `json:"isDoubleSide"`
	Header       string   `json:"header"`
	SizeCategory string   `json:"sizeCategory"`
}

// Instead of using []string, use hashmap to store the value
// func (q *Quotation) getPrintingCost(srv *sheets.Service, spreadsheetId string, range_ string) (string, []string, error) {
func (q *Quotation) getPrintingCost(srv *sheets.Service, spreadsheetId string, range_ string) (string, map[string]string, error) {
	// Go to google sheet and get the printing cost
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, range_).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}
	var search_str_noOfColours string = q.NoOfColours
	var search_str_material string = q.Material
	var search_str_sizeCategory string = q.SizeCategory

	var quotationStringTemplate string = "<Header>\n"
	var priceMap = make(map[string]string)
	for _, quantity := range q.Quantity {
		for _, row := range resp.Values {
			if len(row) == 6 {
				quantity_string := strconv.Itoa(quantity)
				if err != nil {
					fmt.Println("Error converting int to string \n", err)
				}
				if row[1] == search_str_noOfColours && row[2] == search_str_material && row[4] == quantity_string && row[3] == search_str_sizeCategory {
					temp := fmt.Sprintf("📌 *%s pcs*: RM%s Printing <Primary%s><Secondary%s><Third%s><ReadiedSizeDiscount%s><Total%s>\n", row[4], row[5], strconv.Itoa(quantity), strconv.Itoa(quantity), strconv.Itoa(quantity), strconv.Itoa(quantity), strconv.Itoa(quantity))
					quotationStringTemplate += temp
					priceMap[quantity_string] = row[5].(string)
				}
			}
		}
	}

	return quotationStringTemplate, priceMap, nil
}

func (q *Quotation) addTotalPriceInString(priceMap map[string]string, quantity string, value string) map[string]string {
	// convert priceMap value to float
	if value == "not available" {
		priceMap[quantity] = "not available"
		return priceMap
	}
	if priceMap[quantity] == "not available" {
		return priceMap
	}
	price, err := strconv.ParseFloat(priceMap[quantity], 64)
	if err != nil {
		fmt.Println("Error converting string to float \n", err)
	}
	// convert value to float
	value_float, err := strconv.ParseFloat(value, 64)
	if err != nil {
		fmt.Println("Error converting string to float \n", err)
	}
	// add the value to the price
	price += value_float
	// convert price back to string
	priceMap[quantity] = fmt.Sprintf("%.2f", price)
	return priceMap
}

func (q *Quotation) getPrimaryAddOn(srv *sheets.Service, spreadsheetId string, range_ string, quotationStringTemplate string, priceMap map[string]string) (string, map[string]string, error) {
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, range_).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}
	var search_string_sizeCategory string = q.SizeCategory
	var search_string_finishing string = q.PrimaryAddOns.SurfaceProtectionPrinting // This one return empty

	for _, quantity := range q.Quantity {
		for _, row := range resp.Values {
			if len(row) >= 4 {
				quantity_string := strconv.Itoa(quantity)
				if err != nil {
					fmt.Println("Error converting int to string \n", err)
				}
				if row[0] == search_string_finishing && row[1] == search_string_sizeCategory && row[2] == quantity_string {
					quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Primary%s>", quantity_string), fmt.Sprintf("+ RM%s %s", row[3], search_string_finishing), -1)
					priceMap = q.addTotalPriceInString(priceMap, quantity_string, row[3].(string))
				}
			}
		}
	}

	return quotationStringTemplate, priceMap, nil
}

func checkSecondaryAddOnMatch(row []interface{}, search_string_sizeCategory string, quantity_string string, targetSearchString string) (string, bool) {
	if row[1] == search_string_sizeCategory && row[2] == quantity_string && row[0] == targetSearchString {
		if row[3] == "not available" {
			return targetSearchString, true
		}
		return targetSearchString, true
	}
	return "", false
}

func (q *Quotation) getSecondaryAddOn(srv *sheets.Service, spreadsheetId string, range_ string, quotationStringTemplate string, priceMap map[string]string) (string, map[string]string, error) {
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, range_).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}
	var search_string_sizeCategory string = q.SizeCategory
	var search_string_windowHoleWithoutTransparentPVCSheet string = q.SecondaryAddOns.WindowHoleWithoutTransparentPVCSheet
	var search_string_hotstamping string = q.SecondaryAddOns.Hotstamping
	var search_string_emboss_deboss string = q.SecondaryAddOns.EmbossDeboss
	var search_string_string string = q.SecondaryAddOns.String
	var search_string_windowHoleWithTransparentPVCSheet string = q.SecondaryAddOns.WindowHoleWithTransparentPVCSheet
	// var search_string_spotUV1Side string = q.SecondaryAddOns.SpotUV1Side

	for _, quantity := range q.Quantity {
		for _, row := range resp.Values {
			if len(row) <= 5 && len(row) >= 4 {
				quantity_string := strconv.Itoa(quantity)
				if err != nil {
					fmt.Println("Error converting int to string \n", err)
				}

				windowHoleWithoutTransparentPVCSheet, ok := checkSecondaryAddOnMatch(row, search_string_windowHoleWithoutTransparentPVCSheet, quantity_string, "window hole without transparent pvc sheet")
				if ok {
					quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Secondary%s>", quantity_string), fmt.Sprintf(" + RM%s %s<Secondary%s>", row[3], windowHoleWithoutTransparentPVCSheet, quantity_string), -1)
					priceMap = q.addTotalPriceInString(priceMap, quantity_string, row[3].(string))
				}
				hotstamping, ok := checkSecondaryAddOnMatch(row, search_string_hotstamping, quantity_string, "hot stamping")
				if ok {
					quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Secondary%s>", quantity_string), fmt.Sprintf(" + RM%s %s<Secondary%s>", row[3], hotstamping, quantity_string), -1)
					priceMap = q.addTotalPriceInString(priceMap, quantity_string, row[3].(string))
				}
				emboss_deboss, ok := checkSecondaryAddOnMatch(row, search_string_emboss_deboss, quantity_string, "emboss / deboss")
				if ok {
					quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Secondary%s>", quantity_string), fmt.Sprintf(" + RM%s %s<Secondary%s>", row[3], emboss_deboss, quantity_string), -1)
					priceMap = q.addTotalPriceInString(priceMap, quantity_string, row[3].(string))
				}
				stringFinishing, ok := checkSecondaryAddOnMatch(row, search_string_string, quantity_string, "string")
				if ok {
					quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Secondary%s>", quantity_string), fmt.Sprintf(" + RM%s %s<Secondary%s>", row[3], stringFinishing, quantity_string), -1)
					priceMap = q.addTotalPriceInString(priceMap, quantity_string, row[3].(string))
				}
				windowHoleWithTransparentPVCSheet, ok := checkSecondaryAddOnMatch(row, search_string_windowHoleWithTransparentPVCSheet, quantity_string, "window hole with transparent pvc sheet")
				if ok {
					quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Secondary%s>", quantity_string), fmt.Sprintf(" + RM%s %s<Secondary%s>", row[3], windowHoleWithTransparentPVCSheet, quantity_string), -1)
					priceMap = q.addTotalPriceInString(priceMap, quantity_string, row[3].(string))
				}
				if q.SecondaryAddOns.SpotUV1Side == "spotUV1side" {
					spotUV1Side, ok := checkSecondaryAddOnMatch(row, search_string_sizeCategory, quantity_string, "spot uv 1side")
					if ok {
						if q.ThirdAddOns.IsDoubleSide {
							quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Secondary%s>", quantity_string), fmt.Sprintf(" + RM%s %s<Secondary%s>", row[4], spotUV1Side, quantity_string), -1)
							priceMap = q.addTotalPriceInString(priceMap, quantity_string, row[4].(string))

						} else {
							quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Secondary%s>", quantity_string), fmt.Sprintf(" + RM%s %s<Secondary%s>", row[3], spotUV1Side, quantity_string), -1)
							priceMap = q.addTotalPriceInString(priceMap, quantity_string, row[3].(string))
						}
					}
				}
			}
		}
	}

	return quotationStringTemplate, priceMap, nil
}

func (q *Quotation) getThirdAddOnPrinting(srv *sheets.Service, spreadsheetId string, range_ string, quotationStringTemplate string, priceMap map[string]string) (string, map[string]string, error) {
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, range_).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	// Add Cost for another side, if double side printing
	var search_string_sizeCategory string = q.SizeCategory
	var search_str_noOfColours string = q.NoOfColours
	var search_str_material string = q.Material
	if q.ThirdAddOns.IsDoubleSide && q.NoOfColours != "0colour" {
		for _, quantity := range q.Quantity {
			for _, row := range resp.Values {
				if len(row) == 6 {
					quantity_string := strconv.Itoa(quantity)
					if err != nil {
						fmt.Println("Error converting int to string \n", err)
					}
					if row[1] == search_str_noOfColours && row[2] == search_str_material && row[3] == search_string_sizeCategory && row[4] == quantity_string {
						quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Third%s>", quantity_string), fmt.Sprintf(" + RM%s %s<Third%s>", row[5], "printing another side", quantity_string), -1)
						priceMap = q.addTotalPriceInString(priceMap, quantity_string, row[5].(string))
					}
				}
			}
		}
	}
	return quotationStringTemplate, priceMap, nil
}

func (q *Quotation) getThirdAddOnFinishing(srv *sheets.Service, spreadsheetId string, range_ string, quotationStringTemplate string, priceMap map[string]string) (string, map[string]string, error) {
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, range_).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}
	var search_string_sizeCategory string = q.SizeCategory
	var search_string_finishing string = q.ThirdAddOns.FinishingAnotherSide
	if q.ThirdAddOns.IsDoubleSide && q.NoOfColours != "0colour" {
		for _, quantity := range q.Quantity {
			for _, row := range resp.Values {
				if len(row) == 5 {
					quantity_string := strconv.Itoa(quantity)
					if err != nil {
						fmt.Println("Error converting int to string \n", err)
					}

					if row[0] == search_string_finishing && row[1] == search_string_sizeCategory && row[2] == quantity_string {
						quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Third%s>", quantity_string), fmt.Sprintf(" + RM%s %s", row[4], search_string_finishing), -1)
						priceMap = q.addTotalPriceInString(priceMap, quantity_string, row[4].(string))
					}
				}
			}
		}
	}

	return quotationStringTemplate, priceMap, nil
}

func (q *Quotation) provideDiscountForReadiedSize(quotationStringTemplate string, priceMap map[string]string) (string, map[string]string, error) {
	var discountMap = make(map[string]float64)
	discountMap["A1"] = 300
	discountMap["A2"] = 250
	discountMap["A3"] = 200
	discountMap["A4"] = 150
	discountMap["A5"] = 100

	if q.ReadiedSize {
		for _, quantity := range q.Quantity {
			quantity_string := strconv.Itoa(quantity)
			// price, err := strconv.ParseFloat(priceMap[quantity_string], 64)
			// if err != nil {
			// 	fmt.Println("Error converting string to float \n", err)
			// }
			if q.SizeCategory == "A1" {
				// price = price - discountMap["A1"]
				quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<ReadiedSizeDiscount%s>", quantity_string), fmt.Sprintf(" - %.2f Readied Size", discountMap["A1"]), -1)
			} else if q.SizeCategory == "A2" {
				// price = price - discountMap["A2"]
				quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<ReadiedSizeDiscount%s>", quantity_string), fmt.Sprintf(" - %.2f Readied Size", discountMap["A2"]), -1)
			} else if q.SizeCategory == "A3" {
				// price = price - discountMap["A3"]
				quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<ReadiedSizeDiscount%s>", quantity_string), fmt.Sprintf(" - %.2f Readied Size", discountMap["A3"]), -1)
			} else if q.SizeCategory == "A4" {
				// price = price - discountMap["A4"]
				quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<ReadiedSizeDiscount%s>", quantity_string), fmt.Sprintf(" - %.2f Readied Size", discountMap["A4"]), -1)
			} else if q.SizeCategory == "A5" {
				// price = price - discountMap["A5"]
				quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<ReadiedSizeDiscount%s>", quantity_string), fmt.Sprintf(" - %.2f Readied Size", discountMap["A5"]), -1)
			}
			// fmt.Println("discount given", price)
			priceMap = q.addTotalPriceInString(priceMap, quantity_string, fmt.Sprintf("-%.2f", discountMap[q.SizeCategory]))
		}
	} else {
		for _, quantity := range q.Quantity {
			quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<ReadiedSizeDiscount%s>", strconv.Itoa(quantity)), "", -1)
		}
	}
	return quotationStringTemplate, priceMap, nil
}

func (q *Quotation) addTotalToTemplate(quotationStringTemplate string, priceMap map[string]string) string {
	priceMapKeys := make([]string, 0, len(priceMap))
	for k := range priceMap {
		priceMapKeys = append(priceMapKeys, k)
	}
	sort.Strings(priceMapKeys)
	for _, key := range priceMapKeys {
		quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Total%s>", key), fmt.Sprintf(" = RM%s\n", priceMap[key]), -1)
	}
	return quotationStringTemplate
}

func (q *Quotation) addHeaderRemoveTemplate(quotationStringTemplate string) string {
	var readiedCustomedSizeDisplay string = ""
	if q.ReadiedSize {
		readiedCustomedSizeDisplay = "Readied Size Shape"
	} else {
		readiedCustomedSizeDisplay = "Customed Size Shape"
	}
	var singleDoubleSiteDisplay string = ""
	if q.ThirdAddOns.IsDoubleSide {
		singleDoubleSiteDisplay = "Double Side Printing"
	} else {
		singleDoubleSiteDisplay = "Single Side Printing"
	}
	var printingAddons string = q.Material
	if q.PrimaryAddOns.SurfaceProtectionPrinting != "no finishing (may cause colour rubbing issue)" {
		printingAddons += fmt.Sprintf(" + %s", q.PrimaryAddOns.SurfaceProtectionPrinting)
	}
	if q.SecondaryAddOns.WindowHoleWithoutTransparentPVCSheet != "none" {
		// Add key for window hole without transparent pvc sheet
		printingAddons += " + window hole without transparent pvc sheet " + q.SecondaryAddOns.WindowHoleWithoutTransparentPVCSheet
	}
	if q.SecondaryAddOns.WindowHoleWithTransparentPVCSheet != "none" {
		printingAddons += " + window hole with transparent pvc sheet " + q.SecondaryAddOns.WindowHoleWithTransparentPVCSheet
	}
	if q.SecondaryAddOns.Hotstamping != "none" {
		printingAddons += " + hot stamping " + q.SecondaryAddOns.Hotstamping
	}
	if q.SecondaryAddOns.EmbossDeboss != "none" {
		printingAddons += " + emboss / deboss " + q.SecondaryAddOns.EmbossDeboss
	}
	if q.SecondaryAddOns.String != "none" {
		printingAddons += " + string " + q.SecondaryAddOns.String
	}
	if q.SecondaryAddOns.SpotUV1Side != "none" {
		printingAddons += " + spot uv 1side"
	}
	if q.ThirdAddOns.IsDoubleSide && q.NoOfColours != "0colour" {
		printingAddons += fmt.Sprintf(" + %s ", "printing another side")
		if q.ThirdAddOns.FinishingAnotherSide != "no finishing (may cause colour rubbing issue)" {
			printingAddons += fmt.Sprintf(" + %s %s", q.ThirdAddOns.FinishingAnotherSide, "another side")
		}
	}
	var colourDisplay string = ""
	if q.NoOfColours == "0colour" {
		colourDisplay = "no printing"
	} else if q.NoOfColours == "1colour" {
		colourDisplay = "1colour printing (cyan / magenta / yellow / black)"
	} else {
		colourDisplay = "colourful printing CMYK"
	}
	var machineDisplay string = ""
	if q.Quantity[0] <= 500 && q.Quantity[len(q.Quantity)-1] <= 500 {
		machineDisplay = "digital offset / litho offset"
	} else if q.Quantity[0] < 500 && q.Quantity[len(q.Quantity)-1] >= 500 {
		machineDisplay = "litho offset (1000pcs & above), digital offset (10-500pcs)"
	} else if q.Quantity[len(q.Quantity)-1] > 500 {
		machineDisplay = "litho offset"
	} else {
		machineDisplay = "Logic Error"
	}

	header := fmt.Sprintf(`*QUOTATION : BOX PRINTING : %s *
product : box
machine : %s
shape : shape such like open lid / hinged / cake / top bottom / drawer & etc
material : (see below)
finishing : die cut / die cut + gluing
quantity : (see below) For quantity more than 2000pcs, please whatsapp us +60163443238 to request quotation due to paper price fluctuation issue. 
print side : %s (%s x %s)
colour : %s
print process : 6-7days (art card) / 8-10days (with beautify finishing) / 13-14days (carton material) excluded sat, sun, public holiday & pre-preparation works


estimated price :
[ %s ] 
%s
`, q.SizeCategory, machineDisplay, singleDoubleSiteDisplay, q.NoOfColours, q.NoOfColours, colourDisplay, readiedCustomedSizeDisplay, printingAddons)
	quotationStringTemplate = strings.Replace(quotationStringTemplate, "<Header>", header, -1)
	// remove all the template
	for _, quantity := range q.Quantity {
		quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Secondary%s>", strconv.Itoa(quantity)), "", -1)
		quotationStringTemplate = strings.Replace(quotationStringTemplate, fmt.Sprintf("<Third%s>", strconv.Itoa(quantity)), "", -1)
	}
	return quotationStringTemplate
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func connectToGoogleSheet() (*sheets.Service, error) {
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	return srv, err
}

func load_env() error {
	err := godotenv.Load()
	return err
}

func getValueFromGoogleSheet(srv *sheets.Service, spreadsheetId string, range_ string) ([][]interface{}, error) {
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, range_).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}
	return resp.Values, err
}

// func calcuateQuotation(quotation Quotation, value [][]interface{}) (Pricing, error) {
// 	var search_str_material string
// 	for i := 0; i < len(materials); i++ {
// 		if materials[i] == quotation.Material {
// 			search_str_material = quotation.Material
// 			break
// 		}
// 	}
// 	var colourSearchStr string
// 	if quotation.NoOfColours <= 1 {
// 		colourSearchStr = fmt.Sprintf("%d%s", quotation.NoOfColours, "colour")
// 	} else {
// 		colourSearchStr = fmt.Sprintf("%d%s", quotation.NoOfColours, "colours")
// 	}
// 	noOfQuantity := len(quotation.Quantity)
// 	// search_str_material = strings.Replace(search_str_material, "_", " ", -1)
// 	prices := Pricing{
// 		Quantity:     quotation.Quantity[0],
// 		NoOfColours:  quotation.NoOfColours,
// 		ReadiedSize:  quotation.ReadiedSize,
// 		PriceLabel:   search_str_material,
// 		Price:        []string{},
// 		Header:       fmt.Sprintf("Quotation for %s, %s \n", search_str_material, quotation.SizeCategory),
// 		IsDoubleSide: quotation.IsDoubleSide,
// 		SizeCategory: quotation.SizeCategory,
// 	}
// 	tempQuantitySearchIdx := 0
// 	for _, row := range value {
// 		if len(row) == 6 {
// 			quantity_search := strconv.Itoa(quotation.Quantity[tempQuantitySearchIdx])
// 			if row[2] == search_str_material && row[3] == quotation.SizeCategory && row[1] == colourSearchStr && row[4] == quantity_search {
// 				if row[5] == "" || row[5] == "not available" {
// 					row[5] = " Not Available"
// 				}
// 				prices.Price = append(prices.Price, fmt.Sprintf("%s pcs: RM%s Printing", row[4], row[5]))
// 				tempQuantitySearchIdx++
// 				if len(prices.Price) == noOfQuantity {
// 					break
// 				}
// 			}
// 		}
// 	}
// 	return prices, nil
// }

// func calculateAddOn(price Pricing, addOns []string, value [][]interface{}, quotation Quotation) (Pricing, error) {
// 	for _, addOn := range addOns {
// 		price.PriceLabel += fmt.Sprintf(" + %s", addOn)
// 		tempQuantitySearchIdx := 0
// 		if addOn == "no finishing (may cause colour rubbing issue)" {
// 			for i := 0; i < len(price.Price); i++ {
// 				price.Price[i] += " + Not Available (no finishing)"
// 			}
// 			continue
// 		}
// 		// If strings contains _ mean is secondary finishing
// 		if strings.Contains(addOn, "_") {
// 			//split string by dash
// 			addOnType := strings.Split(addOn, "_")[0]
// 			spec := strings.Split(addOn, "_")[1]
// 			for _, row := range value {
// 				if len(row) != 1 && len(row) != 0 {
// 					quantity_search := strconv.Itoa(quotation.Quantity[tempQuantitySearchIdx])
// 					if row[0] == addOnType && row[1] == spec && row[2] == quantity_search {
// 						if quotation.IsDoubleSide {
// 							if len(row) != 5 {
// 								price.Price[tempQuantitySearchIdx] += fmt.Sprintf(" + Not Available (%s)", addOnType)
// 							} else {
// 								if row[4] == "" || row[4] == "not available" {
// 									row[4] = fmt.Sprintf(" + Not Available (%s)", addOnType)
// 								}
// 								price.Price[tempQuantitySearchIdx] += fmt.Sprintf(" + RM %s (%s)", row[4], addOnType)
// 							}

// 						} else {
// 							if row[3] == "" {
// 								row[3] = "Not Available"
// 							}
// 							price.Price[tempQuantitySearchIdx] += fmt.Sprintf(" + RM%s", row[3])
// 						}
// 						if tempQuantitySearchIdx == len(quotation.Quantity)-1 {
// 							break
// 						}
// 						tempQuantitySearchIdx++
// 					}
// 				}
// 			}
// 			continue
// 		} else {
// 			for _, row := range value {
// 				if len(row) != 1 && len(row) != 0 {
// 					quantity_search := strconv.Itoa(quotation.Quantity[tempQuantitySearchIdx])
// 					if row[0] == addOn && row[1] == price.SizeCategory && row[2] == quantity_search {
// 						if quotation.IsDoubleSide {
// 							if row[4] == "" {
// 								row[4] = fmt.Sprintf(" + Not Available (%s)", addOn)
// 							}
// 							price.Price[tempQuantitySearchIdx] += fmt.Sprintf(" + RM%s (%s)", row[4], addOn)
// 						} else {
// 							if row[3] == "" {
// 								row[3] = fmt.Sprintf(" + Not Available (%s)", addOn)
// 							}
// 							price.Price[tempQuantitySearchIdx] += fmt.Sprintf(" + RM%s (%s)", row[3], addOn)
// 						}
// 						if tempQuantitySearchIdx == len(quotation.Quantity)-1 {
// 							break
// 						}
// 						tempQuantitySearchIdx++
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return price, nil
// }

func main() {
	engine := html.New("./views", ".html")
	err := load_env()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	spreadsheetId := os.Getenv("SPREADSHEET_ID")
	srv, err := connectToGoogleSheet()
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}
	app := fiber.New(fiber.Config{
		Views: engine,
	})
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "POST",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
	app.Get("/", func(c *fiber.Ctx) error {
		// Render index template
		return c.Render("index", fiber.Map{
			"host":                                 os.Getenv("HOST"),
			"port":                                 os.Getenv("PORT"),
			"materials":                            materials,
			"categorySize":                         categorySize,
			"quantityRange":                        quantityRange,
			"surfaceProtectionPrinting":            surfaceProtectionPrinting,
			"windowHoleWithoutTransparentPVCSheet": windowHoleWithoutTransparentPVCSheet,
			"windowHoleWithTransparentPVCSheet":    windowHoleWithTransparentPVCSheet,
			"hotstamping":                          hotstamping,
			"embossDeboss":                         emboss_deboss,
			"stringFinishing":                      stringFinishing,
			"finishingAnotherSide":                 finishingAnotherSide,
			"noOfColours":                          noOfColours,
			"isReadiedSize":                        isReadiedSize,
		})
	})

	app.Post("/getQuotation", func(c *fiber.Ctx) error {
		quotation := new(Quotation)
		if err := c.BodyParser(quotation); err != nil {
			return err
		}
		fmt.Println("Quotation: ", quotation)
		quotationStringTemplate, priceMap, err := quotation.getPrintingCost(srv, spreadsheetId, "printing_raw")
		if err != nil {
			fmt.Println("Unable to get printing cost")
		}
		quotationStringTemplate, priceMap, err = quotation.getPrimaryAddOn(srv, spreadsheetId, "primary_secondary_addon_raw", quotationStringTemplate, priceMap)
		if err != nil {
			fmt.Println("Unable to get primary addon")
		}
		quotationStringTemplate, priceMap, err = quotation.getSecondaryAddOn(srv, spreadsheetId, "primary_secondary_addon_raw", quotationStringTemplate, priceMap)
		if err != nil {
			fmt.Println("Unable to get secondary addon")
		}
		quotationStringTemplate, priceMap, err = quotation.getThirdAddOnPrinting(srv, spreadsheetId, "third_addon_raw", quotationStringTemplate, priceMap)
		if err != nil {
			fmt.Println("Unable to get Third addon printing")
		}
		quotationStringTemplate, priceMap, err = quotation.getThirdAddOnFinishing(srv, spreadsheetId, "primary_secondary_addon_raw", quotationStringTemplate, priceMap)
		if err != nil {
			fmt.Println("Unable to get secondary addon finishing")
		}
		quotationStringTemplate, priceMap, err = quotation.provideDiscountForReadiedSize(quotationStringTemplate, priceMap)
		if err != nil {
			fmt.Println("Unable to provide discount for readied size")
		}
		quotationStringTemplate = quotation.addTotalToTemplate(quotationStringTemplate, priceMap)
		quotationStringTemplate = quotation.addHeaderRemoveTemplate(quotationStringTemplate)
		for key, value := range priceMap {
			fmt.Println("Key:", key, "Value:", value)
		}
		fmt.Println(quotationStringTemplate)
		return c.SendString(quotationStringTemplate)
	})

	log.Fatal(app.ListenTLS(":8000", "cert.pem", "key.pem"))

}

func categorizeMachineType(pricing Pricing, quotation Quotation) (Pricing, error) {
	if (quotation.Quantity[0] <= 500) && (quotation.Quantity[len(quotation.Quantity)-1] <= 500) {
		pricing.PriceLabel = "machine type: digital offset \n" + pricing.PriceLabel
		return pricing, nil
	}
	if (quotation.Quantity[0] > 500) && (quotation.Quantity[len(quotation.Quantity)-1] <= 5000) {
		pricing.PriceLabel = "machine type: litho offset \n" + pricing.PriceLabel
		return pricing, nil
	}
	if (quotation.Quantity[0] <= 500) && (quotation.Quantity[len(quotation.Quantity)-1] <= 5000) {
		pricing.PriceLabel = "machine type: litho offset (1000pcs & above), digital offset (10-500pcs) \n" + pricing.PriceLabel
		return pricing, nil
	}
	return pricing, errors.New("unable to categorize machine type")
}
