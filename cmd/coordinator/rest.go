package main

import (
	"github.com/gorilla/mux"
	"net/http"
	"fmt"
	"log"
	"io/ioutil"
	"bytes"
	"database/sql"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if html, err := ioutil.ReadFile("./graphplotter/index.html"); err != nil {
		log.Fatal(err)
	} else {
		var buffer bytes.Buffer

		buffer.WriteString( "<script type=\"text/javascript\">")
		buffer.WriteString("var data3=[1, 8, 10, 16, 25, 30];")
		buffer.WriteString("</script>")
		buffer.Write(html)

		w.Write(buffer.Bytes())
	}
}

func startRestServer(port int, db *sql.DB) {
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	http.Handle("/", r)

	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprint("127.0.0.1:", port),
	}
	log.Fatal(srv.ListenAndServe())
}