package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	var (
		addrListFile  string
		scanNeighbors bool
	)

	flag.StringVar(&addrListFile, "f", "", "ip or fqdn list file")
	flag.BoolVar(&scanNeighbors, "s", false, "scan neighbors. If the -f flag is set, it is ignored")
	flag.Parse()


	var input io.Reader
	if addrListFile != "" && addrListFile != "-" {
		if f, err := os.Open(addrListFile); err != nil {
			log.Fatal(err)
		} else {
			input = f
		}
	} else if !scanNeighbors {

	}
	log.Println("addrListFile:", addrListFile)
	log.Println("scanNeighbors:", scanNeighbors)
}

type 
