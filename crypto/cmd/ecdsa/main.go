package main

import (
	"fmt"
	"log"

	"themix.new.io/crypto/sha256"
	ecdsa "themix.new.io/crypto/themixECDSA"
)

func main() {
	_, err := ecdsa.GenerateEcdsaKey("./")
	if err != nil {
		log.Fatal("ecdsa.GenerateEcdsaKey: ", err)
	}
	priv, _ := ecdsa.LoadKey("./")
	hash, err := sha256.ComputeHash([]byte{'a', 'b', 'c'})
	if err != nil {
		log.Fatal("sha256.ComputeHash: ", err)
	}
	sig, err := ecdsa.SignECDSA(priv, hash)
	if err != nil {
		log.Fatal("ecdsa.SignECDSA: ", err)
	}
	b, err := ecdsa.VerifyECDSA(&priv.PublicKey, sig, hash)
	if err != nil {
		log.Fatal("ecdsa.VerifyECDSA: ", err)
	}
	if !b {
		fmt.Println("Message verification failed")
	} else {
		fmt.Println("Message verification succeeded")
	}
}
