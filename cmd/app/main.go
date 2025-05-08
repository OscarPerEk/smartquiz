package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"smartquiz/app"
	"smartquiz/public"

	"github.com/anthdm/superkit/kit"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("lets see if i get this")
	kit.Setup()
	router := chi.NewMux()

	app.InitializeMiddleware(router)

	if kit.IsDevelopment() {
		router.Handle("/public/*", disableCache(staticDev()))
	} else if kit.IsProduction() {
		router.Handle("/public/*", staticProd())
	}

	kit.UseErrorHandler(app.ErrorHandler)
	router.HandleFunc("/*", kit.Handler(app.NotFoundHandler))

	app.InitializeRoutes(router)
	app.RegisterEvents()

	listenAddr := os.Getenv("HTTP_LISTEN_ADDR")
	// In development link the full Templ proxy url.
	url := "http://localhost:7331"
	url = "localhost:7331"
	url = fmt.Sprintf("localhost%v", listenAddr)
	if kit.IsProduction() {
		url = fmt.Sprintf("http://0.0.0.0%s", listenAddr)
		fmt.Println("kit is in production, using: ", url)
		// url = fmt.Sprintf("http://localhost%s", listenAddr)
	}

	fmt.Printf("application running in %s at %s\n", kit.Env(), url)

	err := http.ListenAndServe(url, router)
	if err != nil {
		log.Fatalf("servering fialed: %v", err)
	}

}

func staticDev() http.Handler {
	return http.StripPrefix("/public/", http.FileServerFS(os.DirFS("public")))
}

func staticProd() http.Handler {
	return http.StripPrefix("/public/", http.FileServerFS(public.AssetsFS))
}

func disableCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func init() {
	fmt.Println("insidie init of main")
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("loaded env variables: %s, %s, %s", os.Getenv("DB_DRIVER"), os.Getenv("DB_NAME"), os.Getenv("DB_PASSWORD"))
}
