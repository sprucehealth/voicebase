package main

import (
	"carefront/libs/address_validation"
	"fmt"
)

func main() {

	authId := "84a88b2b-acc0-402b-8b67-c6af09e6080a"
	authToken := "M9rSnFoqXJmI71YaKAU2D3DpVAjJhBXDTPjWSUwdGl2iPkkDFIdTJrTrCBrrv1WufIeDJNkC68rP/Qo8iKOUng=="

	smartyStreetsService := address_validation.SmartyStreetsService{
		AuthId:    authId,
		AuthToken: authToken,
	}

	cityStateInfo, err := smartyStreetsService.ZipcodeLookup("98366")
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v", cityStateInfo)

}
