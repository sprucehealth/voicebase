package main

import (
	"carefront/libs/payment/stripe"
	"fmt"
)

func main() {

	stripeService := &stripe.StripeService{
		SecretKey: "sk_test_zDj7fkTWgftpInZv5enbGa0B",
	}

	customer, err := stripeService.CreateCustomerWithDefaultCard("tok_103a2I2OcgXOgdiKuX0rYZXm")
	if err != nil {
		panic(err.Error())
	}

	cards, err := stripeService.GetCardsForCustomer(customer.Id)

	for _, card := range cards {
		fmt.Println("CustomerId is " + customer.Id)
		fmt.Println("CardId is " + card.ThirdPartyId)
		fmt.Println("CardFingerprint is " + card.Fingerprint)
		fmt.Println("CardType is " + card.Type)
		fmt.Printf("CardExpYear is %d and ExpMonth is %d and Last4 are %d\n", card.ExpYear, card.ExpMonth, card.Last4)
	}
}
