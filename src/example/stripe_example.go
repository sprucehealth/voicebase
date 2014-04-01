package main

import "carefront/libs/payment/stripe"

func main() {

	stripeService := &stripe.StripeService{
		SecretKey: "sk_test_zDj7fkTWgftpInZv5enbGa0B",
	}

	// customer, err := stripeService.CreateCustomerWithDefaultCard("tok_103a2I2OcgXOgdiKuX0rYZXm")
	// if err != nil {
	// 	panic(err.Error())
	// }

	err := stripeService.DeleteCardForCustomer("cus_3bcTHTsMOnrVTM", "card_103bcT2OcgXOgdiKEMJLhO37")

	if err != nil {
		panic(err)
	}
}
