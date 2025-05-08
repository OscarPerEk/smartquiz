module smartquiz

go 1.23.0

toolchain go1.24.3

// uncomment for local development on the superkit core.
// replace github.com/anthdm/superkit => ../

require (
	github.com/a-h/templ v0.3.865
	github.com/anthdm/superkit v0.0.0-20240701091803-e7f8e0aad3e9
	github.com/go-chi/chi/v5 v5.2.1
	github.com/go-resty/resty/v2 v2.16.5
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/mattn/go-sqlite3 v1.14.28
	golang.org/x/crypto v0.38.0
	gorm.io/driver/sqlite v1.5.7
	gorm.io/gorm v1.26.1
)

require (
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/gorilla/sessions v1.4.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/text v0.25.0 // indirect
)
