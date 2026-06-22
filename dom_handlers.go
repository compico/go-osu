package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

var doms = "./static/doms/"

func domInterfaceHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	t, err := template.ParseFiles(doms + "interface.html")

	if err != nil {
		fmt.Fprintf(w, "Error \"ParseFiles\": %v\n", err.Error())
	}

	if err := t.Execute(w, fh.BeatmapsDirectories); err != nil {
		fmt.Fprintf(w, "Error \"Execute\": %v\n", err.Error())
	}
}
