package main

import (
	"carefront/libs/erx"
	"fmt"
)

func main() {
	singleSignOn := erx.GenerateSingleSignOn()
	fmt.Println("SingleSignOnCode: " + singleSignOn.SingleSignOnCode)
	fmt.Println("SingleSignOnUserIdVerify: " + singleSignOn.SingleSignOnUserIdVerify)
}
