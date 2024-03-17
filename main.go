package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var (
	materials                            = []string{"art card 350gsm", "art card 300gsm", "art card 260gsm", "boxboard 350gsm", ""}
	categorySize                         = []string{"A1+", "A1", "½A1 (RR)", "⅓A2++ (rr)", "A2++", ""}
	quantityRange                        = []int{100, 200, 300, 400, 500, 1000, 1500, 2000, 3000, 4000, 5000}
	surfaceProtectionPrinting            = []string{"no finishing (may cause colour rubbing issue)", "water base normal 1 side", "water base food grade 1 side", "uv varnish 1side"}
	windowHoleWithoutTransparentPVCSheet = []string{"within 3mm to 50mm", "within 90mm x 54mm", "within 148mm x 105mm", "within 210mm x 148mm", "within 222mm x 190mm", "within 297mm x 210mm", "within 300mm x 297mm", "within 420mm x 297mm"}
	windowHoleWithTransparentPVCSheet    = []string{"within 45mm x 45mm", "within 90mm x 54mm", "within 90mm x 90mm", "within 148mm x 210mm", "within 297mm x 210mm"}
	hotstamping                          = []string{"within 16 square inch", "within 24 square inch", "within 32 square inch"}
	emboss_deboss                        = []string{"within 16 square inch", "within 24 square inch", "within 32 square inch"}
	stringFinishing                      = []string{"12inch", "14inch", "16inch", "18inch", "20inch", "22inch", "24inch", "26inch", "28inch", "30inch"}
)

type Quotation struct {
	SizeCategory string   `json:"sizeCategory"`
	Quantity     []int    `json:"quantity"`
	Material     string   `json:"material"`
	NoOfColours  int      `json:"noOfColours"`
	ReadiedSize  bool     `json:"readiedSize"`
	IsDoubleSide bool     `json:"isDoubleSide"`
	AddOns       []string `json:"addOns"`
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

func calcuateQuotation(quotation Quotation, value [][]interface{}) (Pricing, error) {
	var search_str_material string
	for i := 0; i < len(materials); i++ {
		if materials[i] == quotation.Material {
			search_str_material = quotation.Material
			break
		}
	}
	var colourSearchStr string
	if quotation.NoOfColours <= 1 {
		colourSearchStr = fmt.Sprintf("%d%s", quotation.NoOfColours, "colour")
	} else {
		colourSearchStr = fmt.Sprintf("%d%s", quotation.NoOfColours, "colours")
	}
	noOfQuantity := len(quotation.Quantity)
	// search_str_material = strings.Replace(search_str_material, "_", " ", -1)
	prices := Pricing{
		Quantity:     quotation.Quantity[0],
		NoOfColours:  quotation.NoOfColours,
		ReadiedSize:  quotation.ReadiedSize,
		PriceLabel:   search_str_material,
		Price:        []string{},
		Header:       fmt.Sprintf("Quotation for %s, %s \n", search_str_material, quotation.SizeCategory),
		IsDoubleSide: quotation.IsDoubleSide,
		SizeCategory: quotation.SizeCategory,
	}
	tempQuantitySearchIdx := 0
	for _, row := range value {
		if len(row) == 6 {
			quantity_search := strconv.Itoa(quotation.Quantity[tempQuantitySearchIdx])
			if row[2] == search_str_material && row[3] == quotation.SizeCategory && row[1] == colourSearchStr && row[4] == quantity_search {
				if row[5] == "" || row[5] == "not available" {
					row[5] = " Not Available"
				}
				prices.Price = append(prices.Price, fmt.Sprintf("%s pcs: RM%s Printing", row[4], row[5]))
				tempQuantitySearchIdx++
				if len(prices.Price) == noOfQuantity {
					break
				}
			}
		}
	}
	return prices, nil
}

func calculateAddOn(price Pricing, addOns []string, value [][]interface{}, quotation Quotation) (Pricing, error) {
	for _, addOn := range addOns {
		price.PriceLabel += fmt.Sprintf(" + %s", addOn)
		tempQuantitySearchIdx := 0
		if addOn == "no finishing (may cause colour rubbing issue)" {
			for i := 0; i < len(price.Price); i++ {
				price.Price[i] += " + Not Available (no finishing)"
			}
			continue
		}
		// If strings contains _ mean is secondary finishing
		if strings.Contains(addOn, "_") {
			//split string by dash
			addOnType := strings.Split(addOn, "_")[0]
			spec := strings.Split(addOn, "_")[1]
			for _, row := range value {
				if len(row) != 1 && len(row) != 0 {
					quantity_search := strconv.Itoa(quotation.Quantity[tempQuantitySearchIdx])
					if row[0] == addOnType && row[1] == spec && row[2] == quantity_search {
						if quotation.IsDoubleSide {
							if len(row) != 5 {
								price.Price[tempQuantitySearchIdx] += fmt.Sprintf(" + Not Available (%s)", addOnType)
							} else {
								if row[4] == "" || row[4] == "not available" {
									row[4] = fmt.Sprintf(" + Not Available (%s)", addOnType)
								}
								price.Price[tempQuantitySearchIdx] += fmt.Sprintf(" + RM %s (%s)", row[4], addOnType)
							}

						} else {
							if row[3] == "" {
								row[3] = "Not Available"
							}
							price.Price[tempQuantitySearchIdx] += fmt.Sprintf(" + RM%s", row[3])
						}
						if tempQuantitySearchIdx == len(quotation.Quantity)-1 {
							break
						}
						tempQuantitySearchIdx++
					}
				}
			}
			continue
		} else {
			for _, row := range value {
				if len(row) != 1 && len(row) != 0 {
					quantity_search := strconv.Itoa(quotation.Quantity[tempQuantitySearchIdx])
					if row[0] == addOn && row[1] == price.SizeCategory && row[2] == quantity_search {
						if quotation.IsDoubleSide {
							if row[4] == "" {
								row[4] = fmt.Sprintf(" + Not Available (%s)", addOn)
							}
							price.Price[tempQuantitySearchIdx] += fmt.Sprintf(" + RM%s (%s)", row[4], addOn)
						} else {
							if row[3] == "" {
								row[3] = fmt.Sprintf(" + Not Available (%s)", addOn)
							}
							price.Price[tempQuantitySearchIdx] += fmt.Sprintf(" + RM%s (%s)", row[3], addOn)
						}
						if tempQuantitySearchIdx == len(quotation.Quantity)-1 {
							break
						}
						tempQuantitySearchIdx++
					}
				}
			}
		}
	}
	return price, nil
}

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

	app.Get("/", func(c *fiber.Ctx) error {
		// Render index template
		return c.Render("index", fiber.Map{
			"materials":                            materials,
			"categorySize":                         categorySize,
			"quantityRange":                        quantityRange,
			"surfaceProtectionPrinting":            surfaceProtectionPrinting,
			"windowHoleWithoutTransparentPVCSheet": windowHoleWithoutTransparentPVCSheet,
			"hotstamping":                          hotstamping,
			"embossDeboss":                         emboss_deboss,
			"stringFinishing":                      stringFinishing,
		})
	})

	app.Post("/getQuotation", func(c *fiber.Ctx) error {
		quotation := new(Quotation)
		if err := c.BodyParser(quotation); err != nil {
			return err
		}
		var value [][]interface{}
		if quotation.IsDoubleSide {
			value, err = getValueFromGoogleSheet(srv, spreadsheetId, "double sides print raw")
			if err != nil {
				return err
			}
		} else {
			value, err = getValueFromGoogleSheet(srv, spreadsheetId, "Printing Raw")
			if err != nil {
				return err
			}
		}
		prices, err := calcuateQuotation(*quotation, value)
		if err != nil {
			return err
		}
		value, err = getValueFromGoogleSheet(srv, spreadsheetId, "finishing raw (imported)!A1:F")
		if err != nil {
			return err
		}
		prices, err = calculateAddOn(prices, quotation.AddOns, value, *quotation)
		if err != nil {
			return err
		}
		prices, err = categorizeMachineType(prices, *quotation)
		if err != nil {
			return err
		}
		return c.JSON(prices)
	})

	log.Fatal(app.Listen(":8000"))

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
