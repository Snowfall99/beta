package main

import (
	"flag"
	"log"

	bls "themix.new.io/crypto/themixBLS"
)

func main() {
	n := flag.Int("n", 4, "number of nodes")
	th := flag.Int("t", 2, "number of shares")
	flag.Parse()
	// Crypto setup
	err := bls.GenerateBlsKey("./", *n, *th)
	if err != nil {
		log.Fatal("bls.GenerateBlsKey: ", err)
	}
	_, _, err = bls.LoadBlsKey("./", *n, *th)
	if err != nil {
		log.Fatal("bls.LoadBlsKey: ", err)
	}
}
