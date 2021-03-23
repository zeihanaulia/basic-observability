package main

import (
	"net/http"
)

func (s *Service) index(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("welcome to order service"))
}
