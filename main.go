package main

import (
	"log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

type Quotation struct {
	SizeCategory string "json:sizeCategory"
	QuantityFrom int "json:quantityFrom"
	QuantityTo int "json:quantityTo"
	Material string "json:material"
}	

func main()  {
	engine := html.New("./views", ".html")

	app := fiber.New(fiber.Config{
        Views: engine,
    })

	app.Get("/", func(c *fiber.Ctx) error {
        // Render index template
        return c.Render("index", fiber.Map{
            "Title": "Hello, World!",
        })
    })

	app.Get("/getQuotation", func(c *fiber.Ctx) error {
		// Connect to Google Sheets
		quotation := new(Quotation)
		if err := c.BodyParser(quotation); err != nil {
            return err
        }
		return c.JSON(quotation)
	})

    log.Fatal(app.Listen(":80"))
	
}