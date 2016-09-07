package main

import "vimeoserver/server"

func main() {
	service := server.NewVimeoService()
	service.HTTPServer.ListenAndServe()
}
