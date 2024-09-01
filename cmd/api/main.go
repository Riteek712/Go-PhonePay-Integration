package main

import (
	"fmt"
	"phonepe-api/internal/server"
)

const (
	//Prod
	PhonePeURL = "https://api.phonepe.com/apis/hermes"
	MerchantId = "M222GU0OFKMJU"
	SaltIndex  = "1"
	SaltKey    = "96434309-7796-489d-8924-ab56988a6076"
)

func main() {

	server := server.NewServer()

	err := server.ListenAndServe()
	if err != nil {
		panic(fmt.Sprintf("cannot start server: %s", err))
	}
}
