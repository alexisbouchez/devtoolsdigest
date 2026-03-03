package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	mux.HandleFunc("GET /", handleIndex)
	mux.HandleFunc("POST /digest", handleCreateDigest)
	mux.HandleFunc("GET /digest", handleEditDigest)
	mux.HandleFunc("POST /digest/save", handleSaveDigest)
	mux.HandleFunc("GET /digests", handleListDigests)
	mux.HandleFunc("GET /digests/{date}", handleViewDigest)

	log.Println("listening on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", mux))
}
