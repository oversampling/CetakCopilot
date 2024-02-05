package main

import (
	"log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)


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


    log.Fatal(app.Listen(":80"))
	
}