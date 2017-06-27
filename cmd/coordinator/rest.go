package main

import (
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if html, err := ioutil.ReadFile("./graphplotter/index.html"); err != nil {
		log.Fatal(err)
	} else {
		var buffer bytes.Buffer
		jsonStr, err := coordinator.getFullCaptureFromDb()
		if err != nil {
			log.Fatal(err)
		}

		buffer.WriteString("<script type=\"text/javascript\">")
		buffer.WriteString("var data=")
		buffer.WriteString(jsonStr)
		buffer.WriteString(";")
		buffer.WriteString("var yMax=")
		buffer.WriteString(strconv.FormatInt(coordinator.GetMaxLatency(), 10))
		buffer.WriteString(";")
		buffer.WriteString("</script>")
		buffer.Write(html)

		w.Write(buffer.Bytes())
	}
}

func startRestServer(port int) {
	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprint("127.0.0.1:", port),
	}
	log.Fatal(srv.ListenAndServe())
}
