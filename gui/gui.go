package gui

import (
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/gobuffalo/packr"
	"net/http"
)

func Run(addr string) {
	box := packr.NewBox("./html")

	http.Handle("/", http.FileServer(box))
	log.Error(http.ListenAndServe(addr, nil).Error())
}
