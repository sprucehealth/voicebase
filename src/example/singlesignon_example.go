package main

import (
	"carefront/libs/erx"
	"fmt"
)

func main() {
	singleSignOn := erx.GenerateSingleSignOn()
	fmt.Println("SingleSignOnCode: " + singleSignOn.Code)
	fmt.Println("SingleSignOnUserIdVerify: " + singleSignOn.UserIdVerify)
}
