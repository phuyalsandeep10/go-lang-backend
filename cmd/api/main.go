package main

// @title HomeInsight Properties API
// @version 1.0
// @description A comprehensive property management API for real estate data
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8000
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	cfg := LoadConfiguration()
	app := NewApp(cfg)
	defer app.cleanup()
	app.InitializeServer()
	app.StartServer()
}
