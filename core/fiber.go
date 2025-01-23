package core

import (
	"github.com/gofiber/fiber/v2"
)

func SetupFiberApp() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "lfg",
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true, "data": nil})
	})

	return app
}

func ShutdownFiberApp(app *fiber.App) {
	_ = app.Shutdown()
}
