package main

import (
	"context"
	"encoding/json"
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
	materials     = []string{"art_card_350gsm", "art_card_300gsm", "art_card_260gsm", "boxboard_350gsm"}
	categorySize  = []string{"A1+", "A1", "½A1 (RR)", "⅓A2++ (rr)", "A2++"}
	quantityRange = []int{100, 200, 300, 400, 500, 1000, 1500, 2000, 3000, 4000, 5000}
)

type Quotation struct {
	SizeCategory string `json:"sizeCategory"`
	Quantity     []int  `json:"quantity"`
	Material     string `json:"material"`
	NoOfColours  int    `json:"noOfColours"`
	ReadiedSize  bool   `json:"readiedSize"`
}

type Pricing struct {
	Quantity    int      `json:"quantity"`
	Price       []string `json:"price"`
	PriceLabel  string   `json:"priceLabel"`
	ReadiedSize bool     `json:"readiedSize"`
	NoOfColours int      `json:"noOfColours"`
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
	search_str_material = strings.Replace(search_str_material, "_", " ", -1)
	prices := Pricing{
		Quantity:    quotation.Quantity[0],
		NoOfColours: quotation.NoOfColours,
		ReadiedSize: quotation.ReadiedSize,
		PriceLabel:  search_str_material,
		Price:       []string{},
	}
	tempQuantitySearchIdx := 0
	for _, row := range value {
		if len(row) == 6 {
			quantity_search := strconv.Itoa(quotation.Quantity[tempQuantitySearchIdx])
			if row[2] == search_str_material && row[3] == quotation.SizeCategory && row[1] == colourSearchStr && row[4] == quantity_search {
				if row[5] == "" || row[5] == "not available" {
					row[5] = "#N/A"
				}
				prices.Price = append(prices.Price, fmt.Sprintf("%s pcs: RM%s printing \n", row[4], row[5]))
				tempQuantitySearchIdx++
				if len(prices.Price) == noOfQuantity {
					break
				}
			}
		}
	}
	return prices, nil
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
			"materials":     materials,
			"categorySize":  categorySize,
			"quantityRange": quantityRange,
		})
	})

	app.Post("/getQuotation", func(c *fiber.Ctx) error {
		quotation := new(Quotation)
		if err := c.BodyParser(quotation); err != nil {
			return err
		}
		value, err := getValueFromGoogleSheet(srv, spreadsheetId, "Printing Raw!A1:F")
		if err != nil {
			return err
		}
		prices, _ := calcuateQuotation(*quotation, value)
		return c.JSON(prices)
	})

	log.Fatal(app.Listen(":8000"))

}
